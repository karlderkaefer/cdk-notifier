package provider

// Copyright 2013 The go-github AUTHORS. All rights reserved.
// client implementation inspired from https://github.com/google/go-github/blob/master/github/github.go

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/sirupsen/logrus"
)

var (
	errNonNilContext = errors.New("context must be non-nil")
)

const (
	BitbucketDefaultBaseURL = "https://api.bitbucket.org/2.0/"
)

type BitbucketRepositoryService bitbucketService

type service struct {
	client *BitbucketClient
}

type BitbucketClient struct {
	*http.Client
	common       service
	BaseURL      *url.URL
	UserAgent    string
	Repositories *BitbucketRepositoryService
}

type bitbucketService struct {
	client *BitbucketClient
}

type IBitbucketRepositoryService interface {
	ListComments(ctx context.Context, owner string, repo string, prId int64, opts *ListCommentOptions) (*BitbucketComments, *http.Response, error)
	EditComment(ctx context.Context, owner string, repo string, prId int64, commentID int64, comment *BitbucketComment) (*BitbucketComment, *http.Response, error)
	CreateComment(ctx context.Context, owner string, repo string, prId int64, comment *BitbucketComment) (*BitbucketComment, *http.Response, error)
	DeleteComment(ctx context.Context, owner string, repo string, prId int64, commentID int64) (*BitbucketComment, *http.Response, error)
}

type BitbucketComment struct {
	Content *BitbucketContent `json:"content,omitempty"`
	Id      *int64            `json:"id,omitempty"`
	Links   *BitbucketLinks   `json:"links,omitempty"`
}

type BitbucketContent struct {
	Raw string `json:"raw,omitempty"`
}

type BitbucketComments struct {
	Values []BitbucketComment `json:"values,omitempty"`
}

type BitbucketLinks struct {
	Html BitbucketHtml `json:"html,omitempty"`
}

type BitbucketHtml struct {
	Href string `json:"href,omitempty"`
}

type BitbucketProxy struct {
	Proxied  http.RoundTripper
	username string
	password string
}

type ListCommentOptions struct {
	Query      string `url:"q,omitempty"`
	Fields     string `url:"fields,omitempty"`
	PageLength int    `url:"pagelen,omitempty"`
}

func (proxy BitbucketProxy) RoundTrip(req *http.Request) (res *http.Response, e error) {
	msg := fmt.Sprintf("Sending request to %s/%s", req.URL.Host, req.URL.Path)
	logrus.Debug(strings.ReplaceAll(msg, "\n", ""))
	req.SetBasicAuth(proxy.username, proxy.password)
	req.Header.Add("Accept", "application/json")
	return proxy.Proxied.RoundTrip(req)
}

func NewBitbucketComment(content string) *BitbucketComment {
	return &BitbucketComment{
		Content: &BitbucketContent{content},
	}
}

func NewBitbucketClient(username string, password string) *BitbucketClient {
	httpClient := &http.Client{
		Timeout: time.Duration(15) * time.Second,
		Transport: BitbucketProxy{
			http.DefaultTransport, username, password,
		},
	}
	baseURL, _ := url.Parse(BitbucketDefaultBaseURL)
	client := &BitbucketClient{
		Client:    httpClient,
		BaseURL:   baseURL,
		UserAgent: userAgent,
	}
	client.common.client = client
	client.Repositories = (*BitbucketRepositoryService)(&client.common)
	return client
}

func (s *BitbucketRepositoryService) ListComments(ctx context.Context, owner string, repo string, prId int64, opts *ListCommentOptions) (*BitbucketComments, *http.Response, error) {
	u := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", owner, repo, prId)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	comments := &BitbucketComments{}
	resp, err := s.client.Do(ctx, req, comments)
	if err != nil {
		return nil, resp, err
	}
	return comments, resp, nil
}

func (s *BitbucketRepositoryService) EditComment(ctx context.Context, owner string, repo string, prId int64, commentID int64, comment *BitbucketComment) (*BitbucketComment, *http.Response, error) {
	u := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments/%d", owner, repo, prId, commentID)
	req, err := s.client.NewRequest(http.MethodPut, u, comment)
	if err != nil {
		return nil, nil, err
	}
	commentResp := &BitbucketComment{}
	resp, err := s.client.Do(ctx, req, commentResp)
	if err != nil {
		return nil, resp, err
	}
	return commentResp, resp, nil
}

func (s *BitbucketRepositoryService) CreateComment(ctx context.Context, owner string, repo string, prId int64, comment *BitbucketComment) (*BitbucketComment, *http.Response, error) {
	u := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", owner, repo, prId)
	req, err := s.client.NewRequest(http.MethodPost, u, comment)
	if err != nil {
		return nil, nil, err
	}
	commentResp := &BitbucketComment{}
	resp, err := s.client.Do(ctx, req, commentResp)
	if err != nil {
		return nil, resp, err
	}
	return commentResp, resp, nil
}

func (s *BitbucketRepositoryService) DeleteComment(ctx context.Context, owner string, repo string, prId int64, commentID int64) (*BitbucketComment, *http.Response, error) {
	u := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments/%d", owner, repo, prId, commentID)
	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, nil, err
	}
	commentResp := &BitbucketComment{}
	resp, err := s.client.Do(ctx, req, commentResp)
	if err != nil {
		return nil, resp, err
	}
	return commentResp, resp, nil
}

func (c *BitbucketClient) NewRequest(method string, url string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}
	u, err := c.BaseURL.Parse(url)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	return req, nil
}

func addOptions(s string, opts interface{}) (string, error) {
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opts)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

func (c *BitbucketClient) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	if ctx == nil {
		return nil, errNonNilContext
	}
	resp, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		return nil, err
	}
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	decErr := json.Unmarshal(body, v)
	if decErr != nil {
		logrus.Warnf("could not parse response to %s", reflect.TypeOf(v))
	}
	if resp.StatusCode >= 300 {
		err = fmt.Errorf("BitBucket API Error: %s %s", resp.Status, body)
	}
	return resp, err
}
