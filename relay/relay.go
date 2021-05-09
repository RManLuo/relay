package relay

import (
	"log"
	"net"

	"neko-relay/limits"

	cache "github.com/patrickmn/go-cache"
	"github.com/txthinking/runnergroup"
)

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
	Traffic       *TF
	Protocol      string
}

// NewRelay returns a Relay.
func NewRelay(local, remote string, tcpTimeout, udpTimeout int, traffic *TF, protocol string) (*Relay, error) {
	taddr, err := net.ResolveTCPAddr("tcp", local)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", local)
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
		Traffic:       traffic,
		Protocol:      protocol,
	}
	return s, nil
}

// Run server.
func (s *Relay) ListenAndServe() error {
	if s.Protocol == "tcp" || s.Protocol == "tcp+udp" {
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
	}
	if s.Protocol == "udp" || s.Protocol == "tcp+udp" {
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
	}
	if s.Protocol == "websocket" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: func() error {
				return s.RunWsServer()
			},
			Stop: func() error {
				if s.TCPListen != nil {
					return s.TCPListen.Close()
				}
				return nil
			},
		})
	}
	if s.Protocol == "tls" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: func() error {
				return s.RunTlsServer()
			},
			Stop: func() error {
				if s.TCPListen != nil {
					return s.TCPListen.Close()
				}
				return nil
			},
		})
	}
	if s.Protocol == "ws_tunnel_server" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: func() error {
				return s.RunWsTunnelServer()
			},
			Stop: func() error {
				if s.TCPListen != nil {
					return s.TCPListen.Close()
				}
				return nil
			},
		})
	}
	if s.Protocol == "ws_tunnel_client" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: func() error {
				return s.RunWsTunnelClient()
			},
			Stop: func() error {
				if s.TCPListen != nil {
					return s.TCPListen.Close()
				}
				return nil
			},
		})
	}

	return s.RunnerGroup.Wait()
}

// Shutdown server.
func (s *Relay) Shutdown() error {
	return s.RunnerGroup.Done()
}
