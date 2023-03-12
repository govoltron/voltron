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

package adapter

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
)

type HTTPServer struct {

	// Multicore indicates whether the engine will be effectively created with multi-cores, if so,
	// then you must take care with synchronizing memory between all event callbacks, otherwise,
	// it will run the engine with single thread. The number of threads in the engine will be automatically
	// assigned to the value of logical CPUs usable by the current process.
	Multicore bool

	// LockOSThread is used to determine whether each I/O event-loop is associated to an OS thread, it is useful when you
	// need some kind of mechanisms like thread local storage, or invoke certain C libraries (such as graphics lib: GLib)
	// that require thread-level manipulation via cgo, or want all I/O event-loops to actually run in parallel for a
	// potential higher performance.
	LockOSThread bool

	// NumEventLoop is set up to start the given number of event-loop goroutine.
	// Note: Setting up NumEventLoop will override Multicore.
	NumEventLoop int

	// ReuseAddr indicates whether to set up the SO_REUSEADDR socket option.
	ReuseAddr bool

	// ReusePort indicates whether to set up the SO_REUSEPORT socket option.
	ReusePort bool

	// SocketRecvBuffer sets the maximum socket receive buffer in bytes.
	SocketRecvBuffer int

	// SocketSendBuffer sets the maximum socket send buffer in bytes.
	SocketSendBuffer int

	// TCPKeepAlive sets up a duration for (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration

	Router chi.Router

	err error

	wg sync.WaitGroup
}

// Start implements voltron.Adapter
func (hs *HTTPServer) Start(ctx context.Context, addr string) error {
	// hs.l.addr = addr
	// hs.l.svr.OnBoot = hs.l.OnBoot
	// hs.l.svr.OnShutdown = hs.l.OnShutdown
	// hs.l.svr.OnConnect = hs.l.OnConnect
	// hs.l.svr.OnDisconnect = hs.l.OnDisconnect

	// go func() {
	// 	hs.l.svr.RunContext(ctx, "tcp", addr)
	// }()

	var listener = &TCPListener{
		Multicore:        hs.Multicore,
		LockOSThread:     hs.LockOSThread,
		NumEventLoop:     hs.NumEventLoop,
		ReuseAddr:        hs.ReuseAddr,
		ReusePort:        hs.ReusePort,
		SocketRecvBuffer: hs.SocketRecvBuffer,
		SocketSendBuffer: hs.SocketSendBuffer,
		TCPKeepAlive:     hs.TCPKeepAlive,
	}

	listener.AsyncStart(ctx, addr)

	return http.Serve(listener, hs.Router)
}

// Stop implements voltron.Adapter
func (hs *HTTPServer) Stop(ctx context.Context) error {
	// return hs.l.CloseContext(ctx)
	return nil
}

// Shutdown implements voltron.Adapter
func (hs *HTTPServer) Shutdown() {
	// hs.l.Shutdown()
}

// AsyncStart implements voltron.Adapter
func (hs *HTTPServer) AsyncStart(ctx context.Context, addr string) {
	hs.wg.Add(1)
	go func() {
		defer hs.wg.Done()
		hs.err = hs.Start(ctx, addr)
	}()
}

// Wait implements voltron.Adapter
func (hs *HTTPServer) Wait() error {
	hs.wg.Wait()
	return hs.err
}
