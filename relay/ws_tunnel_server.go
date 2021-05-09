package relay

import (
	"io"
	"net"
	"net/http"
	"time"

	proxyprotocol "github.com/pires/go-proxyproto"
	"golang.org/x/net/websocket"
)

func (s *Relay) RunWsTunnelServer() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return err
	}
	defer s.TCPListen.Close()
	Router := http.NewServeMux()
	Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		io.WriteString(w, "Never gonna give you up!")
		return
	})
	Router.Handle("/ws/", websocket.Handler(func(ws *websocket.Conn) {
		s.WsTunnelServerHandle(ws)
	}))
	http.Serve(s.TCPListen, Router)
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

func (s *Relay) WsTunnelServerHandle(ws *websocket.Conn) error {
	ws.PayloadType = websocket.BinaryFrame
	c, err := net.Dial("tcp", s.RemoteTCPAddr.String())
	if err != nil {
		ws.Close()
		return nil
	}

	header, err := proxyprotocol.HeaderProxyFromAddrs(byte(5), &Addr{
		NetworkType:   ws.Request().Header.Get("X-Forward-Protocol"),
		NetworkString: ws.Request().Header.Get("X-Forward-Address"),
	}, c.LocalAddr()).Format()
	if err == nil {
		c.Write(header)
	}

	go io.Copy(ws, c)
	go io.Copy(c, ws)

	go func() {
		var buf [1024 * 16]byte
		for {
			if s.TCPTimeout != 0 {
				if err := ws.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					return
				}
			}
			n, err := ws.Read(buf[:])
			if err != nil {
				return
			}
			if s.Traffic != nil {
				s.Traffic.RW.Lock()
				s.Traffic.TCP_UP += uint64(n)
				s.Traffic.RW.Unlock()
			}
			if _, err := c.Write(buf[0:n]); err != nil {
				return
			}
		}
	}()
	var buf [1024 * 16]byte
	for {
		if s.TCPTimeout != 0 {
			if err := c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
				return nil
			}
		}
		n, err := c.Read(buf[:])
		if err != nil {
			return nil
		}
		if s.Traffic != nil {
			s.Traffic.RW.Lock()
			s.Traffic.TCP_DOWN += uint64(n)
			s.Traffic.RW.Unlock()
		}
		if _, err := ws.Write(buf[0:n]); err != nil {
			return nil
		}
	}
	return nil
}
