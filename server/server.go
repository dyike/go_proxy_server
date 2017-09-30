package server

import (
	"bufio"
	"encoding/base64"
	"go_proxy_server/lib"
	"net"
	"strings"
)

// Server struct
type Server struct {
	listener   net.Listener
	addr       string
	credential string
}

// New create a proxy server
func New(Addr, credential string, genAuth bool) *Server {
	if genAuth {
		credential = RandStringBytesMaskImprSrc(16) + ":" + RandStringBytesMaskImprSrc(16)
	}
	return &Server{addr: Addr, credential: base64.StdEncoding.EncodeToString([]byte(credential))}
}

// Start a proxy server
func (s *Server) Start() {
	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		log.Error(err.Error())
	}

	if s.credential != "" {
		log.Info("use %s for auth\n", s.credential)
	}
	log.Info("proxy listen in %s, waiting for connection...\n", s.addr)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Error(err.Error())
			continue
		}
		go s.newConn(conn).serve()
	}
}

// newConn create a conn to serve client request
func (s *Server) newConn(rwc net.Conn) *conn {
	return &conn{
		server: s,
		rwc:    rwc,
		brc:    bufio.NewReader(rwc),
	}
}

// isAuth return weather the client should be authenticate
func (s *Server) isAuth() bool {
	return s.credential != ""
}

// validateCredentials parse "Basic basic-credentials" and validate it
func (s *Server) validateCredential(basicCredential string) bool {
	c := strings.Split(basicCredential, " ")
	if len(c) == 2 && strings.EqualFold(c[0], "Basic") && c[1] == s.credential {
		return true
	}
	return false
}
