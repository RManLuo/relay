package relay

import (
	"fmt"
	"net"
	"time"
)

// RunTCPServer starts tcp server.
func (s *Relay) RunTCPServer() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		fmt.Print(s.Local, "<=>", s.Remote, err)
		return err
	}
	defer s.TCPListen.Close()
	count := 0
	for {
		c, err := s.TCPListen.AcceptTCP()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			count++
			if count > 10 {
				break
			}
			continue
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
