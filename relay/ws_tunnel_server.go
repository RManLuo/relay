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
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		io.WriteString(w, "Never gonna give you up!")
		return
	})
	if tcp {
		handler.Handle("/wstcp/", websocket.Handler(s.WsTunnelServerTcpHandle))
	}
	if udp {
		handler.Handle("/wsudp/", websocket.Handler(s.WsTunnelServerUdpHandle))
	}
	svr := &http.Server{Handler: handler}
	svr.Serve(s.TCPListen)
	defer svr.Shutdown(nil)
	return nil
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
