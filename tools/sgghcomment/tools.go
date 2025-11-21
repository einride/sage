package sgghcomment

import (
	"context"
	"os/exec"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const (
	version    = "226e3abf52137d2af2f5d92a03402fa1839add43"
	binaryName = "ghcomment"
)

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(binaryName), args...)
}

func PrepareCommand(ctx context.Context) error {
	_, err := sgtool.GoInstall(ctx, "github.com/einride/ghcomment", version)
	return err
	// binDir := sg.FromToolsDir(binaryName, version)
	// binary := filepath.Join(binDir, binaryName)
	// if err := sgtool.FromRemote(
	// 	ctx,
	// 	fmt.Sprintf(
	// 		"https://github.com/einride/ghcomment/releases/download/v%s/ghcomment_%s_%s_%s.tar.gz",
	// 		version,
	// 		version,
	// 		runtime.GOOS,
	// 		runtime.GOARCH,
	// 	),
	// 	sgtool.WithDestinationDir(binDir),
	// 	sgtool.WithUntarGz(),
	// 	sgtool.WithSkipIfFileExists(binary),
	// 	sgtool.WithSymlink(binary),
	// ); err != nil {
	// 	return fmt.Errorf("unable to download %s: %w", binaryName, err)
	// }
	// if err := os.Chmod(binary, 0o755); err != nil {
	// 	return fmt.Errorf("unable to make %s command: %w", binaryName, err)
	// }
	// return nil
}
