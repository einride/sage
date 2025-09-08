package sgpostgres

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

const (
	imageName         = "postgres"
	version           = "14"
	image             = imageName + ":" + version
	pgEnvVariableName = "POSTGRES_URL"
	dbUser            = "postgres"
)

// RunLocal runs a postgres instance in Docker on the local host.
//
// Primary goal is to have a shared local instance for test runs. Heavily inspired by
// Spanner emulator, tools/sgcloudspanner/emulator.go .
func RunLocal(
	ctx context.Context,
	databaseName string,
	databasePassword string,
) (_ func(), err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("run Postgres local instance: %w", err)
		}
	}()
	sg.Logger(ctx).Println("starting Postgres local instance ...")
	if localHost, ok := os.LookupEnv(pgEnvVariableName); ok {
		sg.Logger(ctx).Printf("a Postgres local instance is already running on %s", localHost)
		return func() {}, nil
	}
	if !isDockerDaemonRunning(ctx) {
		return nil, fmt.Errorf("the Docker daemon does not seem to be running")
	}

	if databaseName == "" {
		return nil, fmt.Errorf("databaseName is empty")
	}
	if databasePassword == "" {
		return nil, fmt.Errorf("databasePassword is empty")
	}

	err = sgdocker.Command(ctx, "pull", image).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to pull docker image %s: %w", image, err)
	}
	dockerRunCmd := sgdocker.Command(
		ctx,
		"run",
		"-d",
		"--publish-all",
		"-e",
		fmt.Sprintf("POSTGRES_PASSWORD=%s", databasePassword),
		"-e",
		fmt.Sprintf("POSTGRES_DB=%s", databaseName),
		image,
	)

	var dockerRunStdout strings.Builder
	dockerRunCmd.Stdout = &dockerRunStdout
	if err := dockerRunCmd.Run(); err != nil {
		return nil, err
	}
	containerID := strings.TrimSpace(dockerRunStdout.String())
	cleanup := func() {
		sg.Logger(ctx).Println("stopping down Postgres local instance ...")
		cmd := sgdocker.Command(ctx, "kill", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			sg.Logger(ctx).Printf("failed to kill postgres container: %v", err)
		}
		cmd = sgdocker.Command(ctx, "rm", "-v", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			sg.Logger(ctx).Printf("failed to remove postgres container: %v", err)
		}
		if err := os.Unsetenv(pgEnvVariableName); err != nil {
			sg.Logger(ctx).Printf("failed to unset %s: %v", pgEnvVariableName, err)
		}
	}
	pgHostPort, err := inspectPortAddress(ctx, containerID, "5432/tcp")
	if err != nil {
		cleanup()
		return nil, err
	}
	hostPort := strings.Split(pgHostPort, ":")
	if len(hostPort) != 2 {
		cleanup()
		return nil, fmt.Errorf("unexpected host port combination: %s", pgHostPort)
	}
	host := hostPort[0]
	port := hostPort[1]

	dbURL := databaseURL(dbUser, databasePassword, host, port, databaseName)

	if err := os.Setenv(pgEnvVariableName, dbURL); err != nil {
		cleanup()
		return nil, err
	}
	sg.Logger(ctx).Printf("running Postgres on host: %s , port: %s", host, port)
	if err := awaitReachable(ctx, pgHostPort, 100*time.Millisecond, 10*time.Second); err != nil {
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
	lines := strings.Split(output, "\n")
	// docker port can return ipv6 mapping as well, take the first non ipv6 mapping.
	for _, line := range lines {
		mapping := strings.TrimSpace(line)
		if _, err := net.ResolveTCPAddr("tcp4", mapping); err == nil {
			sg.Logger(ctx).Printf("mapping: %s", mapping)

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

func databaseURL(dbUser, dbPassword, host, port, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, host, port, dbName)
}
