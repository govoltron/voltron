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
	"time"

	"github.com/govoltron/layer4"
)

type TCPServer struct {

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

	// OnBoot fires when the server is ready for accepting connections.
	OnBoot func()

	// OnShutdown fires when the server is being shut down, it is called right after
	// all event-loops and connections are closed.
	OnShutdown func()

	// OnConnect fires when a new connection has been opened.
	//
	// The Conn conn has information about the connection such as its local and remote addresses.
	OnConnect func(conn layer4.Conn)

	// OnDisconnect fires when a connection has been closed.
	//
	// The parameter err is the last known connection error.
	OnDisconnect func(conn layer4.Conn, err error)

	// OnNewConnection fires when a new connection has been opened.
	OnNewConnection func() (handler layer4.ConnEventHandler)

	ctx context.Context

	// The underlay server.
	svr layer4.Server
}

// Start implements voltron.Adapter
func (ts *TCPServer) Start(ctx context.Context, addr string) error {
	ts.svr.Multicore = ts.Multicore
	ts.svr.LockOSThread = ts.LockOSThread
	ts.svr.NumEventLoop = ts.NumEventLoop
	ts.svr.ReuseAddr = ts.ReuseAddr
	ts.svr.ReusePort = ts.ReusePort
	ts.svr.SocketRecvBuffer = ts.SocketRecvBuffer
	ts.svr.SocketSendBuffer = ts.SocketSendBuffer
	ts.svr.TCPKeepAlive = ts.TCPKeepAlive

	ts.svr.OnBoot = ts.OnBoot
	ts.svr.OnShutdown = ts.OnShutdown
	ts.svr.OnConnect = ts.OnConnect
	ts.svr.OnDisconnect = ts.OnDisconnect
	ts.svr.OnNewConnection = ts.OnNewConnection

	var (
		cancelFunc func()
	)
	ts.ctx, cancelFunc = context.WithCancel(context.TODO())

	defer func() {
		cancelFunc()
	}()

	return ts.svr.RunContext(ctx, "tcp", addr)
}

// Stop implements voltron.Adapter
func (ts *TCPServer) Stop(ctx context.Context) error {
	return ts.svr.Stop(ctx)
}

// Shutdown implements voltron.Adapter
func (ts *TCPServer) Shutdown() {
	ts.svr.Shutdown()
}

// Wait implements voltron.Adapter
func (ts *TCPServer) Wait() {
	if ts.ctx == nil {
		return
	}
	var (
		ticker = time.NewTicker(time.Millisecond)
	)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ts.ctx.Done():
			return
		case <-ticker.C:
			//
		}
	}
}

// Ready implements voltron.Adapter
func (ts *TCPServer) Ready() bool {
	if ts.ctx == nil {
		return false
	}
	select {
	case <-ts.ctx.Done():
		return false
	default:
		break
	}
	return true
}
