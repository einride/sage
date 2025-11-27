package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// goModuleInfo is the response from the Go proxy @latest endpoint.
type goModuleInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

// GetLatestGoModuleVersion fetches the latest version from the Go proxy.
// The module path should be the full module path (e.g., "github.com/google/go-licenses/v2").
// Returns the version string (e.g., "v2.0.1").
func GetLatestGoModuleVersion(ctx context.Context, module string) (string, error) {
	// Escape module path for URL (slashes become %2F, uppercase letters get escaped)
	escapedModule := strings.ReplaceAll(module, "/", "%2F")
	url := fmt.Sprintf("https://proxy.golang.org/%s/@latest", escapedModule)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch module info for %s: %w", module, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("go proxy returned status %d for %s", resp.StatusCode, module)
	}

	var info goModuleInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return info.Version, nil
}
