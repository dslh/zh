// Package gh provides GitHub API access via the gh CLI or a personal access token.
package gh

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/dslh/zh/internal/exitcode"
)

// Client provides GitHub GraphQL API access.
type Client struct {
	method     string // "gh" or "pat"
	token      string // PAT token (only for method=pat)
	endpoint   string // GraphQL endpoint URL
	httpClient *http.Client
	verbose    bool
	logFunc    func(format string, args ...any)
}

// Option configures a Client.
type Option func(*Client)

// WithVerbose enables verbose logging.
func WithVerbose(logFunc func(format string, args ...any)) Option {
	return func(c *Client) {
		c.verbose = true
		c.logFunc = logFunc
	}
}

// WithEndpoint sets a custom GitHub GraphQL endpoint (useful for testing).
func WithEndpoint(endpoint string) Option {
	return func(c *Client) { c.endpoint = endpoint }
}

// New creates a new GitHub API client. Method should be "gh" or "pat".
// For "pat" method, token must be provided.
// Returns nil if method is "none" or empty.
func New(method, token string, opts ...Option) *Client {
	if method == "" || method == "none" {
		return nil
	}
	c := &Client{
		method:     method,
		token:      token,
		endpoint:   githubGraphQLEndpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

const githubGraphQLEndpoint = "https://api.github.com/graphql"

// graphQLRequest is the JSON body sent to the GitHub GraphQL API.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the top-level JSON response from the GitHub GraphQL API.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Execute sends a GraphQL query to the GitHub API.
func (c *Client) Execute(query string, variables map[string]any) (json.RawMessage, error) {
	if c.method == "gh" {
		return c.executeViaGhCLI(query, variables)
	}
	return c.executeViaPAT(query, variables)
}

func (c *Client) executeViaPAT(query string, variables map[string]any) (json.RawMessage, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, exitcode.General("marshaling GitHub request", err)
	}

	if c.verbose {
		c.log("→ GitHub POST %s\n", c.endpoint)
		c.log("→ Query: %s\n", query)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, exitcode.General("creating GitHub request", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "zh-cli/dev")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exitcode.General("GitHub API request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exitcode.General("reading GitHub response", err)
	}

	if c.verbose {
		c.log("← GitHub %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, exitcode.Auth("GitHub authentication failed — check your token", nil)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, exitcode.Generalf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, exitcode.General("parsing GitHub response", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, exitcode.Generalf("GitHub GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

func (c *Client) executeViaGhCLI(query string, variables map[string]any) (json.RawMessage, error) {
	// Build the JSON request body and pass via stdin to avoid shell escaping issues
	// with $ characters in GraphQL variables.
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, exitcode.General("marshaling GitHub request", err)
	}

	args := []string{"api", "graphql", "--input", "-"}

	if c.verbose {
		c.log("→ gh %v\n", args)
		c.log("→ Body: %s\n", string(bodyBytes))
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdin = bytes.NewReader(bodyBytes)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, exitcode.Generalf("gh CLI error: %s", errMsg)
	}

	if c.verbose {
		c.log("← gh response: %d bytes\n", stdout.Len())
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(stdout.Bytes(), &gqlResp); err != nil {
		return nil, exitcode.General("parsing gh CLI response", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, exitcode.Generalf("GitHub GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

func (c *Client) log(format string, args ...any) {
	if c.logFunc != nil {
		c.logFunc(format, args...)
	}
}
