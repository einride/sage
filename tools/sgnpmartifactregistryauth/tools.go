package sgnpmartifactregistryauth

import (
	"context"
	"fmt"
	"os/exec"

	"go.einride.tech/sage/sg"
)

const (
	name    = "npm-artifact-registry-auth"
	version = "3.1.2"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	sg.Logger(ctx).Println("authenticating npm to artifact registry...")
	return sg.Command(
		ctx,
		"npx",
		fmt.Sprintf("google-artifactregistry-auth@%s", version),
	).Run()
}
