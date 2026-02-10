package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dslh/zh/internal/exitcode"
)

const DefaultRESTEndpoint = "https://api.zenhub.com"

// RESTEndpoint returns the REST API base URL derived from the client's
// GraphQL endpoint. For the default endpoint this returns
// "https://api.zenhub.com"; for test servers it returns the mock URL.
func (c *Client) RESTEndpoint() string {
	// If using the default GraphQL endpoint, use the default REST endpoint.
	if c.endpoint == DefaultEndpoint {
		return DefaultRESTEndpoint
	}
	// For test/custom endpoints, strip the /public/graphql suffix.
	ep := strings.TrimSuffix(c.endpoint, "/public/graphql")
	return ep
}

// RESTIssueRef identifies an issue by repository GitHub ID and issue number,
// as used by the ZenHub REST API v1.
type RESTIssueRef struct {
	RepoID      int `json:"repo_id"`
	IssueNumber int `json:"issue_number"`
}

// UpdateEpicIssues adds and/or removes issues from a legacy epic via the
// ZenHub REST API v1. The epic is identified by the GitHub repository ID and
// issue number of its backing GitHub issue.
func (c *Client) UpdateEpicIssues(epicRepoID, epicIssueNumber int, addIssues, removeIssues []RESTIssueRef) error {
	body := map[string]any{}
	if len(addIssues) > 0 {
		body["add_issues"] = addIssues
	}
	if len(removeIssues) > 0 {
		body["remove_issues"] = removeIssues
	}

	url := fmt.Sprintf("%s/p1/repositories/%d/epics/%d/update_issues",
		c.RESTEndpoint(), epicRepoID, epicIssueNumber)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return exitcode.General("marshaling request", err)
	}

	if c.verbose {
		c.log("→ POST %s\n", url)
		c.log("→ Body: %s\n", string(bodyBytes))
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return exitcode.General("creating request", err)
	}

	req.Header.Set("X-Authentication-Token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return exitcode.General("API request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return exitcode.General("reading response", err)
	}

	if c.verbose {
		c.log("← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		c.log("← Body: %s\n", truncate(string(respBody), 2000))
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return exitcode.Auth("authentication failed — check your API key", nil)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return exitcode.Generalf("API returned HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	return nil
}
