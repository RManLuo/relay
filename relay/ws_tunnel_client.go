package relay

import (
	"log"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelClient() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return err
	}
	defer s.TCPListen.Close()
	for {
		c, err := s.TCPListen.AcceptTCP()
		if err != nil {
			return err
		}
		go func(c *net.TCPConn) {
			defer c.Close()
			if s.TCPTimeout != 0 {
				if err := c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					log.Println(err)
					return
				}
			}
			if err := s.WsTunnelClientHandle(c); err != nil {
				log.Println(err)
			}
		}(c)
	}
	return nil
}

func (s *Relay) WsTunnelClientHandle(c *net.TCPConn) error {
	addr := s.RemoteTCPAddr.IP.String() + ":" + strconv.Itoa(s.RemoteTCPAddr.Port)
	ws_config, err := websocket.NewConfig("ws://"+addr+"/ws/", "http://"+addr+"/ws/")
	if err != nil {
		c.Close()
		return err
	}
	ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4240.198 Safari/537.36")
	ws_config.Header.Set("X-Forward-For", s.RemoteTCPAddr.IP.String())
	ws_config.Header.Set("X-Forward-Protocol", c.RemoteAddr().Network())
	ws_config.Header.Set("X-Forward-Address", c.RemoteAddr().String())

	rc, err := websocket.DialConfig(ws_config)
	defer rc.Close()
	if err != nil {
		c.Close()
		return err
	}
	rc.PayloadType = websocket.BinaryFrame

	if s.TCPTimeout != 0 {
		if err := rc.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
			return err
		}
	}

	go func() {
		var buf [1024 * 16]byte
		for {
			if s.TCPTimeout != 0 {
				if err := c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					return
				}
			}
			n, err := c.Read(buf[:])
			if err != nil {
				return
			}
			if s.Traffic != nil {
				s.Traffic.RW.Lock()
				s.Traffic.TCP_DOWN += uint64(n)
				s.Traffic.RW.Unlock()
			}
			if _, err := rc.Write(buf[0:n]); err != nil {
				return
			}
		}

	}()
	var buf [1024 * 16]byte
	for {
		if s.TCPTimeout != 0 {
			if err := rc.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
				return nil
			}
		}
		n, err := rc.Read(buf[:])
		if err != nil {
			return nil
		}
		if s.Traffic != nil {
			s.Traffic.RW.Lock()
			s.Traffic.TCP_UP += uint64(n)
			s.Traffic.RW.Unlock()
		}
		if _, err := c.Write(buf[0:n]); err != nil {
			return nil
		}
	}
	return nil
}
