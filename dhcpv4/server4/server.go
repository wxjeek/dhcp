package server4

import (
	"log"
	"net"
	"sync"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

/*
  To use the DHCPv4 server code you have to call NewServer with two arguments:
  - a handler function, that will be called every time a valid DHCPv4 packet is
    received, and
  - an address to listen on.

  The handler is a function that takes as input a packet connection, that can be
  used to reply to the client; a peer address, that identifies the client sending
  the request, and the DHCPv4 packet itself. Just implement your custom logic in
  the handler.

  The address to listen on is used to know IP address, port and optionally the
  scope to create and UDP socket to listen on for DHCPv4 traffic.

  Example program:


package main

import (
	"log"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

func handler(conn net.PacketConn, peer net.Addr, m *dhcpv4.DHCPv4) {
	// this function will just print the received DHCPv4 message, without replying
	log.Print(m.Summary())
}

func main() {
	laddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 67,
	}
	server, err := dhcpv4.NewServer(laddr, handler)
	if err != nil {
		log.Fatal(err)
	}

	// This never returns. If you want to do other stuff, dump it into a
	// goroutine.
	server.Serve()
}

*/

// Handler is a type that defines the handler function to be called every time a
// valid DHCPv4 message is received
type Handler func(conn net.PacketConn, peer net.Addr, m *dhcpv4.DHCPv4)

// Server represents a DHCPv4 server object
type Server struct {
	conn       net.PacketConn
	connMutex  sync.Mutex
	shouldStop chan bool
	Handler    Handler
}

// Serve serves requests.
func (s *Server) Serve() {
	log.Printf("Server listening on %s", s.conn.LocalAddr())
	log.Print("Ready to handle requests")
	for {
		rbuf := make([]byte, 4096) // FIXME this is bad
		n, peer, err := s.conn.ReadFrom(rbuf)
		if err != nil {
			log.Printf("Error reading from packet conn: %v", err)
			return
		}
		log.Printf("Handling request from %v", peer)

		m, err := dhcpv4.FromBytes(rbuf[:n])
		if err != nil {
			log.Printf("Error parsing DHCPv4 request: %v", err)
			continue
		}
		go s.Handler(s.conn, peer, m)
	}
}

// Close sends a termination request to the server, and closes the UDP listener
func (s *Server) Close() error {
	return s.conn.Close()
}

// ServerOpt adds optional configuration to a server.
type ServerOpt func(s *Server)

// WithConn configures the server with the given connection.
func WithConn(c net.PacketConn) ServerOpt {
	return func(s *Server) {
		s.conn = c
	}
}

// NewServer initializes and returns a new Server object
func NewServer(addr *net.UDPAddr, handler Handler, opt ...ServerOpt) (*Server, error) {
	s := &Server{
		Handler:    handler,
		shouldStop: make(chan bool, 1),
	}

	for _, o := range opt {
		o(s)
	}
	if s.conn == nil {
		var err error
		s.conn, err = net.ListenUDP("udp4", addr)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}