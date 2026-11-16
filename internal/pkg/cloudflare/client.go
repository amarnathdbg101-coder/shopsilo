// Package goclos provides a zero-dependency, production-grade Go SDK for Cloudflare Workers AI
// and S3-compatible R2 Object Storage.
package goclos

import (
	"net/http"
	"strings"
	"time"
)

// Client is the primary interface to Cloudflare services.
type Client struct {
	cfg        Config
	httpClient *http.Client
	AI         *AIService
	R2         *R2Service
}

// New creates an initialized Cloudflare Client.
func New(cfg Config) (*Client, error) {
	cfg.AccountID = strings.TrimSpace(cfg.AccountID)
	cfg.APIToken = strings.TrimSpace(cfg.APIToken)

	if cfg.AccountID == "" || cfg.APIToken == "" {
		return nil, ErrMissingCredentials
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	c := &Client{
		cfg:        cfg,
		httpClient: httpClient,
	}

	c.AI = &AIService{client: c}
	c.R2 = &R2Service{client: c}

	return c, nil
}

// AccountID returns the active account ID.
func (c *Client) AccountID() string {
	return c.cfg.AccountID
}
