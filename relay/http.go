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
	s.Svr = &http.Server{Handler: handler}
	if tls {
		s.Svr.ServeTLS(s.TCPListen, Config.Certfile, Config.Keyfile)
	} else {
		s.Svr.Serve(s.TCPListen)
	}
	// defer svr.Shutdown(nil)
	return nil
}
