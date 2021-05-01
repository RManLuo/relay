package relay

import (
	"io"
	"net"
	"strconv"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelDst() (err error) {
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return
	}
	defer s.TCPListen.Close()

	for {
		conn, err := s.TCPListen.Accept()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			break
		}
		go WS_Tunnel_Dst_Handle(s, conn)
	}
	return
}

func WS_Tunnel_Dst_Handle(s *Relay, conn net.Conn) {
	addr := s.TCPAddr.IP.String() + ":" + strconv.Itoa(s.TCPAddr.Port)
	ws_config, err := websocket.NewConfig("ws://"+addr+"/ws/", "http://"+addr+"/ws/")
	if err != nil {
		conn.Close()
		return
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RemoteTCPAddr.IP.String())
	ws_config.Header.Set("X-Forward-Protocol", conn.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", conn.RemoteAddr().String())
	proxy, err := websocket.DialConfig(ws_config)
	if err != nil {
		conn.Close()
		return
	}
	proxy.PayloadType = websocket.BinaryFrame

	go io.Copy(conn, proxy)
	go io.Copy(proxy, conn)
	// go copyIO(conn, proxy, r)
	// go copyIO(proxy, conn, r)
}
