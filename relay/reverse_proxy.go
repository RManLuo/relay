package relay

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type RP struct {
	RProxy *httputil.ReverseProxy
	host   string
}

func (f *RP) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	if f.host != "" {
		req.Host = f.host
	}
	f.RProxy.ServeHTTP(wr, req)
}
func NewRP(target, host string) *RP {
	u, _ := url.Parse(target)
	return &RP{
		RProxy: httputil.NewSingleHostReverseProxy(u),
		host:   host,
	}
}
