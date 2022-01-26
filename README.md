# Sage

Sage is a Make-like build tool inspired by [Mage](https://magefile.org/) that
provides a curated and maintained set of build [tools](./tools) for Go projects.

[![Release](https://github.com/einride/sage/actions/workflows/release.yml/badge.svg)](https://github.com/einride/sage/actions/workflows/release.yml)

## Requirements

- [Go](https://golang.org/doc/install) >= 1.17
- [GNU Make](https://www.gnu.org/software/make/)

## Getting started

To initilize Sage in a repository, just run:

```bash
go run go.einride.tech/sage/cmd/init@latest
```

Run `make`.

## Usage

Sage imports, and targets within the Sagefiles, can be written to Makefiles, you can generate as many Makefiles as you want, see more at [Makefiles / Sage namespaces](https://github.com/einride/sage#makefiles--sage-namespaces).

### Sagefiles

You can have as many Sagefiles as you want in the `.sage` folder.

#### Targets

Any public function in the main package will be exported. Functions can have no return value but error. The following arguments are supported: Optional first argument of context.Context, string, int or bool.

```golang
func All() {
  sg.Deps(
	  FormatYaml,
	  sg.Fn(ConvcoCheck, "origin/main..HEAD"),
  )
}

func FormatYaml() error {
	return sgyamlfmt.FormatYAML()
}

func ConvcoCheck(ctx context.Context, rev string) error {
	logr.FromContextOrDiscard(ctx).Info("checking...")
	return sgconvco.Command(ctx, "check", rev).Run()
}
```

#### Makefiles / Sage namespaces

To generate Makefiles, a `main` method needs to exist in one of the Sagefiles where we call the `sg.GenerateMakefiles` method.

```golang
func main() {
	sg.GenerateMakefiles(
		sg.Makefile{
			Path:          sg.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}
```

If another makefile is desired, lets say one that only includes Terraform targets, we utilize the `sg.Namespace` type and just add another `Makefile` to the `GenerateMakefiles` method and specify the namespace, path and default target.

```golang

func init() {
	sg.GenerateMakefiles(
		sg.Makefile{
			Path:          sg.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
		sg.Makefile{
			Path:      sg.FromGitRoot("terraform/Makefile"),
			Namespace: Terraform{},
		},
	)
}

type Terraform sg.Namespace

func (Terraform) TerraformInitDev() {
	sg.SerialDeps(
		Terraform.DevConfig,
		Terraform.Init,
	)
}
```

#### Dependencies

Dependencies can be defined just by specificing the function, or with `sg.Fn` if the function takes arguments. `Deps` runs in parallel while `SerialDeps` runs serially.

```golang
sg.Deps(
	sg.Fn(ConvcoCheck, "origin/main..HEAD"),
	GolangciLint,
	GoReview,
)
sg.SerialDeps(
	GoModTidy,
	GitVerifyNoDiff,
)
```
