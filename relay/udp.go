package relay

import (
	"fmt"
	"net"
	"time"
)

func (s *Relay) ListenUDP() (err error) {
	wait := 1.0
	for s.Status == 1 && s.UDPConn == nil {
		s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
		if err != nil {
			fmt.Println("Listen UDP", s.Laddr, err, "(retry in", wait, "s)")
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 1.1
		}
	}
	return
}
func (s *Relay) AcceptAndHandleUDP(handle func(c net.Conn) error) error {
	wait := 1.0
	table := make(map[string]*UDPDistribute)
	buf := make([]byte, 1024*16)
	for s.Status == 1 && s.UDPConn != nil {
		n, addr, err := s.UDPConn.ReadFrom(buf)
		if err != nil {
			fmt.Println("Accept", s.Laddr, err)
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 1.1
			break
		} else {
			wait = 1.0
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
			handle(c)
		}()
	}
	return nil
}
func (s *Relay) RunUDPServer() error {
	s.ListenUDP()
	defer s.UDPConn.Close()
	s.AcceptAndHandleUDP(s.UDPHandle)
	return nil
}

func (s *Relay) UDPHandle(c net.Conn) error {
	defer c.Close()
	rc, err := net.DialTimeout("udp", s.Raddr, time.Duration(s.UDPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial UDP", s.Laddr, "<=>", s.Raddr, err)
		return err
	}
	defer rc.Close()
	go Copy(c, rc, s)
	Copy(rc, c, s)
	return nil
}
