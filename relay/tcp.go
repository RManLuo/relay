package relay

import (
	"net"
	"time"
)

// RunTCPServer starts tcp server.
func (s *Relay) RunTCPServer() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return err
	}
	defer s.TCPListen.Close()
	for {
		c, err := s.TCPListen.AcceptTCP()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			break
		}
		go s.TCPHandle(c)
	}
	return nil
}

// TCPHandle handles request.
func (s *Relay) TCPHandle(c *net.TCPConn) error {
	defer c.Close()
	rc, err := net.DialTimeout("tcp", s.Remote, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		return err
	}
	defer rc.Close()
	go Copy(c, rc, s.Traffic)
	Copy(rc, c, s.Traffic)

	return nil
}
