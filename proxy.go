package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"golang.org/x/time/rate"
)

type Server struct {
	target  *url.URL
	proxy   *httputil.ReverseProxy
	wsProxy *WebsocketProxy
	myTransport
}

func (cfg *ConfigData) NewServer() (*Server, error) {
	url, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}
	wsurl, err := url.Parse(cfg.WSURL)
	if err != nil {
		return nil, err
	}
	s := &Server{target: url, proxy: httputil.NewSingleHostReverseProxy(url), wsProxy: NewProxy(wsurl)}
	s.myTransport.blockRangeLimit = cfg.BlockRangeLimit
	s.myTransport.url = cfg.URL
	s.matcher, err = newMatcher(cfg.Allow)
	if err != nil {
		return nil, err
	}
	s.visitors = make(map[string]*rate.Limiter)
	s.noLimitIPs = make(map[string]struct{})
	for _, ip := range cfg.NoLimit {
		s.noLimitIPs[ip] = struct{}{}
	}
	s.proxy.Transport = &s.myTransport
	s.wsProxy.Transport = &s.myTransport
	return s, nil
}

func (p *Server) RPCProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-rpc-proxy", "rpc-proxy")
	p.proxy.ServeHTTP(w, r)
}

func (p *Server) WSProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-rpc-proxy", "rpc-proxy")
	p.wsProxy.ServeHTTP(w, r)
}
