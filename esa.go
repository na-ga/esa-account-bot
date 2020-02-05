package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

//
type EsaClient struct {
	endpoint string
	teamName string
	token    string
	http     *http.Client
}

//
func NewEsaClient(teamName, token string) (*EsaClient, error) {
	if teamName == "" {
		return nil, errors.New("teamName is required")
	}
	if token == "" {
		return nil, errors.New("token is required")
	}
	return &EsaClient{
		endpoint: "https://api.esa.io/v1/teams/" + teamName,
		teamName: teamName,
		token:    token,
		http:     &http.Client{},
	}, nil
}

type QueryOption func(*url.URL) error

func QueryOptionPage(value int) QueryOption {
	return func(in *url.URL) error {
		if value < 1 {
			return fmt.Errorf("invalid page query: %d", value)
		}
		q := in.Query()
		q.Set("page", fmt.Sprintf("%d", value))
		in.RawQuery = q.Encode()
		return nil
	}
}

func QueryOptionPerPage(value int) QueryOption {
	return func(in *url.URL) error {
		if value < 1 && 100 < value {
			return fmt.Errorf("invalid per page query: %d", value)
		}
		q := in.Query()
		q.Set("per_page", fmt.Sprintf("%d", value))
		in.RawQuery = q.Encode()
		return nil
	}
}

func QueryOptionSort(value string) QueryOption {
	return func(in *url.URL) error {
		if value == "" {
			return fmt.Errorf("invalid sort query: %s", value)
		}
		q := in.Query()
		q.Set("sort", value)
		in.RawQuery = q.Encode()
		return nil
	}
}

func QueryOptionOrder(value string) QueryOption {
	return func(in *url.URL) error {
		if value != "asc" && value != "desc" {
			return fmt.Errorf("invalid order query: %s", value)
		}
		q := in.Query()
		q.Set("order", value)
		in.RawQuery = q.Encode()
		return nil
	}
}

func buildURL(raw string, options ...QueryOption) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	for _, opt := range options {
		if err := opt(u); err != nil {
			return "", err
		}
	}
	return u.String(), nil
}

//
func (c *EsaClient) GetTeamName() string {
	return c.teamName
}

//
func (c *EsaClient) InviteAccount(email string) error {
	url, err := buildURL(c.endpoint + "/invitations")
	if err != nil {
		return err
	}
	body := bytes.NewBuffer([]byte("{\"member\":{\"emails\":[\"" + email + "\"]}}"))
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || 300 <= res.StatusCode {
		return fmt.Errorf("invalid status code: %d", res.StatusCode)
	}
	return nil
}

//
func (c *EsaClient) DeleteAccount(screenName string) error {
	url, err := buildURL(c.endpoint + "/members/" + screenName)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return fmt.Errorf("status code is 404, the specified account (%s) has already been deleted", screenName)
	}
	if res.StatusCode < 200 || 300 <= res.StatusCode {
		return fmt.Errorf("invalid status code: %d", res.StatusCode)
	}
	return nil
}

type ListAccountResponse struct {
	Members    []*Member `json:"members"`
	PrevPage   int       `json:"prev_page"`
	NextPage   int       `json:"next_page"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	MaxPerPage int       `json:"max_per_page"`
}

type Member struct {
	Name           string `json:"name"`
	ScreenName     string `json:"screen_name"`
	Icon           string `json:"icon"`
	PostsCount     int    `json:"posts_count"`
	JoinedAt       string `json:"joined_at"`
	LastAccessedAt string `json:"last_accessed_at"`
	Email          string `json:"email"`
}

func (m *Member) LastAccessedTime() (time.Time, error) {
	return time.Parse(time.RFC3339, m.LastAccessedAt)
}

func (m *Member) JoinedTime() (time.Time, error) {
	return time.Parse(time.RFC3339, m.JoinedAt)
}

//
func (c *EsaClient) ListAccount(options ...QueryOption) (*ListAccountResponse, error) {
	url, err := buildURL(c.endpoint+"/members", options...)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || 300 <= res.StatusCode {
		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var ret *ListAccountResponse
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}
