// Copyright 2021 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package netpoll

import (
	"context"
	"net"
	"time"
)

// Dialer extends net.Dialer's API, just for interface compatibility.
// DialConnection is recommended, but of course all functions are practically the same.
// The returned net.Conn can be directly asserted as Connection if error is nil.
type Dialer interface {
	DialConnection(network, address string, timeout time.Duration, ctx ...context.Context) (connection Connection, err error)

	DialTimeout(network, address string, timeout time.Duration, ctx ...context.Context) (conn net.Conn, err error)
}

// DialConnection is a default implementation of Dialer.
func DialConnection(network, address string, timeout time.Duration) (connection Connection, err error) {
	return defaultDialer.DialConnection(network, address, timeout)
}

// NewDialer only support TCP and unix socket now.
func NewDialer() Dialer {
	return &dialer{}
}

var defaultDialer = NewDialer()

type dialer struct{}

// DialTimeout implements Dialer.
func (d *dialer) DialTimeout(network, address string, timeout time.Duration, ctx ...context.Context) (net.Conn, error) {
    var conn net.Conn
    var err error
    if len(ctx) > 0{
        conn, err = d.DialConnection(network, address, timeout, ctx[0])
    } else{
        conn, err = d.DialConnection(network, address, timeout)
    }
    return conn, err
}

// DialConnection implements Dialer.
func (d *dialer) DialConnection(network, address string, timeout time.Duration, ctxPrev ...context.Context) (connection Connection, err error) {
	ctx := context.Background()
	if len(ctxPrev) > 0{
		if val, ok := ctxPrev[0].Value("DSCP").(int); ok{
			ctx = context.WithValue(ctx, "DSCP", val)
		}
		if val, ok := ctxPrev[0].Value("MARK").(int); ok{
			ctx = context.WithValue(ctx, "MARK", val)
		}
	}
	if timeout > 0 {
		subCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		ctx = subCtx
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		var raddr *TCPAddr
		raddr, err = ResolveTCPAddr(network, address)
		if err != nil {
			return nil, err
		}
		connection, err = DialTCP(ctx, network, nil, raddr)
	// case "udp", "udp4", "udp6":  // TODO: unsupport now
	case "unix", "unixgram", "unixpacket":
		var raddr *UnixAddr
		raddr, err = ResolveUnixAddr(network, address)
		if err != nil {
			return nil, err
		}
		connection, err = DialUnix(network, nil, raddr)
	default:
		return nil, net.UnknownNetworkError(network)
	}
	return connection, err
}

// sysDialer contains a Dial's parameters and configuration.
type sysDialer struct {
	net.Dialer
	network, address string
}
