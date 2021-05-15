package relay

import (
	"io"
	"log"
	"net"

	"neko-relay/limits"

	"github.com/txthinking/runnergroup"
)

var (
	CertFile, KeyFile string
)

type Relay struct {
	TCPAddr     *net.TCPAddr
	UDPAddr     *net.UDPAddr
	TCPListen   *net.TCPListener
	UDPConn     *net.UDPConn
	TCPTimeout  int
	UDPTimeout  int
	Local       string
	Remote      string
	RIP         string
	Traffic     *TF
	Protocol    string
	RunnerGroup *runnergroup.RunnerGroup
}

// NewRelay returns a Relay.
func NewRelay(local, remote, rip string, tcpTimeout, udpTimeout int, traffic *TF, protocol string) (*Relay, error) {
	taddr, err := net.ResolveTCPAddr("tcp", local)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		return nil, err
	}
	if err := limits.Raise(); err != nil {
		log.Println("Try to raise system limits, got", err)
	}
	s := &Relay{
		TCPAddr:     taddr,
		UDPAddr:     uaddr,
		TCPTimeout:  tcpTimeout,
		UDPTimeout:  udpTimeout,
		Local:       local,
		Remote:      remote,
		RIP:         rip,
		Traffic:     traffic,
		Protocol:    protocol,
		RunnerGroup: runnergroup.New(),
	}
	return s, nil
}
func (s *Relay) closeTCP() error {
	if s.TCPListen != nil {
		return s.TCPListen.Close()
	}
	return nil
}
func (s *Relay) closeUDP() error {
	if s.UDPConn != nil {
		return s.UDPConn.Close()
	}
	return nil
}

// Run server.
func (s *Relay) ListenAndServe() error {
	if s.Protocol == "tcp" || s.Protocol == "tcp+udp" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunTCPServer,
			Stop:  s.closeTCP,
		})
	}
	if s.Protocol == "udp" || s.Protocol == "tcp+udp" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunUDPServer,
			Stop:  s.closeUDP,
		})
	}
	if s.Protocol == "websocket" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunWsServer,
			Stop:  s.closeTCP,
		})
	}
	if s.Protocol == "tls" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunTlsServer,
			Stop:  s.closeTCP,
		})
	}
	if s.Protocol == "ws_tunnel_server" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: func() error {
				return s.RunWsTunnelServer(true, true)
			},
			Stop: func() error {
				s.closeTCP()
				s.closeUDP()
				return nil
			},
		})
	}
	if s.Protocol == "ws_tunnel_client" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunWsTunnelTcpClient,
			Stop:  s.closeTCP,
		})
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunWsTunnelUdpClient,
			Stop:  s.closeUDP,
		})
	}

	if s.Protocol == "wss_tunnel_server" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: func() error {
				return s.RunWssTunnelServer(true, true, CertFile, KeyFile)
			},
			Stop: func() error {
				s.closeTCP()
				s.closeUDP()
				return nil
			},
		})
	}
	if s.Protocol == "wss_tunnel_client" {
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunWssTunnelTcpClient,
			Stop:  s.closeTCP,
		})
		s.RunnerGroup.Add(&runnergroup.Runner{
			Start: s.RunWssTunnelUdpClient,
			Stop:  s.closeUDP,
		})
	}

	return s.RunnerGroup.Wait()
}

// Shutdown server.
func (s *Relay) Shutdown() error {
	return s.RunnerGroup.Done()
}

func Copy(dst io.Writer, src io.Reader, tf *TF) error {
	// n, err := io.Copy(dst, src)
	// if err != nil {
	// 	return nil
	// }
	// if tf != nil {
	// 	tf.Add(uint64(n))
	// }
	// return nil
	var buf [1024 * 16]byte
	for {
		n, err := src.Read(buf[:])
		if err != nil {
			return nil
		}
		if tf != nil {
			tf.Add(uint64(n))
		}
		if _, err := dst.Write(buf[0:n]); err != nil {
			return nil
		}
	}
}
