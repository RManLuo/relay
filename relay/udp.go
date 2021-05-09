package relay

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/txthinking/socks5"
)

// RunUDPServer starts udp server.
func (s *Relay) RunUDPServer() error {
	var err error
	s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}
	defer s.UDPConn.Close()
	for {
		b := make([]byte, 65507)
		n, addr, err := s.UDPConn.ReadFromUDP(b)
		if err != nil {
			return err
		}
		go func(addr *net.UDPAddr, b []byte) {
			if err := s.UDPHandle(addr, b); err != nil {
				log.Println(err)
				return
			}
		}(addr, b[0:n])
	}
	return nil
}

// UDPHandle handles packet.
func (s *Relay) UDPHandle(addr *net.UDPAddr, b []byte) error {
	src := addr.String()
	send := func(ue *socks5.UDPExchange, data []byte) error {
		_, err := ue.RemoteConn.Write(data)
		if err != nil {
			return err
		}
		return nil
	}

	dst := s.RemoteUDPAddr.String()
	var ue *socks5.UDPExchange
	iue, ok := s.UDPExchanges.Get(src + dst)
	if ok {
		ue = iue.(*socks5.UDPExchange)
		return send(ue, b)
	}

	var laddr *net.UDPAddr
	any, ok := s.UDPSrc.Get(src + dst)
	if ok {
		laddr = any.(*net.UDPAddr)
	}
	rc, err := net.DialUDP("udp", laddr, s.RemoteUDPAddr)
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			// we dont choose lock, so ignore this error
			return nil
		}
		return err
	}
	if laddr == nil {
		s.UDPSrc.Set(src+dst, rc.LocalAddr().(*net.UDPAddr), -1)
	}
	ue = &socks5.UDPExchange{
		ClientAddr: addr,
		RemoteConn: rc,
	}
	if err := send(ue, b); err != nil {
		ue.RemoteConn.Close()
		return err
	}
	s.UDPExchanges.Set(src+dst, ue, -1)
	go func(ue *socks5.UDPExchange, dst string) {
		defer func() {
			ue.RemoteConn.Close()
			s.UDPExchanges.Delete(ue.ClientAddr.String() + dst)
		}()
		var buf [65507]byte
		for {
			if s.UDPTimeout != 0 {
				if err := ue.RemoteConn.SetDeadline(time.Now().Add(time.Duration(s.UDPTimeout) * time.Second)); err != nil {
					log.Println(err)
					break
				}
			}
			n, err := ue.RemoteConn.Read(buf[:])
			if err != nil {
				break
			}
			if s.Traffic != nil {
				s.Traffic.RW.Lock()
				s.Traffic.UDP += uint64(n)
				s.Traffic.RW.Unlock()
			}
			if _, err := s.UDPConn.WriteToUDP(buf[0:n], ue.ClientAddr); err != nil {
				break
			}
		}
	}(ue, dst)
	return nil
}
