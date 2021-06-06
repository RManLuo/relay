package relay

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelServer(tcp, udp bool) error {
	s.ListenTCP()
	defer s.TCPListen.Close()
	handler := http.NewServeMux()
	if tcp {
		handler.Handle("/wstcp/", websocket.Handler(s.WsTunnelServerTcpHandle))
	}
	if udp {
		handler.Handle("/wsudp/", websocket.Handler(s.WsTunnelServerUdpHandle))
	}
	handler.Handle("/", NewRP(Config.Fakeurl, Config.Fakehost))

	svr := &http.Server{Handler: handler}
	svr.Serve(s.TCPListen)
	defer svr.Shutdown(nil)
	return nil
}

func (s *Relay) WsTunnelServerTcpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	rc, err := net.DialTimeout("tcp", s.Raddr, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial TCP", s.Laddr, "<=>", s.Raddr, err)
		return
	}
	defer rc.Close()
	go Copy(rc, ws, s)
	Copy(ws, rc, s)
	return
}

func (s *Relay) WsTunnelServerUdpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	rc, err := net.DialTimeout("udp", s.Raddr, time.Duration(s.UDPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial UDP", s.Laddr, "<=>", s.Raddr, err)
		return
	}
	defer rc.Close()

	go Copy(rc, ws, s)
	Copy(ws, rc, s)
	return
}
