package relay

import (
	"log"
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
			return err
		}
		go func(c *net.TCPConn) {
			defer c.Close()
			if s.TCPTimeout != 0 {
				if err := c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					log.Println(err)
					return
				}
			}
			if err := s.TCPHandle(c); err != nil {
				log.Println(err)
			}
		}(c)
	}
	return nil
}

// TCPHandle handles request.
func (s *Relay) TCPHandle(c *net.TCPConn) error {
	tmp, err := net.Dial("tcp", s.RemoteTCPAddr.String())
	if err != nil {
		return err
	}
	rc := tmp.(*net.TCPConn)
	defer rc.Close()
	if s.TCPTimeout != 0 {
		if err := rc.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
			return err
		}
	}

	go func() {
		var buf [1024 * 16]byte
		for {
			if s.TCPTimeout != 0 {
				if err := rc.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					return
				}
			}
			n, err := rc.Read(buf[:])
			if err != nil {
				return
			}
			if s.traffic != nil {
				s.traffic.RW.Lock()
				s.traffic.TCP_UP += uint64(n)
				s.traffic.RW.Unlock()
			}
			if _, err := c.Write(buf[0:n]); err != nil {
				return
			}
		}
	}()
	var buf [1024 * 16]byte
	for {
		if s.TCPTimeout != 0 {
			if err := c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
				return nil
			}
		}
		n, err := c.Read(buf[:])
		if err != nil {
			return nil
		}
		if s.traffic != nil {
			s.traffic.RW.Lock()
			s.traffic.TCP_DOWN += uint64(n)
			s.traffic.RW.Unlock()
		}
		if _, err := rc.Write(buf[0:n]); err != nil {
			return nil
		}
	}
	return nil
}
