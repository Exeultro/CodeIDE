package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Sandbox struct {
	cli          *client.Client
	workingDir   string
	workingDirMu sync.RWMutex
}

func NewSandbox() (*Sandbox, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Sandbox{
		cli:        cli,
		workingDir: "/",
	}, nil
}

func (s *Sandbox) SetWorkingDir(dir string) {
	s.workingDirMu.Lock()
	defer s.workingDirMu.Unlock()
	s.workingDir = dir
}

func (s *Sandbox) GetWorkingDir() string {
	s.workingDirMu.RLock()
	defer s.workingDirMu.RUnlock()
	return s.workingDir
}

// CreateContainer создаёт контейнер с Python и возвращает его ID
func (s *Sandbox) CreateContainer(ctx context.Context, name string) (string, error) {
	resp, err := s.cli.ContainerCreate(ctx,
		&container.Config{
			Image: "python:3.11-slim",
			Cmd:   []string{"sleep", "infinity"},
			Tty:   true,
		},
		&container.HostConfig{},
		nil,
		nil,
		name,
	)
	if err != nil {
		return "", err
	}
	// Запускаем контейнер
	if err := s.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}
	return resp.ID, nil
}

// RunCode выполняет Python-код в контейнере
func (s *Sandbox) RunCode(ctx context.Context, containerID string, code string, language string) (string, error) {
	// Определяем имя файла и команду запуска
	var filename string
	var runCmd []string

	switch language {
	case "python", "py":
		filename = "/tmp/code.py"
		runCmd = []string{"python", filename}
	case "javascript", "js":
		filename = "/tmp/code.js"
		runCmd = []string{"node", filename}
	case "typescript", "ts":
		filename = "/tmp/code.ts"
		runCmd = []string{"npx", "ts-node", filename}
	case "go":
		filename = "/tmp/code.go"
		runCmd = []string{"go", "run", filename}
	case "bash", "sh":
		filename = "/tmp/code.sh"
		runCmd = []string{"bash", filename}
	default:
		filename = "/tmp/code.py"
		runCmd = []string{"python", filename}
	}

	// Записываем код в контейнер
	writeCmd := []string{"sh", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", filename, code)}
	execConfig := container.ExecOptions{
		Cmd:          writeCmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := s.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create write exec: %v", err)
	}

	attachResp, err := s.cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach write: %v", err)
	}
	attachResp.Close()

	// Ждем завершения записи
	for {
		inspect, err := s.cli.ContainerExecInspect(ctx, execID.ID)
		if err != nil {
			return "", err
		}
		if !inspect.Running {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Выполняем код
	execRunConfig := container.ExecOptions{
		Cmd:          runCmd,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/tmp",
	}

	execID2, err := s.cli.ContainerExecCreate(ctx, containerID, execRunConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create run exec: %v", err)
	}

	attachResp2, err := s.cli.ContainerExecAttach(ctx, execID2.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach run: %v", err)
	}
	defer attachResp2.Close()

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, attachResp2.Reader)
	if err != nil {
		return "", err
	}

	output := outBuf.String()
	if errBuf.Len() > 0 {
		errOutput := errBuf.String()
		if output != "" {
			output += "\n"
		}
		output += errOutput
	}

	if strings.TrimSpace(output) == "" {
		output = "(no output)\n"
	}

	return output, nil
}

// Exec выполняет произвольную команду в контейнере
func (s *Sandbox) Exec(ctx context.Context, containerID string, command string) (string, error) {
	// Обработка cd
	if strings.HasPrefix(command, "cd ") {
		newDir := strings.TrimPrefix(command, "cd ")
		newDir = strings.TrimSpace(newDir)

		// Проверяем существование директории
		checkCmd := []string{"sh", "-c", fmt.Sprintf("test -d %s && echo 'exists'", newDir)}
		execConfig := container.ExecOptions{
			Cmd:          checkCmd,
			AttachStdout: true,
			AttachStderr: true,
		}
		execID, err := s.cli.ContainerExecCreate(ctx, containerID, execConfig)
		if err != nil {
			return "", err
		}
		attachResp, err := s.cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
		if err != nil {
			return "", err
		}
		defer attachResp.Close()

		var outBuf bytes.Buffer
		_, err = stdcopy.StdCopy(&outBuf, &outBuf, attachResp.Reader)
		if err != nil {
			return "", err
		}

		if strings.Contains(outBuf.String(), "exists") {
			s.SetWorkingDir(newDir)
			return fmt.Sprintf("Changed directory to %s\n", newDir), nil
		}
		return fmt.Sprintf("Directory not found: %s\n", newDir), nil
	}

	// Для pwd
	if command == "pwd" {
		return s.GetWorkingDir() + "\n", nil
	}

	// Для остальных команд, запускаем в текущей директории
	execCmd := []string{"sh", "-c", fmt.Sprintf("cd %s && %s", s.GetWorkingDir(), command)}

	execConfig := container.ExecOptions{
		Cmd:          execCmd,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   s.GetWorkingDir(),
	}

	execID, err := s.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", err
	}

	attachResp, err := s.cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return "", err
	}
	defer attachResp.Close()

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, attachResp.Reader)
	if err != nil {
		return "", err
	}

	output := outBuf.String()
	if errBuf.Len() > 0 {
		output += errBuf.String()
	}

	return output, nil
}
