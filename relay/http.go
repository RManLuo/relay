package relay

import (
	"net/http"
	"net/url"
)

func (s *Relay) RunHttpServer(tls bool) error {
	s.ListenTCP()
	defer s.TCPListen.Close()
	handler := http.NewServeMux()

	target := "http://" + s.Raddr
	if tls {
		target = "https://" + s.Raddr
	}
	u, err := url.Parse(target)
	if err != nil {
		return err
	}
	handler.Handle("/", NewSingleHostReverseProxy(u, s))
	svr := &http.Server{Handler: handler}
	if tls {
		svr.ServeTLS(s.TCPListen, Config.Certfile, Config.Keyfile)
	} else {
		svr.Serve(s.TCPListen)
	}
	// defer svr.Shutdown(nil)
	return nil
}
