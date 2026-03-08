package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const maxLogLines = 500

type Service struct {
	Config     ServiceConfig
	Status     string // "stopped", "running", "stopping"
	Logs       []string
	baseDir    string
	logMaxSize int64
	mu         sync.Mutex
	opMu       sync.Mutex // serializes Start/Stop operations
	tailDone   chan struct{}
}

func NewService(cfg ServiceConfig, baseDir string, logMaxSize int64) *Service {
	s := &Service{
		Config:     cfg,
		Status:     "stopped",
		Logs:       make([]string, 0),
		baseDir:    baseDir,
		logMaxSize: logMaxSize,
	}
	s.detectStatus()
	s.loadRecentLogs()
	if s.Status == "running" {
		s.startTail()
	}
	return s
}

func (s *Service) pidFile() string {
	return filepath.Join(s.baseDir, "pids", s.Config.Name+".pid")
}

func (s *Service) logFile() string {
	return filepath.Join(s.baseDir, "logs", s.Config.Name+".log")
}

func (s *Service) detectStatus() {
	pid, err := s.readPid()
	if err != nil {
		s.Status = "stopped"
		return
	}
	if isProcessRunning(pid) {
		s.Status = "running"
	} else {
		s.Status = "stopped"
		os.Remove(s.pidFile())
	}
}

func (s *Service) loadRecentLogs() {
	f, err := os.Open(s.logFile())
	if err != nil {
		return
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > maxLogLines {
		lines = lines[len(lines)-maxLogLines:]
	}
	s.Logs = lines
}

func (s *Service) Start() error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	if s.Status == "running" {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	// Ensure dirs exist
	os.MkdirAll(filepath.Join(s.baseDir, "pids"), 0755)
	os.MkdirAll(filepath.Join(s.baseDir, "logs"), 0755)

	// Truncate log file if it exceeds max size
	s.truncateLogIfNeeded()

	// Open log file for append
	logF, err := os.OpenFile(s.logFile(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	cmd := exec.Command("bash", "-c", s.Config.Cmd)
	cmd.Dir = s.Config.Dir
	cmd.Stdout = logF
	cmd.Stderr = logF
	// Start in new process group so it survives TUI exit
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		logF.Close()
		s.mu.Lock()
		s.appendLog("[ksv] failed to start: " + err.Error())
		s.mu.Unlock()
		return err
	}

	// Write PID
	os.WriteFile(s.pidFile(), []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	// Close log file in parent — child has its own fd
	logF.Close()

	// Release the process so it's not waited on by us
	cmd.Process.Release()

	s.mu.Lock()
	s.Status = "running"
	s.appendLog("[ksv] started (pid " + strconv.Itoa(cmd.Process.Pid) + ")")
	s.startTail()
	s.mu.Unlock()

	return nil
}

func (s *Service) Stop() {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	if s.Status != "running" {
		s.mu.Unlock()
		return
	}
	pid, err := s.readPid()
	s.Status = "stopping"
	s.mu.Unlock()

	if err != nil {
		s.mu.Lock()
		s.Status = "stopped"
		s.mu.Unlock()
		return
	}

	// Kill the process group to also kill children
	syscall.Kill(-pid, syscall.SIGTERM)

	// Wait briefly for graceful shutdown, then force kill
	for i := 0; i < 50; i++ {
		if !isProcessRunning(pid) {
			break
		}
		sleepMs(100)
	}
	if isProcessRunning(pid) {
		syscall.Kill(-pid, syscall.SIGKILL)
	}

	s.mu.Lock()
	s.Status = "stopped"
	s.appendLog("[ksv] stopped")
	os.Remove(s.pidFile())
	s.stopTail()
	s.mu.Unlock()
}

func (s *Service) RefreshStatus() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Don't interfere while Stop() is actively killing the process
	if s.Status == "stopping" {
		return
	}

	pid, err := s.readPid()
	if err != nil {
		if s.Status == "running" {
			s.Status = "stopped"
			s.appendLog("[ksv] process exited")
			s.stopTail()
		}
		return
	}
	if !isProcessRunning(pid) {
		s.Status = "stopped"
		s.appendLog("[ksv] process exited")
		os.Remove(s.pidFile())
		s.stopTail()
	}
}

func (s *Service) startTail() {
	if s.tailDone != nil {
		return
	}
	s.tailDone = make(chan struct{})
	go s.tailLog()
}

func (s *Service) stopTail() {
	if s.tailDone != nil {
		close(s.tailDone)
		s.tailDone = nil
	}
}

func (s *Service) tailLog() {
	f, err := os.Open(s.logFile())
	if err != nil {
		return
	}
	defer f.Close()

	// Seek to end
	f.Seek(0, io.SeekEnd)

	reader := bufio.NewReader(f)
	for {
		select {
		case <-s.tailDone:
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			sleepMs(200)
			continue
		}
		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			continue
		}
		s.mu.Lock()
		s.appendLog(line)
		s.mu.Unlock()
	}
}

func (s *Service) appendLog(line string) {
	s.Logs = append(s.Logs, line)
	if len(s.Logs) > maxLogLines {
		s.Logs = s.Logs[len(s.Logs)-maxLogLines:]
	}
}

func (s *Service) GetLogs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.Logs))
	copy(out, s.Logs)
	return out
}

func (s *Service) GetStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status
}

func (s *Service) readPid() (int, error) {
	data, err := os.ReadFile(s.pidFile())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func (s *Service) truncateLogIfNeeded() {
	if s.logMaxSize <= 0 {
		return
	}
	info, err := os.Stat(s.logFile())
	if err != nil || info.Size() <= s.logMaxSize {
		return
	}

	// Keep the last half of max size
	keepBytes := s.logMaxSize / 2

	f, err := os.Open(s.logFile())
	if err != nil {
		return
	}
	f.Seek(-keepBytes, io.SeekEnd)
	// Skip partial first line
	r := bufio.NewReader(f)
	r.ReadString('\n')
	tail, err := io.ReadAll(r)
	f.Close()
	if err != nil {
		return
	}

	os.WriteFile(s.logFile(), tail, 0644)
}

func isProcessRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

func sleepMs(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
