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
)

// RunEmulator runs the Cloud Spanner emulator in Docker.
func RunEmulator(ctx context.Context) (_ func(), err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("run Cloud Spanner emulator: %w", err)
		}
	}()
	sg.Logger(ctx).Println("starting Cloud Spanner emulator...")
	if emulatorHost, ok := os.LookupEnv("SPANNER_EMULATOR_HOST"); ok {
		sg.Logger(ctx).Printf("a Cloud Spanner emulator is already running on %s", emulatorHost)
		return func() {}, nil
	}
	if !hasDocker() {
		return nil, fmt.Errorf("no Docker client available for running the Cloud Spanner emulator container")
	}
	if !isDockerDaemonRunning() {
		return nil, fmt.Errorf("the Docker daemon does not seem to be running")
	}
	const image = "gcr.io/cloud-spanner-emulator/emulator:latest"
	cmd := sg.Command(ctx, "docker", "pull", image)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var dockerRunCmd *exec.Cmd
	if isRunningOnCloudBuild(ctx) {
		dockerRunCmd = sg.Command(ctx, "docker", "run", "-d", "--network", "cloudbuild", "--publish-all", image)
	} else {
		dockerRunCmd = sg.Command(ctx, "docker", "run", "-d", "--publish-all", image)
	}
	var dockerRunStdout strings.Builder
	dockerRunCmd.Stdout = &dockerRunStdout
	if err := dockerRunCmd.Run(); err != nil {
		return nil, err
	}
	containerID := strings.TrimSpace(dockerRunStdout.String())
	cleanup := func() {
		sg.Logger(ctx).Println("stopping down Cloud Spanner emulator...")
		cmd := sg.Command(ctx, "docker", "kill", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			sg.Logger(ctx).Printf("failed to kill emulator container: %v", err)
		}
		cmd = sg.Command(ctx, "docker", "rm", "-v", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			sg.Logger(ctx).Printf("failed to remove emulator container: %v", err)
		}
		if err := os.Unsetenv("SPANNER_EMULATOR_HOST"); err != nil {
			sg.Logger(ctx).Printf("failed to unset SPANNER_EMULATOR_HOST: %v", err)
		}
	}
	emulatorHost, err := inspectPortAddress(ctx, containerID, "9010/tcp")
	if err != nil {
		cleanup()
		return nil, err
	}
	if err := os.Setenv("SPANNER_EMULATOR_HOST", emulatorHost); err != nil {
		cleanup()
		return nil, err
	}
	sg.Logger(ctx).Printf("running Cloud Spanner emulator on %s", emulatorHost)
	if err := awaitReachable(ctx, emulatorHost, 100*time.Millisecond, 10*time.Second); err != nil {
		cleanup()
		return nil, err
	}
	return cleanup, nil
}

func hasDocker() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func isDockerDaemonRunning() bool {
	return exec.Command("docker", "info").Run() == nil
}

func inspectPortAddress(ctx context.Context, containerID, containerPort string) (string, error) {
	var stdout bytes.Buffer
	cmd := sg.Command(ctx, "docker", "inspect", containerID)
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
	cmd := sg.Command(ctx, "docker", "network", "inspect", "cloudbuild")
	var stdout bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, nil
	return cmd.Run() == nil
}
