package relay

import (
	"net"
	"time"
)

func (s *Relay) RunUDPServer() error {
	var err error
	s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}
	defer s.UDPConn.Close()

	table := make(map[string]*UDPDistribute)
	buf := make([]byte, 1024*16)
	for {
		n, addr, err := s.UDPConn.ReadFrom(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			break
		}
		go func() {
			buf = buf[:n]
			if d, ok := table[addr.String()]; ok {
				if d.Connected {
					d.Cache <- buf
					return
				} else {
					delete(table, addr.String())
				}
			}
			c := NewUDPDistribute(s.UDPConn, addr)
			table[addr.String()] = c
			c.Cache <- buf
			s.UDPHandle(c)
			defer c.Close()
		}()
	}
	return nil
}

func (s *Relay) UDPHandle(c net.Conn) error {
	rc, err := net.DialTimeout("udp", s.Remote, time.Duration(s.UDPTimeout)*time.Second)
	if err != nil {
		return err
	}
	defer rc.Close()
	go Copy(c, rc, s.Traffic)
	Copy(rc, c, s.Traffic)
	return nil
}
