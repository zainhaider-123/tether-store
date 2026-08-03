// Package search queries the skills.sh public search API.
package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const apiURL = "https://skills.sh/api/search"

// Result is one skill hit from skills.sh.
type Result struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Installs    int    `json:"installs"`
	Stars       int    `json:"stars"`
}

type apiResponse struct {
	Skills []Result `json:"skills"`
	// Some responses may use "results"
	Results []Result `json:"results"`
}

// Query searches skills.sh for q. Empty q returns trending/popular if the API allows.
func Query(q string) ([]Result, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	vals := u.Query()
	if q != "" {
		vals.Set("q", q)
	}
	u.RawQuery = vals.Encode()

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "tether/0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skills.sh search: %s: %s", resp.Status, truncate(string(body), 200))
	}

	var ar apiResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		// try bare array
		var list []Result
		if err2 := json.Unmarshal(body, &list); err2 == nil {
			return list, nil
		}
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	if len(ar.Skills) > 0 {
		return ar.Skills, nil
	}
	return ar.Results, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
