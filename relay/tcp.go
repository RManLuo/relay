package relay

import (
	"fmt"
	"net"
	"time"
)

func (s *Relay) ListenTCP() (err error) {
	wait := 2.0
	for s.Status == 1 && s.TCPListen == nil {
		s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
		if err != nil {
			fmt.Println("Listen TCP", s.Laddr, err, "(retry in", wait, "s)")
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 1.1
		}
	}
	return
}

func (s *Relay) AcceptAndHandleTCP(handle func(c *net.TCPConn) error) error {
	wait := 1.0
	for s.Status == 1 && s.TCPListen != nil {
		c, err := s.TCPListen.AcceptTCP()
		if err == nil {
			go handle(c)
			wait = 1.0
		} else {
			fmt.Println("Accept", s.Laddr, err)
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 1.1
		}
	}
	return nil
}

func (s *Relay) RunTCPServer() error {
	s.ListenTCP()
	defer s.TCPListen.Close()
	s.AcceptAndHandleTCP(s.TCPHandle)
	return nil
}

func (s *Relay) TCPHandle(c *net.TCPConn) error {
	defer c.Close()
	rc, err := net.DialTimeout("tcp", s.Raddr, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial TCP", s.Laddr, "<=>", s.Raddr, err)
		return err
	}
	defer rc.Close()
	go Copy(c, rc, s)
	Copy(rc, c, s)

	return nil
}
