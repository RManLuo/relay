package relay

import (
	"io"
	"net"
	"net/http"
	"time"

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
	Router.Handle("/ws/", websocket.Handler(s.WsTunnelServerHandle))
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

func (s *Relay) WsTunnelServerHandle(ws *websocket.Conn) {
	tmp, err := net.Dial("tcp", s.RemoteTCPAddr.String())
	if err != nil {
		ws.Close()
		return
	}
	rc := tmp.(*net.TCPConn)
	defer rc.Close()

	ws.PayloadType = websocket.BinaryFrame

	go func() {
		var buf [1024 * 16]byte
		for {
			// if s.TCPTimeout != 0 {
			// 	if err := ws.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
			// 		return
			// 	}
			// }
			n, err := ws.Read(buf[:])
			if err != nil {
				return
			}
			if s.Traffic != nil {
				s.Traffic.RW.Lock()
				s.Traffic.TCP_UP += uint64(n)
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
				return
			}
		}
		n, err := rc.Read(buf[:])
		if err != nil {
			return
		}
		if s.Traffic != nil {
			s.Traffic.RW.Lock()
			s.Traffic.TCP_DOWN += uint64(n)
			s.Traffic.RW.Unlock()
		}
		if _, err := ws.Write(buf[0:n]); err != nil {
			return
		}
	}
	return
}
