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
	"errors"

	"github.com/govoltron/matrix"
)

type DiscoveryOptions struct {
	srvname string
	broker  *matrix.Broker
}

// Discovery
func Discovery(srvname string) ClientOptions {
	return &DiscoveryOptions{srvname: srvname}
}

// initBroker
func (opts *DiscoveryOptions) initBroker(ctx context.Context, cluster *matrix.Cluster) (err error) {
	if cluster == nil {
		return errors.New("invalid cluster")
	}
	opts.broker, err = cluster.NewBroker(ctx, opts.srvname)
	return
}

// ServiceName implements ClientOptions.
func (opts *DiscoveryOptions) ServiceName() (srvname string) {
	return opts.srvname
}

// Options
func (opts *DiscoveryOptions) Options(ctx context.Context) (options string) {
	return opts.broker.Getenv(ctx, "options")
}

// Broker
func (opts *DiscoveryOptions) Broker() (broker *matrix.Broker) {
	return opts.broker
}
