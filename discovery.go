package main

import (
	"log"
	"strings"
	"sync"
)

// Cluster represents a k8s API server proxy endpoint.
type Cluster struct {
	Name string // human-readable, derived from hostname
	FQDN string // full MagicDNS name (e.g., "ottawa-k8s-operator.keiretsu.ts.net")
}

// Discovery holds the configured cluster endpoints.
type Discovery struct {
	mu       sync.RWMutex
	clusters []Cluster
}

func NewDiscovery(endpoints []string) *Discovery {
	d := &Discovery{}
	for _, ep := range endpoints {
		fqdn := strings.TrimSuffix(ep, ".")
		d.clusters = append(d.clusters, Cluster{
			Name: fqdn,
			FQDN: fqdn,
		})
	}
	log.Printf("configured %d cluster endpoint(s)", len(d.clusters))
	for _, c := range d.clusters {
		log.Printf("  %s (%s)", c.Name, c.FQDN)
	}
	return d
}

func (d *Discovery) Clusters() []Cluster {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return append([]Cluster(nil), d.clusters...)
}
