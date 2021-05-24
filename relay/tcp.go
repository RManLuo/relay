package relay

import (
	"fmt"
	"net"
	"time"
)

func (s *Relay) RunTCPServer() error {
	var err error
	wait := 2.0
	for s.TCPListen == nil {
		s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
		if err != nil {
			fmt.Println("Listen TCP", s.Local, err, "(retry in", wait, "s)")
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 1.1
		}
	}
	defer s.TCPListen.Close()
	wait = 1.0
	for s.TCPListen != nil {
		c, err := s.TCPListen.AcceptTCP()
		if err == nil {
			go s.TCPHandle(c)
			wait = 1.0
		} else {
			fmt.Println("Accept", s.Local, err)
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 1.1
		}
	}
	return nil
}

func (s *Relay) TCPHandle(c *net.TCPConn) error {
	defer c.Close()
	rc, err := net.DialTimeout("tcp", s.Remote, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial TCP", s.Local, "<=>", s.Remote, err)
		return err
	}
	defer rc.Close()
	go Copy(c, rc, s.Traffic)
	Copy(rc, c, s.Traffic)

	return nil
}
