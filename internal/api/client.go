// Package api provides a GraphQL client for the ZenHub API.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/dslh/zh/internal/exitcode"
)

const (
	DefaultEndpoint = "https://api.zenhub.com/public/graphql"
	userAgent       = "zh-cli/dev"
)

// Client is a ZenHub GraphQL API client.
type Client struct {
	endpoint   string
	apiKey     string
	restAPIKey string // separate token for REST v1 API (legacy epic operations)
	httpClient *http.Client
	verbose    bool
	logFunc    func(format string, args ...any) // writes to stderr when verbose
}

// Option configures a Client.
type Option func(*Client)

// WithEndpoint sets a custom API endpoint (useful for testing).
func WithEndpoint(endpoint string) Option {
	return func(c *Client) { c.endpoint = endpoint }
}

// WithVerbose enables verbose request/response logging.
func WithVerbose(logFunc func(format string, args ...any)) Option {
	return func(c *Client) {
		c.verbose = true
		c.logFunc = logFunc
	}
}

// WithRESTAPIKey sets the REST v1 API key for legacy epic operations.
func WithRESTAPIKey(key string) Option {
	return func(c *Client) { c.restAPIKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a new ZenHub API client.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		endpoint:   DefaultEndpoint,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// graphQLRequest is the JSON body sent to the GraphQL API.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the top-level JSON response from the GraphQL API.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

// graphQLError represents an error in a GraphQL response.
type graphQLError struct {
	Message string `json:"message"`
	Path    []any  `json:"path"`
}

// Execute sends a GraphQL query and returns the raw JSON data field.
func (c *Client) Execute(query string, variables map[string]any) (json.RawMessage, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, exitcode.General("marshaling request", err)
	}

	if c.verbose {
		c.log("→ POST %s\n", c.endpoint)
		c.log("→ Query: %s\n", query)
		if len(variables) > 0 {
			varsJSON, _ := json.MarshalIndent(variables, "  ", "  ")
			c.log("→ Variables: %s\n", varsJSON)
		}
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, exitcode.General("creating request", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, exitcode.General("API request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exitcode.General("reading response", err)
	}

	if c.verbose {
		c.log("← %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		c.log("← Body: %s\n", truncate(string(respBody), 2000))
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if secs, err := strconv.Atoi(retryAfter); err == nil {
				return nil, exitcode.Generalf("rate limited — retry after %d seconds", secs)
			}
		}
		return nil, exitcode.Generalf("rate limited — try again later")
	}

	// Handle auth failures
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, exitcode.Auth("authentication failed — check your API key", nil)
	}

	// Handle other HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, exitcode.Generalf("API returned HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, exitcode.General("parsing API response", err)
	}

	// Surface GraphQL-level errors
	if len(gqlResp.Errors) > 0 {
		return gqlResp.Data, &GraphQLError{Errors: gqlResp.Errors}
	}

	return gqlResp.Data, nil
}

// GraphQLError represents one or more errors from the GraphQL API.
type GraphQLError struct {
	Errors []graphQLError
}

func (e *GraphQLError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Message
	}
	msg := fmt.Sprintf("%d GraphQL errors:", len(e.Errors))
	for _, err := range e.Errors {
		msg += "\n  - " + err.Message
	}
	return msg
}

func (c *Client) log(format string, args ...any) {
	if c.logFunc != nil {
		c.logFunc(format, args...)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
