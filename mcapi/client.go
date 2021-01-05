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

type FileResponse struct {
	Data MCFile `json:"data"`
}

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

func (c *Client) GetFileByPath(path string) (*MCFile, error) {
	req := struct {
		Path      string `json:"path"`
		ProjectID int    `json:"project_id"`
	}{
		Path:      path,
		ProjectID: c.projectID,
	}

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

// form = {"name": name, "directory_id": parent_id, "project_id": project_id}
//        form = merge_dicts(form, attrs.to_dict())
//        return File(self.post("/directories", form))

func (c *Client) CreateDirectory(name string, parentDirectoryId int) (*MCFile, error) {
	req := struct {
		Name        string `json:"name"`
		DirectoryId int    `json:"directory_id"`
		ProjectID   int    `json:"project_id"`
	}{
		Name:        name,
		DirectoryId: parentDirectoryId,
		ProjectID:   c.projectID,
	}

	var result FileResponse

	resp, err := c.resty.R().SetResult(&result).SetBody(req).Post("/directories")

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
