package server

import (
	"bufio"
	"bytes"
	"fmt"
	"go_proxy_server/log"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"strings"
)

type conn struct {
	rwc    net.Conn
	brc    *bufio.Reader
	server *Server
}

func (c *conn) serve() {
	defer c.rwc.Close()
	rawHTTPRequestHeader, remote, credential, isHTTPS, err := c.getTunnelInfo()
	if err != nil {
		log.Error(err.Error())
		return
	}

	if c.auth(credential) == false {
		log.Error("Auth fail: " + credential)
		return
	}

	log.Info("connecting to " + remote)

	remoteConn, err := net.Dial("tcp", remote)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if isHTTPS {
		_, err = c.rwc.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			log.Error(err.Error())
			return
		}
	} else {
		_, err = rawHTTPRequestHeader.WriteTo(remoteConn)
		if err != nil {
			log.Error(err.Error())
			return
		}
	}
	// build bidirectional-streams
	log.Info("begin tunnel", c.rwc.RemoteAddr(), "<->", remote)
	c.tunnel(remoteConn)
	log.Info("stop tunnel", c.rwc.RemoteAddr(), "<->", remote)

}

// getClientInfo parse client request header to get some information:
func (c *conn) getTunnelInfo() (rawRequestHeader bytes.Buffer, host, credential string, isHTTPS bool, err error) {
	tp := textproto.NewReader(c.brc)

	// First line: GET /index.html HTTP/1.0
	var requestLine string
	if requestLine, err = tp.ReadLine(); err != nil {
		return
	}

	method, requestURI, _, ok := parseRequestLine(requestLine)
	if !ok {
		err = &BadRequestError{"malformed HTTP request"}
		return
	}

	// https request
	if method == "CONNECT" {
		isHTTPS = true
		requestURI = "http://" + requestURI
	}
	// get remote host
	uriInfo, err := url.ParseRequestURI(requestURI)
	if err != nil {
		return
	}

	// Subsequent lines: Key: value.
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return
	}
	credential = mimeHeader.Get("Proxy-Authorization")
	if uriInfo.Host == "" {
		host = mimeHeader.Get("Host")
	} else {
		if strings.Index(uriInfo.Host, ":") == -1 {
			host = uriInfo.Host + ":80"
		} else {
			host = uriInfo.Host
		}
	}
	// rebuild http request header
	rawRequestHeader.WriteString(requestLine + "\r\n")
	for k, vs := range mimeHeader {
		for _, v := range vs {
			rawRequestHeader.WriteString(fmt.Sprintf("%s:%s\r\n", k, v))
		}
	}
	rawRequestHeader.WriteString("\r\n")
	return
}

func (c *conn) auth(credential string) bool {
	if c.server.isAuth() == false || c.server.validateCredential(credential) {
		return true
	}
	// 407
	_, err := c.rwc.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"*\"\r\n\r\n"))
	if err != nil {
		log.Error(err.Error())
	}
	return false
}

func (c *conn) tunnel(remoteConn net.Conn) {
	go func() {
		_, err := c.brc.WriteTo(remoteConn)
		if err != nil {
			log.Warn(err.Error())
		}
		remoteConn.Close()
	}()
	_, err := io.Copy(c.rwc, remoteConn)
	if err != nil {
		log.Warn(err.Error())
	}
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

type BadRequestError struct {
	something string
}

func (b *BadRequestError) Error() string {
	return b.something
}
