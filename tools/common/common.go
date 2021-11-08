package common

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

func WireGenerate(path string) error {
	fmt.Println("[wire-generate] generating initializers...")
	err := sh.RunV("go", "run", "-mod=mod", "github.com/google/wire/cmd/wire", "gen", path)
	if err != nil {
		return err
	}
	return nil
}

func GoTest() error {
	err := sh.RunV("go", "test", "-race", "-cover", "./...")
	if err != nil {
		return err
	}
	return nil
}

func GomModTidy() error {
	err := sh.RunV("go", "mod", "tidy", "-v")
	if err != nil {
		return err
	}
	return nil
}
