package relay

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelSrc() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return err
	}
	defer s.TCPListen.Close()
	Router := http.NewServeMux()
	// Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	w.WriteHeader(404)
	// 	io.WriteString(w, "Never gonna give you up!")
	// 	return
	// })

	Router.Handle("/", websocket.Handler(func(ws *websocket.Conn) {
		WS_Tunnel_Src_Handle(s, ws)
	}))

	http.Serve(s.TCPListen, Router)
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
			if err := s.TCPHandle(c); err != nil {
				log.Println(err)
			}
		}(c)
	}
	return nil
}

type Addr struct {
	NetworkType   string
	NetworkString string
}

func (this *Addr) Network() string {
	return this.NetworkType
}
func (this *Addr) String() string {
	return this.NetworkString
}

func WS_Tunnel_Src_Handle(s *Relay, ws *websocket.Conn) error {
	ws.PayloadType = websocket.BinaryFrame
	proxy, err := net.Dial("tcp", s.RemoteTCPAddr.String())
	if err != nil {
		ws.Close()
		return nil
	}

	go io.Copy(ws, proxy)
	go io.Copy(proxy, ws)
	// rc := proxy.(*net.TCPConn)

	// header, err := proxyprotocol.HeaderProxyFromAddrs(byte(5), &Addr{
	// 	NetworkType:   ws.Request().Header.Get("X-Forward-Protocol"),
	// 	NetworkString: ws.Request().Header.Get("X-Forward-Address"),
	// }, proxy.LocalAddr()).Format()
	// proxy.Write(header)

	// go func() {
	// 	var buf [1024 * 16]byte
	// 	for {
	// 		if s.TCPTimeout != 0 {
	// 			if err := ws.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
	// 				return
	// 			}
	// 		}
	// 		n, err := ws.Read(buf[:])
	// 		if err != nil {
	// 			return
	// 		}
	// 		if s.traffic != nil {
	// 			s.traffic.RW.Lock()
	// 			s.traffic.TCP_UP += uint64(n)
	// 			s.traffic.RW.Unlock()
	// 		}
	// 		if _, err := proxy.Write(buf[0:n]); err != nil {
	// 			return
	// 		}
	// 	}
	// }()
	// var buf [1024 * 16]byte
	// for {
	// 	if s.TCPTimeout != 0 {
	// 		if err := proxy.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
	// 			return nil
	// 		}
	// 	}
	// 	n, err := proxy.Read(buf[:])
	// 	if err != nil {
	// 		return nil
	// 	}
	// 	if s.traffic != nil {
	// 		s.traffic.RW.Lock()
	// 		s.traffic.TCP_DOWN += uint64(n)
	// 		s.traffic.RW.Unlock()
	// 	}
	// 	if _, err := ws.Write(buf[0:n]); err != nil {
	// 		return nil
	// 	}
	// }
	return nil
}
