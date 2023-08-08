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
	"sync"

	"github.com/govoltron/matrix"
)

type Service interface {

	// Name
	Name() (name string)

	// Init
	Init(ctx context.Context) (err error)

	// Run
	Run(ctx context.Context)

	// Shutdown
	Shutdown(ctx context.Context)
}

type (
	ServiceFunc func(ctx context.Context)
)

func (fun ServiceFunc) Name() (name string)                  { return "service-run-function" }
func (fun ServiceFunc) Init(ctx context.Context) (err error) { return }
func (fun ServiceFunc) Run(ctx context.Context)              { fun(ctx) }
func (fun ServiceFunc) Shutdown(ctx context.Context)         {}

// Service options
type RunOption func(s *service)

// WithAutoReport
func WithAutoReport(addr string, weight int, ttl int64) RunOption {
	return func(s *service) { s.addr, s.weight, s.ttl = addr, weight, ttl }
}

type service struct {

	// Service runtime.
	rt Service

	// Service description.
	description string

	// Reporter
	reporter *matrix.Reporter
	addr     string
	weight   int
	ttl      int64
}

// Type implements Runner
func (s *service) Type() string {
	return "service"
}

// Name implements Runner
func (s *service) Name() string {
	return s.rt.Name()
}

// Description implements Runner
func (s *service) Description() string {
	return s.description
}

// Prepare
func (s *service) Prepare(ctx context.Context, signal chan struct{}, wg *sync.WaitGroup) (err error) {
	// 1. Initialize service
	if err = s.rt.Init(ctx); err != nil {
		return
	}

	// 2. Boot service
	s.boot(ctx, signal, wg)

	return
}

// bootstrap
func (s *service) boot(ctx context.Context, signal chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Wait signal
		select {
		// Cancel
		case <-ctx.Done():
			return
		// Go
		case <-signal:
			// Nothing to do
		}

		// Reporter
		if s.reporter != nil {
			s.reporter.Keepalive(s.addr, s.weight, s.ttl)
			defer s.reporter.Close(ctx)
		}

		// Run
		s.rt.Run(ctx)
	}()
}
