# :herb: Sage

Sage is a Make-like build tool inspired by [Mage](https://magefile.org/) that
provides a curated and maintained set of build [tools](./tools) for Go projects.

For a detailed and AI-generated documentation on the architecture behind this
project, which you can chat with, see the
[DeepWiki documentation](https://deepwiki.com/einride/sage).

[![Release](https://github.com/einride/sage/actions/workflows/release.yml/badge.svg)](https://github.com/einride/sage/actions/workflows/release.yml)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/einride/sage)

## Requirements

- [Go](https://golang.org/doc/install) >= 1.22
- [GNU Make](https://www.gnu.org/software/make/)

## Getting started

To initilize Sage in a repository, just run:

```bash
go run go.einride.tech/sage@latest init
```

Run `make`.

Two changes should now have happened. If the project had a previous `Makefile`
it should have been renamed to `Makefile.old` and a new should have been
created. If the project have a dependabot config, a sage config should have been
added.

## Usage

Sage imports, and targets within the Sagefiles, can be written to Makefiles, you
can generate as many Makefiles as you want, see more at
[Makefiles / Sage namespaces](https://github.com/einride/sage#makefiles--sage-namespaces).

### Sagefiles

You can have as many Sagefiles as you want in the `.sage` folder.

#### Targets

Any public function in the main package will be exported. Functions can have no
return value but error. The following arguments are supported: Optional first
argument of context.Context, string, int or bool.

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

To generate Makefiles, a `main` method needs to exist in one of the Sagefiles
where we call the `sg.GenerateMakefiles` method.

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

If another makefile is desired, lets say one that only includes Terraform
targets, we utilize the `sg.Namespace` type and just add another `Makefile` to
the `GenerateMakefiles` method and specify the namespace, path and default
target.

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

It is also possible to embed a Namespace in order to add metadata to it and
potentially reuse it for different Makefiles, the supported fields for an
embedded Namespace are exported String, Int & Boolean.

```golang

func main() {
	sg.GenerateMakefiles(
		sg.Makefile{
			Path:          sg.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
		sg.Makefile{
			Path:          sg.FromGitRoot("names/name1/Makefile"),
			Namespace:     MyNamespace{Name: "name1"},
		},
		sg.Makefile{
			Path:          sg.FromGitRoot("names/name2/Makefile"),
			Namespace:     MyNamespace{Name: "name2"},
        },
	)
}


type MyNamespace struct {
	sg.Namespace
	Name string
}

func (n MyNamespace) PrintName(ctx context.Context) error {
	fmt.Println(n.Name)
}
```

NOTE: The `sg.GenerateMakefiles` function is evaluated when the sage binary is
built so doing something like this

```golang
sg.Makefile{
	Path:          sg.FromGitRoot("names/name1/Makefile"),
	Namespace:     MyNamespace{Name: os.Getenv("Name")},
},
```

will cause whatever value the environment variable `Name` has at the time to be
hardcoded in the built sage binary.

#### Custom Make target names

By default, Sage converts Go function names from PascalCase to kebab-case for
Make targets (e.g. `BuildImage` becomes `build-image`). This conversion treats
every letter-to-number boundary as a word break, so `BuildImageV2` becomes
`build-image-v-2`.

To override the generated target name, add a `//sage:target` annotation in the
function's doc comment:

```golang
//sage:target build-image-v2
func BuildImageV2(ctx context.Context) error {
	// ...
}
```

This generates:

```makefile
.PHONY: build-image-v2
build-image-v2: $(sagefile)
	@$(sagefile) BuildImageV2
```

Target names must consist of lowercase letters, digits, hyphens, underscores,
and dots, and must not start with a dot or hyphen.

#### Dependencies

Dependencies can be defined just by specificing the function, or with `sg.Fn` if
the function takes arguments. `Deps` runs in parallel while `SerialDeps` runs
serially.

```golang
sg.Deps(
	sg.Fn(ConvcoCheck, "origin/main..HEAD"),
	GolangciLint,
)
sg.SerialDeps(
	GoModTidy,
	GitVerifyNoDiff,
)
```
