package relay

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelTcpClient() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		fmt.Println("Listen", s.Local, err)
		return err
	}
	defer s.TCPListen.Close()
	counter := 0
	for {
		c, err := s.TCPListen.AcceptTCP()
		if err == nil {
			counter = 0
		} else {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				fmt.Println("Accept", s.Local, err)
				continue
			}
			counter++
			if counter > 10 {
				break
			}
			time.Sleep(10 * time.Second)
			continue
		}
		go s.WsTunnelClientTcpHandle(c)
	}
	return nil
}

func (s *Relay) WsTunnelClientTcpHandle(c *net.TCPConn) error {
	defer c.Close()
	ws_config, err := websocket.NewConfig("ws://"+s.Remote+"/wstcp/", "http://"+s.Remote+"/wstcp/")
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RIP)
	ws_config.Header.Set("X-Forward-Host", "www.upyun.com")
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		fmt.Println("Dial ws", s.Remote, err)
		return err
	}
	defer rc.Close()
	rc.PayloadType = websocket.BinaryFrame

	go Copy(rc, c, s.Traffic)
	Copy(c, rc, s.Traffic)
	return nil
}

func (s *Relay) RunWsTunnelUdpClient() error {
	var err error
	s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}
	defer s.UDPConn.Close()
	table := make(map[string]*UDPDistribute)
	buf := make([]byte, 1024*16)
	for {
		n, addr, err := s.UDPConn.ReadFrom(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			break
		}
		go func() {
			buf = buf[:n]
			if d, ok := table[addr.String()]; ok {
				if d.Connected {
					d.Cache <- buf
					return
				} else {
					delete(table, addr.String())
				}
			}
			c := NewUDPDistribute(s.UDPConn, addr)
			table[addr.String()] = c
			c.Cache <- buf
			s.WsTunnelClientUdpHandle(c)
		}()
	}
	return nil
}

func (s *Relay) WsTunnelClientUdpHandle(c net.Conn) error {
	defer c.Close()
	ws_config, err := websocket.NewConfig("ws://"+s.Remote+"/wsudp/", "http://"+s.Remote+"/wsudp/")
	if err != nil {
		return err
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4240.198 Safari/537.36")
	// ws_config.Header.Set("X-Forward-For", s.RemoteTCPAddr.IP.String())
	ws_config.Header.Set("X-Forward-Host", "www.upyun.com")
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	if err != nil {
		return err
	}
	defer rc.Close()
	rc.PayloadType = websocket.BinaryFrame

	go Copy(c, rc, s.Traffic)
	Copy(rc, c, s.Traffic)
	return nil
}
