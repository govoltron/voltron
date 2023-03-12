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
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	"github.com/govoltron/layer4"
)

type listener struct {
	addr string
	svr  layer4.Server
	pipe chan layer4.Conn
}

// OnBoot implements TCPServerEventHandler
func (l *listener) OnBoot() {
	l.pipe = make(chan layer4.Conn, 10240)
}

// OnShutdown implements TCPServerEventHandler
func (l *listener) OnShutdown() {
	close(l.pipe)
}

// OnConnect implements TCPServerEventHandler
func (l *listener) OnConnect(conn layer4.Conn) {
	l.pipe <- conn
}

// OnDisconnect implements TCPServerEventHandler
func (l *listener) OnDisconnect(conn layer4.Conn, err error) {

}

// Accept implements net.Listener
func (l *listener) Accept() (net.Conn, error) {
	conn, ok := <-l.pipe
	if !ok {
		return nil, errors.New("already shutdown")
	}
	return conn, nil
}

// Addr implements net.Listener
func (l *listener) Addr() (addr net.Addr) {
	addr, _ = net.ResolveTCPAddr("tcp", l.addr)
	return
}

// Close implements net.Listener
func (l *listener) Close() error {
	return l.svr.Stop(context.TODO())
}

func (l *listener) CloseContext(ctx context.Context) error {
	return l.svr.Stop(ctx)
}

func (l *listener) Shutdown() {
	l.svr.Shutdown()
}

var (
	_ net.Listener = &listener{}
)

type HTTPServer struct {
	l      listener
	err    error
	wg     sync.WaitGroup
	Router chi.Router
}

// Start implements voltron.Adapter
func (hs *HTTPServer) Start(ctx context.Context, addr string) error {
	hs.l.addr = addr
	hs.l.svr.OnBoot = hs.l.OnBoot
	hs.l.svr.OnShutdown = hs.l.OnShutdown
	hs.l.svr.OnConnect = hs.l.OnConnect
	hs.l.svr.OnDisconnect = hs.l.OnDisconnect

	return http.Serve(&hs.l, hs.Router)
}

// Stop implements voltron.Adapter
func (hs *HTTPServer) Stop(ctx context.Context) error {
	return hs.l.CloseContext(ctx)
}

// Shutdown implements voltron.Adapter
func (hs *HTTPServer) Shutdown() {
	hs.l.Shutdown()
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
