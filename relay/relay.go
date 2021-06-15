package relay

import (
	"io"
	"log"
	"neko-relay/config"
	"neko-relay/limits"
	. "neko-relay/rules"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	Config config.CONF
)

type Relay struct {
	TCPAddr    *net.TCPAddr
	UDPAddr    *net.UDPAddr
	TCPListen  *net.TCPListener
	UDPConn    *net.UDPConn
	Svr        *http.Server
	TCPTimeout int
	UDPTimeout int
	Laddr      string
	Raddr      string
	REMOTE     string
	RIP        string
	RPORT      int
	Traffic    *TF
	Protocol   string
	Status     bool
}

func NewRelay(r Rule, tcpTimeout, udpTimeout int, traffic *TF, protocol string) (*Relay, error) {
	laddr := ":" + strconv.Itoa(int(r.Port))
	raddr := r.RIP + ":" + strconv.Itoa(int(r.Rport))
	taddr, err := net.ResolveTCPAddr("tcp", laddr)
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", laddr)
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
		Laddr:      laddr,
		Raddr:      raddr,
		RIP:        r.RIP,
		REMOTE:     r.Remote,
		Traffic:    traffic,
		Protocol:   protocol,
		Status:     true,
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
	if s.Protocol == "http" {
		go s.RunHttpServer(false)
	}
	if s.Protocol == "https" {
		go s.RunHttpServer(true)
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
	s.Status = false
	if s.Svr != nil {
		s.Svr.Shutdown(nil)
	}
	time.Sleep(10 * time.Millisecond)
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

var (
	Pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 16*1024)
		},
	}
)

func Copy(dst, src net.Conn, s *Relay) error {
	defer src.Close()
	defer dst.Close()
	return Copy_io(dst, src, s)
}

func Copy_io(dst io.Writer, src io.Reader, s *Relay) error {
	// n, err := io.Copy(dst, src)
	// if err != nil {
	// 	return nil
	// }
	// if tf != nil {
	// 	tf.Add(uint64(n))
	// }
	// return nil
	buf := Pool.Get().([]byte)
	defer Pool.Put(buf)
	if n, err := io.CopyBuffer(dst, src, buf); err == nil {
		if s.Traffic != nil {
			s.Traffic.Add(uint64(n))
		}
		return nil
	} else {
		return err
	}
}
