package sgcloudspanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgdocker"
)

type EmulatorContainer struct {
	Context      context.Context
	ContainerID  string
	DatabaseHost string
	AdminHost    string
}

// RunEmulator runs the Cloud Spanner emulator in Docker.
func RunEmulator(ctx context.Context) (_ func(), err error) {
	emulator, err := RunEmulatorContainer(ctx)
	if err != nil {
		return nil, err
	}
	if emulator.ContainerID == "" {
		return emulator.Close, nil
	}
	if err := os.Setenv("SPANNER_EMULATOR_HOST", emulator.DatabaseHost); err != nil {
		emulator.Close()
		return nil, err
	}
	cleanup := func() {
		emulator.Close()
		if err := os.Unsetenv("SPANNER_EMULATOR_HOST"); err != nil {
			sg.Logger(ctx).Printf("failed to unset SPANNER_EMULATOR_HOST: %v", err)
		}
	}
	return cleanup, nil
}

func RunEmulatorContainer(ctx context.Context) (_ EmulatorContainer, err error) {
	emulator := EmulatorContainer{Context: ctx}
	defer func() {
		if err != nil {
			emulator.Close()
			err = fmt.Errorf("run Cloud Spanner emulator: %w", err)
		}
	}()
	sg.Logger(ctx).Println("starting Cloud Spanner emulator...")
	if emulatorHost, ok := os.LookupEnv("SPANNER_EMULATOR_HOST"); ok {
		sg.Logger(ctx).Printf("a Cloud Spanner emulator is already running on %s", emulatorHost)
		return emulator, nil
	}
	if !isDockerDaemonRunning(ctx) {
		return emulator, fmt.Errorf("the Docker daemon does not seem to be running")
	}
	const image = "gcr.io/cloud-spanner-emulator/emulator:latest"
	cmd := sgdocker.Command(ctx, "pull", image)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err != nil {
		return emulator, err
	}
	var dockerRunCmd *exec.Cmd
	if isRunningOnCloudBuild(ctx) {
		dockerRunCmd = sgdocker.Command(ctx, "run", "-d", "--network", "cloudbuild", "--publish-all", image)
	} else {
		dockerRunCmd = sgdocker.Command(ctx, "run", "-d", "--publish-all", image)
	}
	var dockerRunStdout strings.Builder
	dockerRunCmd.Stdout = &dockerRunStdout
	if err := dockerRunCmd.Run(); err != nil {
		return emulator, err
	}
	containerID := strings.TrimSpace(dockerRunStdout.String())
	emulator.ContainerID = containerID
	emulator.DatabaseHost, err = inspectPortAddress(ctx, containerID, "9010/tcp")
	if err != nil {
		return emulator, err
	}
	emulator.AdminHost, err = inspectPortAddress(ctx, containerID, "9020/tcp")
	if err != nil {
		return emulator, err
	}
	sg.Logger(ctx).Printf("running Cloud Spanner emulator on %s", emulator.DatabaseHost)
	if err := awaitReachable(ctx, emulator.DatabaseHost, 100*time.Millisecond, 10*time.Second); err != nil {
		return emulator, err
	}
	return emulator, nil
}

func (e EmulatorContainer) Close() {
	if e.ContainerID == "" {
		return
	}
	sg.Logger(e.Context).Println("stopping down Cloud Spanner emulator...")
	cmd := sgdocker.Command(e.Context, "kill", e.ContainerID)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err != nil {
		sg.Logger(e.Context).Printf("failed to kill emulator container: %v", err)
	}
	cmd = sgdocker.Command(e.Context, "rm", "-v", e.ContainerID)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err != nil {
		sg.Logger(e.Context).Printf("failed to remove emulator container: %v", err)
	}
}

func isDockerDaemonRunning(ctx context.Context) bool {
	cmd := sgdocker.Command(ctx, "info")
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Run() == nil
}

func inspectPortAddress(ctx context.Context, containerID, containerPort string) (string, error) {
	var stdout bytes.Buffer
	cmd := sgdocker.Command(ctx, "inspect", containerID)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	var containers []struct {
		NetworkSettings struct {
			Ports map[string][]struct {
				HostIP   string
				HostPort string
			}
			Networks map[string]struct {
				Gateway string
			}
		}
	}
	if err := json.NewDecoder(&stdout).Decode(&containers); err != nil {
		return "", err
	}
	var host string
	var port string
	for _, container := range containers {
		if hostPorts, ok := container.NetworkSettings.Ports[containerPort]; ok {
			for _, hostPort := range hostPorts {
				host, port = hostPort.HostIP, hostPort.HostPort
				break // prefer first option
			}
		}
		if network, ok := container.NetworkSettings.Networks["cloudbuild"]; ok {
			host = network.Gateway
		}
	}
	if host == "" || port == "" {
		return "", fmt.Errorf("failed to inspect container %s for port %s", containerID, containerPort)
	}
	return fmt.Sprintf("%s:%s", host, port), nil
}

func awaitReachable(ctx context.Context, addr string, wait, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if c, err := net.Dial("tcp", addr); err == nil {
			_ = c.Close()
			return nil
		}
		sg.Logger(ctx).Printf("waiting %v for %s to become reachable...", wait, addr)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("%s was unreachable for %v", addr, maxWait)
}

func isRunningOnCloudBuild(ctx context.Context) bool {
	cmd := sgdocker.Command(ctx, "network", "inspect", "cloudbuild")
	var stdout bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, nil
	return cmd.Run() == nil
}
