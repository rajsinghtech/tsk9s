package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"tailscale.com/tsnet"
)

// ProxyEndpoint is a running local reverse proxy for one cluster.
type ProxyEndpoint struct {
	Cluster   Cluster
	LocalAddr string // "127.0.0.1:<port>"
	listener  net.Listener
	server    *http.Server
}

// ProxyManager manages local reverse proxies for discovered clusters.
type ProxyManager struct {
	srv *tsnet.Server

	mu        sync.Mutex
	endpoints map[string]*ProxyEndpoint // keyed by cluster FQDN
}

func NewProxyManager(srv *tsnet.Server) *ProxyManager {
	return &ProxyManager{
		srv:       srv,
		endpoints: make(map[string]*ProxyEndpoint),
	}
}

// Update reconciles the running proxies with the current set of clusters.
// Starts proxies for new clusters, stops proxies for removed ones.
func (pm *ProxyManager) Update(clusters []Cluster) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	wanted := make(map[string]Cluster, len(clusters))
	for _, c := range clusters {
		wanted[c.FQDN] = c
	}

	// Stop proxies for clusters that disappeared
	for fqdn, ep := range pm.endpoints {
		if _, ok := wanted[fqdn]; !ok {
			log.Printf("stopping proxy for %s", fqdn)
			ep.server.Close()
			ep.listener.Close()
			delete(pm.endpoints, fqdn)
		}
	}

	// Start proxies for new clusters
	for fqdn, c := range wanted {
		if _, ok := pm.endpoints[fqdn]; ok {
			continue
		}
		ep, err := pm.startProxy(c)
		if err != nil {
			log.Printf("failed to start proxy for %s: %v", fqdn, err)
			continue
		}
		pm.endpoints[fqdn] = ep
		log.Printf("proxy for %s listening on %s", fqdn, ep.LocalAddr)
	}
}

// Endpoints returns the current set of proxy endpoints.
func (pm *ProxyManager) Endpoints() []ProxyEndpoint {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	eps := make([]ProxyEndpoint, 0, len(pm.endpoints))
	for _, ep := range pm.endpoints {
		eps = append(eps, *ep)
	}
	return eps
}

func (pm *ProxyManager) startProxy(c Cluster) (*ProxyEndpoint, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	target, _ := url.Parse("https://" + c.FQDN)
	transport := &http.Transport{
		DialContext: pm.srv.Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // API server proxy uses Tailscale certs
		},
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport

	server := &http.Server{Handler: proxy}
	go server.Serve(ln)

	return &ProxyEndpoint{
		Cluster:   c,
		LocalAddr: ln.Addr().String(),
		listener:  ln,
		server:    server,
	}, nil
}

// Close shuts down all proxies.
func (pm *ProxyManager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, ep := range pm.endpoints {
		ep.server.Close()
		ep.listener.Close()
	}
}
