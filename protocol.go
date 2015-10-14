// Package proxoproto implements a net.Listener supporting HAProxy PROTO protocol.
//
// See http://www.haproxy.org/download/1.5/doc/proxy-protocol.txt for details.
package proxyproto

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// prefix is the string we look for at the start of a connection
// to check if this connection is using the proxy protocol.
var prefix = []byte("PROXY ")

// Listener wraps an underlying listener whose connections may be
// using the HAProxy Proxy Protocol (version 1).
// If the connection is using the protocol, RemoteAddr will return the
// correct client address.
type Listener struct {
	Listener net.Listener

	initOnce sync.Once        // guards init, which sets the following:
	connc    chan interface{} // *conn or error
}

func (p *Listener) init() {
	p.connc = make(chan interface{})
}

// conn is used to wrap and underlying connection which
// may be speaking the Proxy Protocol. If it is, the RemoteAddr() will
// return the address of the client instead of the proxy address.
type conn struct {
	bufReader *bufio.Reader
	conn      net.Conn
	dstAddr   *net.TCPAddr
	srcAddr   *net.TCPAddr
	once      sync.Once
}

// Accept waits for and returns the next connection to the listener.
func (p *Listener) Accept() (net.Conn, error) {
	p.initOnce.Do(p.init)
	// Get the underlying connection
	rawc, err := p.Listener.Accept()
	if err != nil {
		return nil, err
	}
	go func() {
		c, err := newConn(rawc)
		if err != nil {
			p.connc <- err
		} else {
			p.connc <- c
		}
	}()
	v := <-p.connc
	if c, ok := v.(*conn); ok {
		return c, nil
	} else {
		return nil, v.(error)
	}
}

// Close closes the underlying listener.
func (p *Listener) Close() error { return p.Listener.Close() }

// Addr returns the underlying listener's network address.
func (p *Listener) Addr() net.Addr { return p.Listener.Addr() }

func newConn(c net.Conn) (*conn, error) {
	pc := &conn{
		bufReader: bufio.NewReader(c),
		conn:      c,
	}
	if err := pc.checkPrefix(); err != nil {
		return nil, err
	}
	return pc, nil
}

// Read is check for the proxy protocol header when doing
// the initial scan. If there is an error parsing the header,
// it is returned and the socket is closed.
func (p *conn) Read(b []byte) (int, error) {
	var err error
	p.once.Do(func() { err = p.checkPrefix() })
	if err != nil {
		return 0, err
	}
	return p.bufReader.Read(b)
}

func (p *conn) Write(b []byte) (int, error) {
	return p.conn.Write(b)
}

func (p *conn) Close() error {
	return p.conn.Close()
}

func (p *conn) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}

// RemoteAddr returns the address of the client if the proxy
// protocol is being used, otherwise just returns the address of
// the socket peer. If there is an error parsing the header, the
// address of the client is not returned, and the socket is closed.
// Once implication of this is that the call could block if the
// client is slow. Using a Deadline is recommended if this is called
// before Read()
func (p *conn) RemoteAddr() net.Addr {
	if p.srcAddr != nil {
		return p.srcAddr
	}
	return p.conn.RemoteAddr()
}

func (p *conn) SetDeadline(t time.Time) error      { return p.conn.SetDeadline(t) }
func (p *conn) SetReadDeadline(t time.Time) error  { return p.conn.SetReadDeadline(t) }
func (p *conn) SetWriteDeadline(t time.Time) error { return p.conn.SetWriteDeadline(t) }

func (p *conn) checkPrefix() error {
	// Incrementally check each byte of the prefix
	for i := 1; i <= len(prefix); i++ {
		inp, err := p.bufReader.Peek(i)
		if err != nil {
			// TOOD: this isn't right. it returns EOF on
			// non-PROXY connections sending payloads
			// shorter than len(prefix).
			return err
		}

		// Check for a prefix mis-match, quit early
		if !bytes.Equal(inp, prefix[:i]) {
			return nil
		}
	}

	// Read the header line
	header, err := p.bufReader.ReadString('\n')
	if err != nil {
		p.conn.Close()
		return err
	}

	// Strip the carriage return and new line
	header = header[:len(header)-2]

	// Split on spaces, should be (PROXY <type> <src addr> <dst addr> <src port> <dst port>)
	parts := strings.Split(header, " ")
	if len(parts) != 6 {
		p.conn.Close()
		return fmt.Errorf("Invalid header line: %s", header)
	}

	// Verify the type is known
	switch parts[1] {
	case "TCP4":
	case "TCP6":
	default:
		p.conn.Close()
		return fmt.Errorf("Unhandled address type: %s", parts[1])
	}

	// Parse out the source address
	ip := net.ParseIP(parts[2])
	if ip == nil {
		p.conn.Close()
		return fmt.Errorf("Invalid source ip: %s", parts[2])
	}
	port, err := strconv.Atoi(parts[4])
	if err != nil {
		p.conn.Close()
		return fmt.Errorf("Invalid source port: %s", parts[4])
	}
	p.srcAddr = &net.TCPAddr{IP: ip, Port: port}

	// Parse out the destination address
	ip = net.ParseIP(parts[3])
	if ip == nil {
		p.conn.Close()
		return fmt.Errorf("Invalid destination ip: %s", parts[3])
	}
	port, err = strconv.Atoi(parts[5])
	if err != nil {
		p.conn.Close()
		return fmt.Errorf("Invalid destination port: %s", parts[5])
	}
	p.dstAddr = &net.TCPAddr{IP: ip, Port: port}

	return nil
}
