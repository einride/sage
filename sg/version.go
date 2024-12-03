package sg

import "runtime/debug"

// GetSageVersion returns go.einride.tech/sage version in use by reading binary build information.
func GetSageVersion() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath && dep.Version != "(devel)" {
			return dep.Version, true
		}
	}
	return "", false
}
