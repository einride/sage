package mgtool

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/mg"
)

const (
	AMD64 = "amd64"
	X8664 = "x86_64"
)

type Prepare mg.Namespace

func IsSupportedVersion(versions []string, version, name string) error {
	for _, a := range versions {
		if a == version {
			return nil
		}
	}
	return fmt.Errorf(
		"the following %s versions are supported: %s",
		name,
		strings.Join(versions, ", "),
	)
}
