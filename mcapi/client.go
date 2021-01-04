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
	resty     *resty.Client
}

var ErrAuth = errors.New("auth")
var ErrMCAPI = errors.New("mcapi")

func NewClient(mcurl string, token string, projectID int) *Client {
	c := &Client{
		token:     token,
		projectID: projectID,
		mcurl:     mcurl,
		resty:     resty.New(),
	}

	c.resty.
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetAuthToken(c.token).
		SetHostURL(c.mcurl)

	return c
}

func (c *Client) ListDirectory(path string) ([]MCFile, error) {
	var result struct {
		Data []MCFile `json:"data"`
	}
	resp, err := c.resty.R().SetQueryParam("path", path).
		SetResult(&result).
		Get(fmt.Sprintf("%s/projects/%d/directories_by_path", c.mcurl, c.projectID))

	if err := c.getAPIError(resp, err); err != nil {
		return nil, err
	}

	return result.Data, nil
}

//  form = {"path": file_path, "project_id": project_id}
//        return File(self.post("/files/by_path", form))

func (c *Client) GetFileByPath(path string) (*MCFile, error) {
	var req struct {
		Path      string `json:"path"`
		ProjectID int    `json:"project_id"`
	}
	req.Path = path
	req.ProjectID = c.projectID

	var result struct {
		Data MCFile `json:"data"`
	}

	resp, err := c.resty.R().SetQueryParam("path", path).
		SetResult(&result).
		SetBody(req).
		Post("/files/by_path")

	if err := c.getAPIError(resp, err); err != nil {
		return nil, err
	}

	return &result.Data, nil
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
