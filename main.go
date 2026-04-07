package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"tailscale.com/tsnet"
)

func main() {
	stateDir := flag.String("state-dir", "./tsk9s-state", "tsnet state directory")
	hostname := flag.String("hostname", "tsk9s", "tailnet hostname")
	kubeconfigPath := flag.String("kubeconfig-path", filepath.Join(os.TempDir(), fmt.Sprintf("tsk9s-%d.kubeconfig", os.Getpid())), "kubeconfig output path")
	endpoints := flag.String("endpoints", "", "comma-separated list of k8s API server proxy FQDNs (e.g., ottawa-k8s-operator.keiretsu.ts.net,robbinsdale-k8s-operator.keiretsu.ts.net)")
	flag.Parse()

	if *endpoints == "" && flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "usage: tsk9s --endpoints=host1.ts.net,host2.ts.net\n")
		os.Exit(1)
	}

	// Parse endpoints from flag or positional args
	var epList []string
	if *endpoints != "" {
		for _, e := range strings.Split(*endpoints, ",") {
			if e = strings.TrimSpace(e); e != "" {
				epList = append(epList, e)
			}
		}
	}
	epList = append(epList, flag.Args()...)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv := &tsnet.Server{
		Hostname: *hostname,
		AuthKey:  os.Getenv("TS_AUTHKEY"),
		Dir:      *stateDir,
	}
	defer srv.Close()

	status, err := srv.Up(ctx)
	if err != nil {
		log.Fatalf("tsnet.Up: %v", err)
	}
	log.Printf("tsk9s online at %s (%v)", status.Self.DNSName, status.TailscaleIPs)

	disc := NewDiscovery(epList)

	proxyMgr := NewProxyManager(srv)
	defer proxyMgr.Close()

	proxyMgr.Update(disc.Clusters())
	if err := writeKubeconfig(*kubeconfigPath, proxyMgr.Endpoints()); err != nil {
		log.Printf("kubeconfig: %v", err)
	}

	ln, err := srv.Listen("tcp", ":80")
	if err != nil {
		log.Fatalf("listen :80: %v", err)
	}
	defer ln.Close()

	handler := newHandler(*kubeconfigPath, disc)

	log.Printf("serving on http://%s", status.Self.DNSName)
	go func() {
		if err := http.Serve(ln, handler); err != nil {
			log.Printf("http.Serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
}
