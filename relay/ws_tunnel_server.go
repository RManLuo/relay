package relay

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelServer(tcp, udp bool) error {
	var err error
	wait := 2.0
	for s.TCPListen == nil {
		s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
		if err != nil {
			fmt.Println("Listen", s.Local, err, "(retry in", wait, "s)")
			time.Sleep(time.Duration(wait) * time.Second)
			wait *= 1.1
		}
	}
	defer s.TCPListen.Close()
	handler := http.NewServeMux()
	if tcp {
		handler.Handle("/wstcp/", websocket.Handler(s.WsTunnelServerTcpHandle))
	}
	if udp {
		handler.Handle("/wsudp/", websocket.Handler(s.WsTunnelServerUdpHandle))
	}
	handler.Handle("/", NewRP("https://www.upyun.com", "www.upyun.com"))

	svr := &http.Server{Handler: handler}
	svr.Serve(s.TCPListen)
	defer svr.Shutdown(nil)
	return nil
}

func (s *Relay) WsTunnelServerTcpHandle(ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	defer ws.Close()

	rc, err := net.DialTimeout("tcp", s.Remote, time.Duration(s.TCPTimeout)*time.Second)
	if err != nil {
		fmt.Println("Dial TCP", s.Local, "<=>", s.Remote, err)
		return
	}
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
		fmt.Println("Dial UDP", s.Local, "<=>", s.Remote, err)
		return
	}
	defer rc.Close()

	go Copy(rc, ws, s.Traffic)
	Copy(ws, rc, s.Traffic)
	return
}
