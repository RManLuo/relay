package relay

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsServer() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return err
	}
	defer s.TCPListen.Close()
	// http.Serve(s.TCPListen, )
	return nil
}

func (s *Relay) WS_Handle(ws *websocket.Conn) error {
	addr := s.RemoteTCPAddr.IP.String() + ":" + strconv.Itoa(s.RemoteTCPAddr.Port)
	ws_config, err := websocket.NewConfig("ws://"+addr+"/host-hkt.nkeo.top", "http://"+addr+"/host-hkt.nkeo.top")
	fmt.Println(ws, ws_config)
	if err != nil {
		return err
	}
	// ws_config.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36")
	// ws_config.Header.Set("X-Forward-For", s.RemoteTCPAddr.IP.String())
	// ws_config.Header.Set("X-Forward-Protocol", ws.RemoteAddr().Network())
	// ws_config.Header.Set("X-Forward-Address", ws.RemoteAddr().String())
	rc, err := websocket.DialConfig(ws_config)
	defer rc.Close()
	if err != nil {
		return err
	}
	if s.TCPTimeout != 0 {
		if err := rc.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
			return err
		}
	}
	go func() {
		var buf [1024 * 16]byte
		for {
			if s.TCPListen == nil {
				ws.Close()
				rc.Close()
				return
			}
			if s.TCPTimeout != 0 {
				if err := ws.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					return
				}
			}
			n, err := ws.Read(buf[:])
			if err != nil {
				return
			}
			if s.traffic != nil {
				s.traffic.RW.Lock()
				s.traffic.TCP_DOWN += uint64(n)
				s.traffic.RW.Unlock()
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
		if s.traffic != nil {
			s.traffic.RW.Lock()
			s.traffic.TCP_UP += uint64(n)
			s.traffic.RW.Unlock()
		}
		if _, err := ws.Write(buf[0:n]); err != nil {
			return nil
		}
	}
	return nil
}
