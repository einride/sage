Mage-tools
==========

Mage-tools is an opinionated set of [mage](https://github.com/magefile/mage) targets to help with build automation of different projects

[![Release](https://github.com/einride/mage-tools/actions/workflows/release.yml/badge.svg)](https://github.com/einride/mage-tools/actions/workflows/release.yml)

Requirements
------------

-	[Go](https://golang.org/doc/install) >= 1.17
-	[GNU Make](https://www.gnu.org/software/make/)

Getting started
---------------

To initilize mage-tools in a repository, just run:

```bash
go run go.einride.tech/mage-tools@latest init
```

Run `make`

Usage
-----

Mage imports, and targets within the magefiles, can be written to Makefiles, you can generate as many Makefiles as you want, see more at [Makefiles / Mage namespaces](https://github.com/einride/mage-tools#makefiles--mage-namespaces).

### Magefiles

You can have as many magefiles as you want in the `.mage` folder, as long as you tag them and put them in the main package

```golang
// +build mage

package main
```

#### Targets

Targets gets generated from expored functions in the main package, they can have no return value but error. Arguments are supported but limited to optional first argument of context.Context, string, int or bool.

```golang
func All() {
  return
}

func FormatYaml() error {
	return mgyamlfmt.FormatYAML()
}

func ConvcoCheck(ctx context.Context, rev string) error {
	mglog.Logger("convco-check").Info("checking...")
	return mgconvco.Command(ctx, "check", rev).Run()
}
```

#### Makefiles / Mage namespaces

To generate makefiles, an `init` method needs to exist in one of the magefiles where we call the `mgmake.GenerateMakefiles` method.

```golang
func init() {
	mgmake.GenerateMakefiles(
		mgmake.Makefile{
			Path:          mgpath.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}
```

If another makefile is desired, lets say one that only includes Terraform targets, we utilize the `mg.Namespace` type and just add another `Makefile` to the `GenerateMakefiles` method and specify the namespace, path and default target.

```golang

func init() {
	mgmake.GenerateMakefiles(
		mgmake.Makefile{
			Path:          mgpath.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
		mgmake.Makefile{
			Path:      mgpath.FromGitRoot("terraform/Makefile"),
			Namespace: Terraform{},
		},
	)
}

type Terraform mg.Namespace

func (Terraform) TerraformInitDev() {
	mg.SerialDeps(
		Terraform.devConfig,
		mgterraform.Init,
	)
}
```

#### Dependencies

Dependencies can be defined just by specificing the function, or with `mg.F` if the function takes arguments. `Deps` runs in parallel while `Serial` runs serially

```golang
mg.Deps(
	mg.F(mgcommitlint.Commitlint, "main"),
	mggolangcilint.GolangciLint,
	mggoreview.Goreview,
)
mg.SerialDeps(
	mggo.GoModTidy,
	mggitverifynodiff.GitVerifyNoDiff,
)
```
