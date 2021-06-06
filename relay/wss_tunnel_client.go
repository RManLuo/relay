package relay

import (
	"crypto/tls"
	"fmt"
	"net"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWssTunnelTcpClient() error {
	s.ListenTCP()
	defer s.TCPListen.Close()
	s.AcceptAndHandleTCP(s.WssTunnelClientTcpHandle)
	return nil
}

func (s *Relay) WssTunnelClientTcpHandle(c *net.TCPConn) error {
	ws_config, err := websocket.NewConfig("wss://"+s.Raddr+"/wstcp/", "https://"+s.Raddr+"/wstcp/")
	if err != nil {
		return err
	}
	ws_config.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RIP)
	ws_config.Header.Set("X-Forward-Host", Config.Fakehost)
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		fmt.Println("Dial ws", s.Raddr, err)
		return err
	}
	rc.PayloadType = websocket.BinaryFrame
	defer rc.Close()

	go Copy(rc, c, s)
	Copy(c, rc, s)
	return nil
}

func (s *Relay) RunWssTunnelUdpClient() error {
	s.ListenUDP()
	defer s.UDPConn.Close()
	s.AcceptAndHandleUDP(s.WssTunnelClientUdpHandle)
	return nil
}

func (s *Relay) WssTunnelClientUdpHandle(c net.Conn) error {
	ws_config, err := websocket.NewConfig("wss://"+s.Raddr+"/wsudp/", "https://"+s.Raddr+"/wsudp/")
	if err != nil {
		return err
	}
	ws_config.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RIP)
	ws_config.Header.Set("X-Forward-Host", Config.Fakehost)
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		return err
	}
	rc.PayloadType = websocket.BinaryFrame
	defer rc.Close()

	go Copy(c, rc, s)
	Copy(rc, c, s)
	return nil
}
