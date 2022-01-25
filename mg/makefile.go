package mg

import (
	"go/doc"
	"path/filepath"

	"go.einride.tech/mage-tools/internal/codegen"
)

func generateMakefile(g *codegen.File, pkg *doc.Package, mk *makefile) error {
	includePath, err := filepath.Rel(filepath.Dir(mk.Path), FromGitRoot(MageDir))
	if err != nil {
		return err
	}
	g.P("# To learn more, see .mage/magefile.go and https://github.com/einride/mage-tools.")
	g.P()
	if len(mk.DefaultTarget) != 0 {
		g.P(".DEFAULT_GOAL := ", mk.DefaultTarget)
		g.P()
	}
	g.P("magefile := ", filepath.Join(includePath, ToolsDir, MagefileBinary))
	return nil
}
