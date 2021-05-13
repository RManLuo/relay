package relay

import (
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWssTunnelServer(tcp, udp bool) error {
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
		Router.Handle("/wstcp/", websocket.Handler(s.WssTunnelServerTcpHandle))
	}
	if udp {
		Router.Handle("/wsudp/", websocket.Handler(s.WssTunnelServerUdpHandle))
	}
	http.ServeTLS(s.TCPListen, Router, certFile, keyFile)
	return nil
}
func (s *Relay) WssTunnelServerTcpHandle(ws *websocket.Conn) {
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

func (s *Relay) WssTunnelServerUdpHandle(ws *websocket.Conn) {
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
