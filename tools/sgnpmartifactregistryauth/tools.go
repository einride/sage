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

func Command(ctx context.Context) *exec.Cmd {
	sg.Logger(ctx).Println("authenticating npm to artifact registry...")
	return sg.Command(
		ctx,
		"npx",
		fmt.Sprintf("google-artifactregistry-auth@%s", version),
	)
}
