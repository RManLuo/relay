package relay

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"neko-relay/limits"

	cache "github.com/patrickmn/go-cache"
	"github.com/txthinking/runnergroup"
	"github.com/txthinking/socks5"
)

type TF struct {
	TCP_UP   uint64
	TCP_DOWN uint64
	UDP      uint64
	RW       *sync.RWMutex
}

func NewTF() *TF {
	return &TF{TCP_UP: 0, TCP_DOWN: 0, UDP: 0, RW: new(sync.RWMutex)}
}

// Relay is relay server.
type Relay struct {
	TCPAddr       *net.TCPAddr
	UDPAddr       *net.UDPAddr
	RemoteTCPAddr *net.TCPAddr
	RemoteUDPAddr *net.UDPAddr
	TCPListen     *net.TCPListener
	UDPConn       *net.UDPConn
	UDPExchanges  *cache.Cache
	TCPTimeout    int
	UDPTimeout    int
	RunnerGroup   *runnergroup.RunnerGroup
	UDPSrc        *cache.Cache
	traffic       *TF
}

// NewRelay returns a Relay.
func NewRelay(addr, remote string, tcpTimeout, udpTimeout int, traffic *TF) (*Relay, error) {
	taddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	rtaddr, err := net.ResolveTCPAddr("tcp", remote)
	if err != nil {
		return nil, err
	}
	ruaddr, err := net.ResolveUDPAddr("udp", remote)
	if err != nil {
		return nil, err
	}
	cs := cache.New(cache.NoExpiration, cache.NoExpiration)
	cs2 := cache.New(cache.NoExpiration, cache.NoExpiration)
	if err := limits.Raise(); err != nil {
		log.Println("Try to raise system limits, got", err)
	}
	s := &Relay{
		TCPAddr:       taddr,
		UDPAddr:       uaddr,
		RemoteTCPAddr: rtaddr,
		RemoteUDPAddr: ruaddr,
		UDPExchanges:  cs,
		TCPTimeout:    tcpTimeout,
		UDPTimeout:    udpTimeout,
		RunnerGroup:   runnergroup.New(),
		UDPSrc:        cs2,
		traffic:       traffic,
	}
	return s, nil
}

// Run server.
func (s *Relay) ListenAndServe() error {
	s.RunnerGroup.Add(&runnergroup.Runner{
		Start: func() error {
			return s.RunTCPServer()
		},
		Stop: func() error {
			if s.TCPListen != nil {
				return s.TCPListen.Close()
			}
			return nil
		},
	})
	s.RunnerGroup.Add(&runnergroup.Runner{
		Start: func() error {
			return s.RunUDPServer()
		},
		Stop: func() error {
			if s.UDPConn != nil {
				return s.UDPConn.Close()
			}
			return nil
		},
	})
	return s.RunnerGroup.Wait()
}

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

// Shutdown server.
func (s *Relay) Shutdown() error {
	return s.RunnerGroup.Done()
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
	var buf [1024 * 32]byte
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
			if s.traffic != nil {
				s.traffic.RW.Lock()
				s.traffic.UDP += uint64(n)
				s.traffic.RW.Unlock()
			}
			if _, err := s.UDPConn.WriteToUDP(buf[0:n], ue.ClientAddr); err != nil {
				break
			}
		}
	}(ue, dst)
	return nil
}
