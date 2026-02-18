package sgcloudspanner

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgdocker"
)

// Cloud Spanner Emulator versions can be found here,
// https://console.cloud.google.com/gcr/images/cloud-spanner-emulator/global/emulator
const (
	url     = "gcr.io/cloud-spanner-emulator/emulator"
	version = "1.5.49"
	image   = url + ":" + version
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
	if !isDockerDaemonRunning(ctx) {
		return nil, fmt.Errorf("the Docker daemon does not seem to be running")
	}
	dockerRunCmd := sgdocker.Command(ctx, "run", "-d", "--publish-all", image)
	var dockerRunStdout strings.Builder
	dockerRunCmd.Stdout = &dockerRunStdout
	if err := dockerRunCmd.Run(); err != nil {
		return nil, err
	}
	containerID := strings.TrimSpace(dockerRunStdout.String())
	cleanup := func() {
		// TODO(radhus): when bumping to Go >=1.21, we should define a context like this instead and avoid storing a logger.
		// ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		logger := sg.Logger(ctx)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Println("stopping down Cloud Spanner emulator...")
		cmd := sgdocker.Command(ctx, "kill", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			logger.Printf("failed to kill emulator container: %v", err)
		}
		cmd = sgdocker.Command(ctx, "rm", "-v", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			logger.Printf("failed to remove emulator container: %v", err)
		}
		if err := os.Unsetenv("SPANNER_EMULATOR_HOST"); err != nil {
			logger.Printf("failed to unset SPANNER_EMULATOR_HOST: %v", err)
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

func isDockerDaemonRunning(ctx context.Context) bool {
	cmd := sgdocker.Command(ctx, "info")
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Run() == nil
}

func inspectPortAddress(ctx context.Context, containerID, containerPort string) (string, error) {
	var stdout bytes.Buffer
	cmd := sgdocker.Command(ctx, "port", containerID, containerPort)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	output := stdout.String()
	lines := strings.SplitSeq(output, "\n")
	// docker port can return ipv6 mapping as well, take the first non ipv6 mapping.
	for line := range lines {
		mapping := strings.TrimSpace(line)
		if _, err := net.ResolveTCPAddr("tcp4", mapping); err == nil {
			return mapping, nil
		}
	}
	return "", fmt.Errorf("no mapping found for %s in container %s", containerPort, containerID)
}

func awaitReachable(ctx context.Context, addr string, wait, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	dialer := net.Dialer{Deadline: deadline}
	for time.Now().Before(deadline) {
		if c, err := dialer.DialContext(ctx, "tcp", addr); err == nil {
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
