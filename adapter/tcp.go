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
	"errors"
	"net"
	"sync"
	"time"

	"github.com/govoltron/layer4"
)

type TCPServerEventHandler interface {

	// OnBoot fires when the server is ready for accepting connections.
	OnBoot()

	// OnShutdown fires when the server is being shut down, it is called right after
	// all event-loops and connections are closed.
	OnShutdown()

	// OnConnect fires when a new connection has been opened.
	//
	// The Conn conn has information about the connection such as its local and remote addresses.
	OnConnect(conn layer4.Conn)

	// OnDisconnect fires when a connection has been closed.
	//
	// The parameter err is the last known connection error.
	OnDisconnect(conn layer4.Conn, err error)

	// OnNewConnection fires when a new connection has been opened.
	OnNewConnection() (handler layer4.ConnEventHandler)
}

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

	// The event handler interface for the server.
	EventHandler TCPServerEventHandler

	// The underlay server.
	svr layer4.Server

	err error

	wg sync.WaitGroup
}

// Start implements voltron.Adapter
func (ts *TCPServer) Start(ctx context.Context, addr string) error {
	if ts.EventHandler == nil {
		panic("invalid server event handler")
	}

	ts.svr.OnBoot = ts.EventHandler.OnBoot
	ts.svr.OnShutdown = ts.EventHandler.OnShutdown
	ts.svr.OnConnect = ts.EventHandler.OnConnect
	ts.svr.OnDisconnect = ts.EventHandler.OnDisconnect
	ts.svr.OnNewConnection = ts.EventHandler.OnNewConnection

	ts.svr.Multicore = ts.Multicore
	ts.svr.LockOSThread = ts.LockOSThread
	ts.svr.NumEventLoop = ts.NumEventLoop
	ts.svr.ReuseAddr = ts.ReuseAddr
	ts.svr.ReusePort = ts.ReusePort
	ts.svr.SocketRecvBuffer = ts.SocketRecvBuffer
	ts.svr.SocketSendBuffer = ts.SocketSendBuffer
	ts.svr.TCPKeepAlive = ts.TCPKeepAlive

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

func (ts *TCPServer) Dup() (dupFD int, err error) {
	return ts.svr.Dup()
}

func (ts *TCPServer) NumConnections() (num int) {
	return ts.svr.NumConnections()
}

// AsyncStart implements voltron.Adapter
func (ts *TCPServer) AsyncStart(ctx context.Context, addr string) {
	ts.wg.Add(1)
	go func() {
		defer ts.wg.Done()
		ts.err = ts.Start(ctx, addr)
	}()
}

// Wait implements voltron.Adapter
func (ts *TCPServer) Wait() error {
	ts.wg.Wait()
	return ts.err
}

type TCPListener struct {

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

	// The underlay server.
	svr layer4.Server

	err error

	wg sync.WaitGroup

	addr     string
	pipeline chan net.Conn
}

func (tl *TCPListener) AsyncStart(ctx context.Context, addr string) {
	tl.svr.Multicore = tl.Multicore
	tl.svr.LockOSThread = tl.LockOSThread
	tl.svr.NumEventLoop = tl.NumEventLoop
	tl.svr.ReuseAddr = tl.ReuseAddr
	tl.svr.ReusePort = tl.ReusePort
	tl.svr.SocketRecvBuffer = tl.SocketRecvBuffer
	tl.svr.SocketSendBuffer = tl.SocketSendBuffer
	tl.svr.TCPKeepAlive = tl.TCPKeepAlive
	tl.svr.OnConnect = func(conn layer4.Conn) { tl.pipeline <- conn }

	tl.addr = addr
	tl.pipeline = make(chan net.Conn, 10240)

	tl.wg.Add(1)
	go func() {
		defer func() {
			close(tl.pipeline)
			tl.wg.Done()
		}()
		tl.err = tl.svr.RunContext(ctx, "tcp", addr)
	}()
}

func (tl *TCPListener) Wait() error {
	tl.wg.Wait()
	return tl.err
}

// Accept implements net.Listener
func (tl *TCPListener) Accept() (net.Conn, error) {
	conn, ok := <-tl.pipeline
	if !ok {
		return nil, errors.New("pipeline is closed")
	}
	return conn, nil
}

// Addr implements net.Listener
func (tl *TCPListener) Addr() (addr net.Addr) {
	addr, _ = net.ResolveTCPAddr("tcp", tl.addr)
	return
}

// Close implements net.Listener
func (tl *TCPListener) Close() error {
	return tl.svr.Stop(context.TODO())
}
