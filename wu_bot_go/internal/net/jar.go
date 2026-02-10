package net

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
)

// JARProcess manages the lifecycle of the Java JAR sidecar.
type JARProcess struct {
	jarPath string
	host    string
	port    int
	cmd     *exec.Cmd
	mu      sync.Mutex
	logFunc func(string)
}

// NewJARProcess creates a new JAR process manager.
func NewJARProcess(jarPath, host string, port int, logFunc func(string)) *JARProcess {
	return &JARProcess{
		jarPath: jarPath,
		host:    host,
		port:    port,
		logFunc: logFunc,
	}
}

// Start launches the JAR process. It blocks until the process exits or ctx is cancelled.
func (j *JARProcess) Start(ctx context.Context) error {
	j.mu.Lock()

	j.cmd = exec.CommandContext(ctx, "java", "-jar", j.jarPath, j.host, fmt.Sprintf("%d", j.port))

	stdout, err := j.cmd.StdoutPipe()
	if err != nil {
		j.mu.Unlock()
		return fmt.Errorf("jar stdout pipe: %w", err)
	}

	stderr, err := j.cmd.StderrPipe()
	if err != nil {
		j.mu.Unlock()
		return fmt.Errorf("jar stderr pipe: %w", err)
	}

	if err := j.cmd.Start(); err != nil {
		j.mu.Unlock()
		return fmt.Errorf("jar start: %w", err)
	}

	j.mu.Unlock()

	// Stream stdout/stderr to log function
	go j.streamLogs("JAR stdout", stdout)
	go j.streamLogs("JAR stderr", stderr)

	err = j.cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// Kill terminates the JAR process.
func (j *JARProcess) Kill() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.cmd != nil && j.cmd.Process != nil {
		if err := j.cmd.Process.Kill(); err != nil {
			log.Printf("[JAR] kill error: %v", err)
		}
	}
}

func (j *JARProcess) streamLogs(prefix string, r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 && j.logFunc != nil {
			j.logFunc(fmt.Sprintf("[%s] %s", prefix, string(buf[:n])))
		}
		if err != nil {
			return
		}
	}
}
