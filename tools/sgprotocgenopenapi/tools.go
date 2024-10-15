package sgprotocgenopenapi

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

// docs:
// https://github.com/google/gnostic/tree/main/cmd/protoc-gen-openapi
// https://buf.build/gnostic/gnostic/docs/main:gnostic.openapi.v3

const (
	version = "0.7.0"
	name    = "protoc-gen-openapi"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "github.com/google/gnostic/cmd/"+name, "v"+version)
	return err
}
