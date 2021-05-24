package relay

import (
	"io"
	"log"
	"neko-relay/config"
	"neko-relay/limits"
	"net"
)

var (
	Config config.CONF
)

type Relay struct {
	TCPAddr    *net.TCPAddr
	UDPAddr    *net.UDPAddr
	TCPListen  *net.TCPListener
	UDPConn    *net.UDPConn
	TCPTimeout int
	UDPTimeout int
	Local      string
	Remote     string
	RIP        string
	Traffic    *TF
	Protocol   string
}

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
		TCPAddr:    taddr,
		UDPAddr:    uaddr,
		TCPTimeout: tcpTimeout,
		UDPTimeout: udpTimeout,
		Local:      local,
		Remote:     remote,
		RIP:        rip,
		Traffic:    traffic,
		Protocol:   protocol,
	}
	return s, nil
}

// Run server.
func (s *Relay) Serve() error {
	if s.Protocol == "tcp" || s.Protocol == "tcp+udp" {
		go s.RunTCPServer()
	}
	if s.Protocol == "udp" || s.Protocol == "tcp+udp" {
		go s.RunUDPServer()
	}
	if s.Protocol == "websocket" {
		go s.RunWsServer()
	}
	if s.Protocol == "tls" {
		go s.RunTCPServer()
	}
	if s.Protocol == "ws_tunnel_server" {
		go s.RunWsTunnelServer(true, true)
	}
	if s.Protocol == "ws_tunnel_client" {
		go s.RunWsTunnelTcpClient()
		go s.RunWsTunnelUdpClient()
	}

	if s.Protocol == "wss_tunnel_server" {
		go s.RunWssTunnelServer(true, true)
	}
	if s.Protocol == "wss_tunnel_client" {
		go s.RunWssTunnelTcpClient()
		go s.RunWssTunnelUdpClient()
	}
	return nil
}

// Shutdown server.
func (s *Relay) Close() error {
	if s.TCPListen != nil {
		s.TCPListen.Close()
		s.TCPListen = nil
	}
	if s.UDPConn != nil {
		s.UDPConn.Close()
		s.UDPConn = nil
	}
	return nil
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
	for tf != nil {
		n, err := src.Read(buf[:])
		if err != nil {
			break
		}
		tf.Add(uint64(n))
		if _, err := dst.Write(buf[0:n]); err != nil {
			break
		}
	}
	return nil
}
