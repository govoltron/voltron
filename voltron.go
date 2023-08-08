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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/govoltron/matrix"
)

var (
	ErrInvalidCluster = errors.New("invalid cluster")
)

type VoltronOption func(vol *Voltron)

type Voltron struct {
	clients  []*client
	services []*service
	cluster  *matrix.Cluster
	ctx      context.Context
	mu       sync.RWMutex
}

// New
func New(ctx context.Context, opts ...VoltronOption) (vol *Voltron) {
	vol = &Voltron{
		ctx:      ctx,
		clients:  make([]*client, 0),
		services: make([]*service, 0),
	}
	// Set options
	for _, setOpt := range opts {
		setOpt(vol)
	}

	return
}

// Run
func (vol *Voltron) Run(ctx context.Context) (err error) {
	if vol.cluster == nil {
		return ErrInvalidCluster
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize
	if err = vol.init(ctx); err != nil {
		return
	}
	// Shutdown all clients
	defer func() {
		for _, c := range vol.clients {
			c.Shutdown(ctx)
		}
	}()

	// Run all services
	return vol.run(ctx)
}

// Setup
func (vol *Voltron) Setup(srv Service, description string, opts ...RunOption) {
	s := &service{
		rt:          srv,
		description: description,
	}
	// Set options
	for _, setOpt := range opts {
		setOpt(s)
	}

	vol.mu.Lock()
	defer vol.mu.Unlock()
	// Add service
	vol.services = append(vol.services, s)
}

// Join
func (vol *Voltron) Join(cluster *matrix.Cluster) {
	if cluster == nil {
		panic(ErrInvalidCluster)
	}
	vol.cluster = cluster
}

// Print
func (vol *Voltron) Print() {
	var (
		buf bytes.Buffer
	)
	vol.Fprint(&buf)
	buf.WriteTo(os.Stdout)
}

// Fprint
func (vol *Voltron) Fprint(w io.Writer) {
	fmt.Fprintf(w, "==================== Overview ====================\n")
	if vol.cluster != nil {
		fmt.Fprintf(w, "集群名: %s\n", vol.cluster.Name())
	} else {
		fmt.Fprintf(w, "集群名: \n")
	}
	fmt.Fprintf(w, "==================== Clients  ====================\n")
	fmt.Fprintf(w, "类型 | 名称 | 描述\n")
	if len(vol.clients) > 0 {
		for _, c := range vol.clients {
			fmt.Fprintf(w, "%s | %s | %s\n", c.Type(), c.Name(), c.Description())
		}
	} else {
		for _, c := range manifest {
			fmt.Fprintf(w, "%s | %s | %s\n", c.Type(), c.Name(), c.Description())
		}
	}
	fmt.Fprintf(w, "==================== Services ====================\n")
	fmt.Fprintf(w, "类型 | 名称 | 描述\n")
	for _, s := range vol.services {
		fmt.Fprintf(w, "%s | %s | %s\n", s.Type(), s.Name(), s.Description())
	}
	fmt.Fprintf(w, "==================================================\n")

}

// init
func (vol *Voltron) init(ctx context.Context) (err error) {
	// Environments
	// TODO

	vol.mu.Lock()
	defer vol.mu.Unlock()
	// Initialize clients
	for _, c := range manifest {
		if discovery, ok := c.options.(*DiscoveryOptions); !ok {
			err = c.Init(ctx)
		} else {
			if err = discovery.initBroker(ctx, vol.cluster); err != nil {
				return
			}
			err = c.InitWithDiscovery(ctx, discovery)
		}
		if err != nil {
			return
		}
		vol.clients = append(vol.clients, c)
	}

	return
}

// run
func (vol *Voltron) run(ctx context.Context) (err error) {
	var (
		wg     = &sync.WaitGroup{}
		signal = make(chan struct{})
		gogo   = func() { close(signal) }
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Prepare all services
	for _, s := range vol.services {
		// Option: reporter
		if s.addr != "" {
			s.reporter = vol.cluster.NewReporter(ctx, s.Name())
		}
		// Prepare
		if err = s.Prepare(ctx, signal, wg); err != nil {
			return
		}
	}

	// Start all services
	gogo()

	// Wait for all running services to exit.
	wg.Wait()

	return
}
