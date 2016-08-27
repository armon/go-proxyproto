package proxyproto

import (
	"bytes"
	"net"
	"testing"

	proto "github.com/pires/go-proxyproto"
)

func TestPassthrough(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	pl := &Listener{Listener: l}

	go func() {
		conn, err := net.Dial("tcp", pl.Addr().String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer conn.Close()

		conn.Write([]byte("ping"))
		recv := make([]byte, 4)
		_, err = conn.Read(recv)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !bytes.Equal(recv, []byte("pong")) {
			t.Fatalf("bad: %v", recv)
		}
	}()

	conn, err := pl.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer conn.Close()

	recv := make([]byte, 4)
	_, err = conn.Read(recv)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !bytes.Equal(recv, []byte("ping")) {
		t.Fatalf("bad: %v", recv)
	}

	if _, err := conn.Write([]byte("pong")); err != nil {
		t.Fatalf("err: %v", err)
	}
}

//func TestTimeout(t *testing.T) {
//	l, err := net.Listen("tcp", "127.0.0.1:0")
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//
//	clientWriteDelay := 200 * time.Millisecond
//	proxyHeaderTimeout := 50 * time.Millisecond
//	pl := &Listener{Listener: l, ProxyHeaderTimeout: proxyHeaderTimeout}
//
//	go func() {
//		conn, err := net.Dial("tcp", pl.Addr().String())
//		if err != nil {
//			t.Fatalf("err: %v", err)
//		}
//		defer conn.Close()
//
//		// Do not send data for a while
//		time.Sleep(clientWriteDelay)
//
//		conn.Write([]byte("ping"))
//		recv := make([]byte, 4)
//		_, err = conn.Read(recv)
//		if err != nil {
//			t.Fatalf("err: %v", err)
//		}
//		if !bytes.Equal(recv, []byte("pong")) {
//			t.Fatalf("bad: %v", recv)
//		}
//	}()
//
//	conn, err := pl.Accept()
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//	defer conn.Close()
//
//	// Check the remote addr is the original 127.0.0.1
//	remoteAddrStartTime := time.Now()
//	addr := conn.RemoteAddr().(*net.TCPAddr)
//	if addr.IP.String() != "127.0.0.1" {
//		t.Fatalf("bad: %v", addr)
//	}
//	remoteAddrDuration := time.Since(remoteAddrStartTime)
//
//	// Check RemoteAddr() call did timeout
//	if remoteAddrDuration >= clientWriteDelay {
//		t.Fatalf("RemoteAddr() took longer than the specified timeout: %v < %v", proxyHeaderTimeout, remoteAddrDuration)
//	}
//
//	recv := make([]byte, 4)
//	_, err = conn.Read(recv)
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//	if !bytes.Equal(recv, []byte("ping")) {
//		t.Fatalf("bad: %v", recv)
//	}
//
//	if _, err := conn.Write([]byte("pong")); err != nil {
//		t.Fatalf("err: %v", err)
//	}
//}

func TestParse_ipv4(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	pl := &Listener{Listener: l}

	go func() {
		conn, err := net.Dial("tcp", pl.Addr().String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer conn.Close()

		// Write out the header!
		header := "PROXY TCP4 10.1.1.1 20.2.2.2 1000 2000\r\n"
		conn.Write([]byte(header))

		conn.Write([]byte("ping"))
		recv := make([]byte, 4)
		_, err = conn.Read(recv)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !bytes.Equal(recv, []byte("pong")) {
			t.Fatalf("bad: %v", recv)
		}
	}()

	conn, err := pl.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer conn.Close()

	recv := make([]byte, 4)
	_, err = conn.Read(recv)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !bytes.Equal(recv, []byte("ping")) {
		t.Fatalf("bad: %v", recv)
	}

	if _, err := conn.Write([]byte("pong")); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the remote addr
	addr := conn.RemoteAddr().(*net.TCPAddr)
	if addr.IP.String() != "10.1.1.1" {
		t.Fatalf("bad: %v", addr)
	}
	if addr.Port != 1000 {
		t.Fatalf("bad: %v", addr)
	}
}

func TestParse_ipv6(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	pl := &Listener{Listener: l}

	go func() {
		conn, err := net.Dial("tcp", pl.Addr().String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer conn.Close()

		// Write out the header!
		header := "PROXY TCP6 ffff::ffff ffff::ffff 1000 2000\r\n"
		conn.Write([]byte(header))

		conn.Write([]byte("ping"))
		recv := make([]byte, 4)
		_, err = conn.Read(recv)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !bytes.Equal(recv, []byte("pong")) {
			t.Fatalf("bad: %v", recv)
		}
	}()

	conn, err := pl.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer conn.Close()

	recv := make([]byte, 4)
	_, err = conn.Read(recv)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !bytes.Equal(recv, []byte("ping")) {
		t.Fatalf("bad: %v", recv)
	}

	if _, err := conn.Write([]byte("pong")); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the remote addr
	addr := conn.RemoteAddr().(*net.TCPAddr)
	if addr.IP.String() != "ffff::ffff" {
		t.Fatalf("bad: %v", addr)
	}
	if addr.Port != 1000 {
		t.Fatalf("bad: %v", addr)
	}
}

func TestParse_ipv4_protov2(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	pl := &Listener{Listener: l}

	go func() {
		conn, err := net.Dial("tcp", pl.Addr().String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer conn.Close()

		// Write out the header!
		header := &proto.Header{
			Version:            2,
			Command:            proto.PROXY,
			TransportProtocol:  proto.TCPv4,
			SourceAddress:      net.ParseIP("10.1.1.1"),
			DestinationAddress: net.ParseIP("20.2.2.2"),
			SourcePort:         1000,
			DestinationPort:    2000,
		}
		header.WriteTo(conn)

		conn.Write([]byte("ping"))
		recv := make([]byte, 4)
		_, err = conn.Read(recv)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !bytes.Equal(recv, []byte("pong")) {
			t.Fatalf("bad: %v", recv)
		}
	}()

	conn, err := pl.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer conn.Close()

	recv := make([]byte, 4)
	_, err = conn.Read(recv)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !bytes.Equal(recv, []byte("ping")) {
		t.Fatalf("bad: %v", recv)
	}

	if _, err := conn.Write([]byte("pong")); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the remote addr
	addr := conn.RemoteAddr().(*net.TCPAddr)
	if addr.IP.String() != "10.1.1.1" {
		t.Fatalf("bad: %v", addr)
	}
	if addr.Port != 1000 {
		t.Fatalf("bad: %v", addr)
	}
}

func TestParse_ipv6_protov2(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	pl := &Listener{Listener: l}

	go func() {
		conn, err := net.Dial("tcp", pl.Addr().String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer conn.Close()

		// Write out the header!
		header := &proto.Header{
			Version:            2,
			Command:            proto.PROXY,
			TransportProtocol:  proto.TCPv6,
			SourceAddress:      net.ParseIP("::1"),
			DestinationAddress: net.ParseIP("::2"),
			SourcePort:         1000,
			DestinationPort:    2000,
		}
		header.WriteTo(conn)

		conn.Write([]byte("ping"))
		recv := make([]byte, 4)
		_, err = conn.Read(recv)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !bytes.Equal(recv, []byte("pong")) {
			t.Fatalf("bad: %v", recv)
		}
	}()

	conn, err := pl.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer conn.Close()

	recv := make([]byte, 4)
	_, err = conn.Read(recv)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !bytes.Equal(recv, []byte("ping")) {
		t.Fatalf("bad: %v", recv)
	}

	if _, err := conn.Write([]byte("pong")); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the remote addr
	addr := conn.RemoteAddr().(*net.TCPAddr)
	if addr.IP.String() != "::1" {
		t.Fatalf("bad: %v", addr)
	}
	if addr.Port != 1000 {
		t.Fatalf("bad: %v", addr)
	}
}

// TODO v2 UDP test
//func TestParse_ipv6_udp_protov2(t *testing.T) {
//	saddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
//	l, err := net.ListenUDP("udp", saddr)
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//
//	pl := &Listener{Listener: l}
//
//	go func() {
//		laddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
//		conn, err := net.DialUDP("udp", laddr, pl.Addr().String())
//		if err != nil {
//			t.Fatalf("err: %v", err)
//		}
//		defer conn.Close()
//
//		// Write out the header!
//		header := &proto.Header{
//			Version: 2,
//			Command:            proto.PROXY,
//			TransportProtocol:  proto.UDPv6,
//			SourceAddress:      net.ParseIP("::1"),
//			DestinationAddress: net.ParseIP("::2"),
//			SourcePort:         1000,
//			DestinationPort:    2000,
//		}
//		header.WriteTo(conn)
//
//		conn.Write([]byte("ping"))
//		recv := make([]byte, 4)
//		_, err = conn.Read(recv)
//		if err != nil {
//			t.Fatalf("err: %v", err)
//		}
//		if !bytes.Equal(recv, []byte("pong")) {
//			t.Fatalf("bad: %v", recv)
//		}
//	}()
//
//	conn, err := pl.Accept()
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//	defer conn.Close()
//
//	recv := make([]byte, 4)
//	_, err = conn.Read(recv)
//	if err != nil {
//		t.Fatalf("err: %v", err)
//	}
//	if !bytes.Equal(recv, []byte("ping")) {
//		t.Fatalf("bad: %v", recv)
//	}
//
//	if _, err := conn.Write([]byte("pong")); err != nil {
//		t.Fatalf("err: %v", err)
//	}
//
//	// Check the remote addr
//	addr := conn.RemoteAddr().(*net.UDPAddr)
//	if addr.IP.String() != "::1" {
//		t.Fatalf("bad: %v", addr)
//	}
//}

func TestParse_BadHeader(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	pl := &Listener{Listener: l}

	go func() {
		conn, err := net.Dial("tcp", pl.Addr().String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer conn.Close()

		// Write out the header!
		header := "PROXY TCP4 what 127.0.0.1 1000 2000\r\n"
		conn.Write([]byte(header))

		conn.Write([]byte("ping"))

		recv := make([]byte, 4)
		_, err = conn.Read(recv)
		if err == nil {
			t.Fatalf("err: %v", err)
		}
	}()

	conn, err := pl.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer conn.Close()

	// Check the remote addr, should be the local addr
	addr := conn.RemoteAddr().(*net.TCPAddr)
	if addr.IP.String() != "127.0.0.1" {
		t.Fatalf("bad: %v", addr)
	}

	// Read should fail
	recv := make([]byte, 4)
	_, err = conn.Read(recv)
	if err == nil {
		t.Fatalf("err: %v", err)
	}
}
