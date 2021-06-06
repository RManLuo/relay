package relay

import (
	"fmt"
	"net"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelTcpClient() error {
	s.ListenTCP()
	defer s.TCPListen.Close()
	s.AcceptAndHandleTCP(s.WsTunnelClientTcpHandle)
	return nil
}

func (s *Relay) WsTunnelClientTcpHandle(c *net.TCPConn) error {
	defer c.Close()
	ws_config, err := websocket.NewConfig("ws://"+s.Raddr+"/wstcp/", "http://"+s.Raddr+"/wstcp/")
	if err != nil {
		fmt.Println("WS Config", s.Raddr, err)
		return err
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RIP)
	ws_config.Header.Set("X-Forward-Host", Config.Fakehost)
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		fmt.Println("Dial WS", s.Raddr, err)
		return err
	}
	defer rc.Close()
	rc.PayloadType = websocket.BinaryFrame

	go Copy(rc, c, s)
	Copy(c, rc, s)
	return nil
}

func (s *Relay) RunWsTunnelUdpClient() error {
	s.ListenUDP()
	defer s.UDPConn.Close()
	s.AcceptAndHandleUDP(s.WsTunnelClientUdpHandle)
	return nil
}

func (s *Relay) WsTunnelClientUdpHandle(c net.Conn) error {
	defer c.Close()
	ws_config, err := websocket.NewConfig("ws://"+s.Raddr+"/wsudp/", "http://"+s.Raddr+"/wsudp/")
	if err != nil {
		fmt.Println("WS Config", s.Raddr, err)
		return err
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RIP)
	ws_config.Header.Set("X-Forward-Host", Config.Fakehost)
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		fmt.Println("Dial WS", s.Raddr, err)
		return err
	}
	defer rc.Close()
	rc.PayloadType = websocket.BinaryFrame

	go Copy(c, rc, s)
	Copy(rc, c, s)
	return nil
}
