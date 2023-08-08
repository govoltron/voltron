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

package voltron

import (
	"context"
	"reflect"
	"sync"

	"github.com/govoltron/matrix"
	"github.com/thecxx/runpoint"
)

type Client interface {

	// Name
	Name() (name string)

	// Init
	Init(ctx context.Context, opts ClientOptions) (err error)

	// ReInit
	ReInit(ctx context.Context, opts ClientOptions) (err error)

	// NewOptions
	NewOptions(ctx context.Context, options []byte, endpoints []matrix.Endpoint) (opts ClientOptions)

	// Shutdown
	Shutdown(ctx context.Context)
}

type ClientOptions interface {

	// Service name.
	ServiceName() (srvname string)
}

type client struct {

	// Variable pointer
	rt Client

	// PC
	pcounter *runpoint.PCounter

	// Client options
	options ClientOptions

	// Client description
	description string
}

// Type
func (c *client) Type() string {
	return "client"
}

// Name
func (c *client) Name() string {
	return c.rt.Name()
}

// Description
func (c *client) Description() string {
	return c.description
}

// Init
func (c *client) Init(ctx context.Context) (err error) {
	return c.rt.Init(ctx, c.options)
}

// Init
func (c *client) InitWithDiscovery(ctx context.Context, discovery *DiscoveryOptions) (err error) {
	watcher := &discoveryWatcher{c: c, discovery: discovery}
	return watcher.Init(ctx)
}

// Shutdown
func (c *client) Shutdown(ctx context.Context) {
	c.rt.Shutdown(ctx)
}

type discoveryWatcher struct {
	c         *client
	discovery *DiscoveryOptions
	options   []byte
	endpoints map[string]matrix.Endpoint
	mu        sync.RWMutex
}

// InitClient
func (w *discoveryWatcher) Init(ctx context.Context) (err error) {
	defer func() {
		w.discovery.broker.Watch(w)
	}()

	w.options = []byte(w.discovery.Options(ctx))
	w.endpoints = make(map[string]matrix.Endpoint)

	endpoints := w.discovery.broker.Endpoints()
	for _, endpoint := range endpoints {
		w.endpoints[endpoint.ID] = endpoint
	}

	return w.c.rt.Init(ctx,
		w.c.rt.NewOptions(ctx, w.options, endpoints),
	)
}

// reinit
func (w *discoveryWatcher) reinit() (err error) {
	endpoints := make([]matrix.Endpoint, 0)
	for _, endpoint := range w.endpoints {
		endpoints = append(endpoints, endpoint)
	}
	ctx := context.Background()
	return w.c.rt.ReInit(ctx,
		w.c.rt.NewOptions(ctx, w.options, endpoints),
	)
}

// OnSetenv implements matrix.BrokerWatcher.
func (w *discoveryWatcher) OnSetenv(key string, value string) {
	if key == "options" {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.options = []byte(value)
		w.reinit()
	}
}

// OnDelenv implements matrix.BrokerWatcher.
func (w *discoveryWatcher) OnDelenv(key string) {
	if key == "options" {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.options = nil
		w.reinit()
	}
}

// OnUpdateEndpoint implements matrix.BrokerWatcher.
func (w *discoveryWatcher) OnUpdateEndpoint(endpoint matrix.Endpoint) {
	w.mu.RLock()
	if ep, ok := w.endpoints[endpoint.ID]; ok &&
		ep.Addr == endpoint.Addr && ep.Weight == endpoint.Weight {
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()

	w.mu.Lock()
	defer w.mu.Unlock()
	w.endpoints[endpoint.ID] = endpoint
	w.reinit()
}

// OnDeleteEndpoint implements matrix.BrokerWatcher.
func (w *discoveryWatcher) OnDeleteEndpoint(id string) {
	w.mu.RLock()
	if _, ok := w.endpoints[id]; !ok {
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()

	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.endpoints, id)
	w.reinit()
}

var (
	mutex    sync.Mutex
	manifest []*client
)

// ClientVarP register a client and bind it to a variable.
func ClientVarP(cliptr Client, opts ClientOptions, description string) {
	rv := reflect.ValueOf(cliptr)
	if cliptr == nil || rv.Kind() != reflect.Pointer || rv.IsNil() {
		panic("expected variable pointer, and not nil")
	}
	cli := &client{
		rt:          cliptr,
		pcounter:    runpoint.PC(1),
		options:     opts,
		description: description,
	}

	mutex.Lock()
	defer mutex.Unlock()
	//
	manifest = append(manifest, cli)
}
