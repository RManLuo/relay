package relay

import (
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelServer(tcp, udp bool) error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return err
	}
	defer s.TCPListen.Close()
	Router := http.NewServeMux()
	Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		io.WriteString(w, "Never gonna give you up!")
		return
	})
	if tcp {
		Router.Handle("/wstcp/", websocket.Handler(s.WsTunnelServerTcpHandle))
	}
	if udp {
		Router.Handle("/wsudp/", websocket.Handler(s.WsTunnelServerUdpHandle))
	}
	http.Serve(s.TCPListen, Router)
	return nil
}

type Addr struct {
	NetworkType   string
	NetworkString string
}

func (this *Addr) Network() string {
	return this.NetworkType
}
func (this *Addr) String() string {
	return this.NetworkString
}

func (s *Relay) WsTunnelServerTcpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	tmp, err := net.DialTimeout("tcp", s.Remote, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		return
	}
	rc := tmp.(*net.TCPConn)
	defer rc.Close()
	go Copy(rc, ws, s.Traffic)
	Copy(ws, rc, s.Traffic)
	return
}

func (s *Relay) WsTunnelServerUdpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	rc, err := net.DialTimeout("udp", s.Remote, time.Duration(s.UDPTimeout)*time.Second)
	if err != nil {
		return
	}
	defer rc.Close()

	go Copy(rc, ws, s.Traffic)
	Copy(ws, rc, s.Traffic)
	return
}
