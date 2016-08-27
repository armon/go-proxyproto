package proxyproto

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"

	proto "github.com/pires/go-proxyproto"
)

// Listener is used to wrap an underlying listener,
// whose connections may be using the HAProxy Proxy Protocol (version 1).
// If the connection is using the protocol, the RemoteAddr() will return
// the correct client address.
//
// Optionally define ProxyHeaderTimeout to set a maximum time to
// receive the Proxy Protocol Header. Zero means no timeout.
type Listener struct {
	Listener           net.Listener
	ProxyHeaderTimeout time.Duration
}

// Conn is used to wrap and underlying connection which
// may be speaking the Proxy Protocol. If it is, the RemoteAddr() will
// return the address of the client instead of the proxy address.
type Conn struct {
	bufReader          *bufio.Reader
	conn               net.Conn
	header             *proto.Header
	once               sync.Once
	proxyHeaderTimeout time.Duration
}

// Accept waits for and returns the next connection to the listener.
func (p *Listener) Accept() (net.Conn, error) {
	// Get the underlying connection
	conn, err := p.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return NewConn(conn, p.ProxyHeaderTimeout), nil
}

// Close closes the underlying listener.
func (p *Listener) Close() error {
	return p.Listener.Close()
}

// Addr returns the underlying listener's network address.
func (p *Listener) Addr() net.Addr {
	return p.Listener.Addr()
}

// NewConn is used to wrap a net.Conn that may be speaking
// the proxy protocol into a proxyproto.Conn
func NewConn(conn net.Conn, timeout time.Duration) *Conn {
	pConn := &Conn{
		bufReader:          bufio.NewReader(conn),
		conn:               conn,
		proxyHeaderTimeout: timeout,
	}
	return pConn
}

// Read is check for the proxy protocol header when doing
// the initial scan. If there is an error parsing the header,
// it is returned and the socket is closed.
func (p *Conn) Read(b []byte) (int, error) {
	var err error
	p.once.Do(func() { err = p.checkHeader() })
	if err != nil {
		// If no proxy protocol header is present, the connection is still valid.
		if err == proto.ErrNoProxyProtocol {
			log.Printf("[WARN] Failed to read proxy protocol header: %v", err)
		} else {
			return 0, err
		}
	}
	return p.bufReader.Read(b)
}

func (p *Conn) Write(b []byte) (int, error) {
	return p.conn.Write(b)
}

func (p *Conn) Close() error {
	return p.conn.Close()
}

func (p *Conn) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}

// RemoteAddr returns the address of the client if the proxy
// protocol is being used, otherwise just returns the address of
// the socket peer. If there is an error parsing the header, the
// address of the client is not returned, and the socket is closed.
// Once implication of this is that the call could block if the
// client is slow. Using a Deadline is recommended if this is called
// before Read()
func (p *Conn) RemoteAddr() net.Addr {
	p.once.Do(func() {
		if err := p.checkHeader(); err != nil && err != proto.ErrNoProxyProtocol {
			log.Printf("[ERR] Failed to read proxy prefix: %v", err)
			p.Close()
			p.bufReader = bufio.NewReader(p.conn)
		}
	})
	if p.header != nil && p.header.Command.IsProxy() {
		if p.header.TransportProtocol.IsStream() {
			return &net.TCPAddr{
				IP:   p.header.SourceAddress,
				Port: int(p.header.SourcePort),
			}
		} else if p.header.TransportProtocol.IsDatagram() {
			return &net.UDPAddr{
				IP:   p.header.SourceAddress,
				Port: int(p.header.SourcePort),
			}
		}
	}
	return p.conn.RemoteAddr()
}

func (p *Conn) SetDeadline(t time.Time) error {
	return p.conn.SetDeadline(t)
}

func (p *Conn) SetReadDeadline(t time.Time) error {
	return p.conn.SetReadDeadline(t)
}

func (p *Conn) SetWriteDeadline(t time.Time) error {
	return p.conn.SetWriteDeadline(t)
}

func (p *Conn) checkHeader() (err error) {
	if p.proxyHeaderTimeout != 0 {
		readDeadLine := time.Now().Add(p.proxyHeaderTimeout)
		p.conn.SetReadDeadline(readDeadLine)
		defer p.conn.SetReadDeadline(time.Time{})
	}

	// TODO golden hammer against blocking forever
	if p.proxyHeaderTimeout == 0 {
		p.proxyHeaderTimeout = 50 * time.Millisecond
	}

	p.header, err = proto.ReadTimeout(p.bufReader, p.proxyHeaderTimeout)

	return
}
