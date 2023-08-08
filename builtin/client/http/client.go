// Copyright 2023 Kami
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/govoltron/matrix"
	"github.com/govoltron/matrix/balance"
	"github.com/govoltron/voltron"
)

var (
	ErrClientNotReady = errors.New("http client is not ready")
)

type ClientOptions struct {
	SevName    string            `json:"-"`
	Raw        string            `json:"-"`
	Endpoints  []matrix.Endpoint `json:"-"`
	Scheme     string            `json:"scheme,omitempty"`
	Host       string            `json:"host,omitempty"`
	Timeout    int64             `json:"timeout,omitempty"`
	RetryCount int               `json:"retry_count,omitempty"`
}

// ServiceName
func (opts *ClientOptions) ServiceName() (name string) {
	return opts.SevName
}

type Client struct {
	scheme    string
	host      string
	endpoints []matrix.Endpoint
	balancer  *balance.WeightRoundRobinBalancer
	client    *httpclient.Client
	ready     uint32
}

// Name implements voltron.Client
func (c *Client) Name() (name string) {
	return "httpclient"
}

// Init implements voltron.Client
func (c *Client) Init(ctx context.Context, opts voltron.ClientOptions) (err error) {
	co, ok := opts.(*ClientOptions)
	if !ok {
		return fmt.Errorf("invalid options for http client")
	}

	if !atomic.CompareAndSwapUint32(&c.ready, 0, 1) {
		return
	}

	hcopts := make([]httpclient.Option, 0)
	if co.RetryCount > 0 {
		hcopts = append(hcopts, httpclient.WithRetryCount(co.RetryCount))
	}
	if co.Timeout > 0 {
		hcopts = append(hcopts, httpclient.WithHTTPTimeout(time.Duration(co.Timeout)*time.Millisecond))
	}

	c.host = co.Host
	c.scheme = co.Scheme
	c.endpoints = co.Endpoints
	c.client = httpclient.NewClient(hcopts...)

	c.balancer = balance.NewWeightRoundRobinBalancer()
	for _, endpoint := range c.endpoints {
		c.balancer.Add(balance.Endpoint{Addr: endpoint.Addr, Weight: endpoint.Weight})
	}

	return
}

// ReInit
func (c *Client) ReInit(ctx context.Context, opts voltron.ClientOptions) (err error) {
	return
}

// NewOptions
func (c *Client) NewOptions(ctx context.Context, options []byte, endpoints []matrix.Endpoint) voltron.ClientOptions {
	var (
		opts = &ClientOptions{}
	)
	opts.Raw = string(options)
	// Decode options
	json.Unmarshal(options, opts)
	// Endpoints
	for _, endpoint := range endpoints {
		opts.Endpoints = append(opts.Endpoints, endpoint)
	}
	return opts
}

// Shutdown implements voltron.Client
func (c *Client) Shutdown(ctx context.Context) {
	atomic.CompareAndSwapUint32(&c.ready, 1, 0)
}

func (c *Client) Get(uri string, headers http.Header) (*http.Response, error) {
	if atomic.LoadUint32(&c.ready) != 1 {
		return nil, ErrClientNotReady
	}
	return c.client.Get(c.buildUrl(uri), c.buildHeaders(headers))
}

func (c *Client) Post(uri string, body io.Reader, headers http.Header) (*http.Response, error) {
	if atomic.LoadUint32(&c.ready) != 1 {
		return nil, ErrClientNotReady
	}
	return c.client.Post(c.buildUrl(uri), body, c.buildHeaders(headers))
}

func (c *Client) Put(uri string, body io.Reader, headers http.Header) (*http.Response, error) {
	if atomic.LoadUint32(&c.ready) != 1 {
		return nil, ErrClientNotReady
	}
	return c.client.Put(c.buildUrl(uri), body, c.buildHeaders(headers))
}

func (c *Client) Patch(uri string, body io.Reader, headers http.Header) (*http.Response, error) {
	if atomic.LoadUint32(&c.ready) != 1 {
		return nil, ErrClientNotReady
	}
	return c.client.Patch(c.buildUrl(uri), body, c.buildHeaders(headers))
}

func (c *Client) Delete(uri string, headers http.Header) (*http.Response, error) {
	if atomic.LoadUint32(&c.ready) != 1 {
		return nil, ErrClientNotReady
	}
	return c.client.Delete(c.buildUrl(uri), c.buildHeaders(headers))
}

// buildUrl
func (c *Client) buildUrl(uri string) string {
	addr := c.scheme + "://" + c.balancer.Next()
	if uri == "/" {
		return addr
	}
	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}
	return addr + uri
}

// buildHeader
func (c *Client) buildHeaders(headers http.Header) (newheaders http.Header) {
	if headers == nil {
		headers = make(http.Header)
	}
	if c.host != "" {
		headers.Set("Host", c.host)
	}
	return headers
}
