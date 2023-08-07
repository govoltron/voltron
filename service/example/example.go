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

package example

import (
	"context"
	"fmt"
)

type Example struct {
}

// New
func New() *Example {
	return &Example{}
}

// Name implements voltron.Service
func (ex *Example) Name() (name string) {
	return "example"
}

// Init implements voltron.Service
func (ex *Example) Init(ctx context.Context) (err error) {
	return
}

// Run implements voltron.Service
func (ex *Example) Run(ctx context.Context) {
	fmt.Printf("Welcome to example\n")
}

// Shutdown implements voltron.Service
func (ex *Example) Shutdown(ctx context.Context) {

}
