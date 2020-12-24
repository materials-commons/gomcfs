package mcapi

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	token     string
	projectID int
	mcurl     string
	client    *resty.Client
}

var ErrAuth = errors.New("auth")
var ErrMCAPI = errors.New("mcapi")

func NewClient(token string, projectID int, mcurl string) *Client {
	return &Client{
		token:     token,
		projectID: projectID,
		mcurl:     mcurl,
		client:    resty.New(),
	}
}

func (c *Client) ListDirectory(path string) error {
	var files int
	resp, err := c.r().SetQueryParam("path", path).
		SetResult(&files).
		Get(fmt.Sprintf("%s/projects/%d/directories_by_path", c.mcurl, c.projectID))

	if err := c.getAPIError(resp, err); err != nil {
		return err
	}

	return nil
}

func (c *Client) getAPIError(resp *resty.Response, err error) error {
	switch {
	case err != nil:
		return err
	case resp.RawResponse.StatusCode == 401:
		return ErrAuth
	case resp.RawResponse.StatusCode > 299:
		return ErrMCAPI
	default:
		return nil
	}
}

var tlsConfig = tls.Config{InsecureSkipVerify: true}

// r is similar to resty.R() except that it sets the TLS configuration
func (c *Client) r() *resty.Request {
	return c.client.SetTLSClientConfig(&tlsConfig).SetAuthToken(c.token).R()
}
