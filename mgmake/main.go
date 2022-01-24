package mgmake

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"go.einride.tech/mage-tools/mg"
)

type mainfileTemplateData struct {
	Funcs []*mg.Function
}

// compile uses the go tool to compile the files into an executable at path.
func compile(magePath, compileTo string, gofiles []string) error {
	// strip off the path since we're setting the path in the build command
	for i := range gofiles {
		gofiles[i] = filepath.Base(gofiles[i])
	}
	// nolint: gosec
	c := exec.Command("go", append([]string{"build", "-o", compileTo}, gofiles...)...)
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	c.Dir = magePath
	if err := c.Run(); err != nil {
		return fmt.Errorf("error compiling magefiles: %w", err)
	}
	return nil
}

// generateMainFile generates the mage mainfile at path.
func generateMainFile(path string, info *mg.PkgInfo) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating generated mainfile: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()
	data := mainfileTemplateData{
		Funcs: info.Funcs,
	}
	if err := template.Must(template.New("").Parse(mageMainfileTplString)).Execute(f, data); err != nil {
		return fmt.Errorf("can't execute mainfile template: %v", err)
	}
	return nil
}
