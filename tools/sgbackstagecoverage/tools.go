package sgbackstagecoverage

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// Post posts code coverage in LCOV format to the Backstage API.
// apiURL is the URL to the API, for example 'https://api.mybackstagedeployment.com'.
// authToken is the Bearer token for the Backstage API.
// coverFile is the coverage file in LCOV format (use sggcov2lcov to convert Go test output).
// entity is the entity ref, for example 'component:default/my-service'.
func Post(ctx context.Context, apiURL, authToken, coverFile, entity string) error {
	url := fmt.Sprintf("%s/api/code-coverage/report?entity=%s&coverageType=lcov", apiURL, entity)
	f, err := os.Open(coverFile)
	if err != nil {
		return fmt.Errorf("opening coverage file: %w", err)
	}
	defer f.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, f)
	if err != nil {
		return fmt.Errorf("creating coverage POST request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("posting coverage data: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("coverage post status not OK: %s", http.StatusText(resp.StatusCode))
	}
	return nil
}
