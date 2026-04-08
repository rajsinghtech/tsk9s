# tsk9s

k9s in the browser over Tailscale. Connects to Kubernetes API server proxies on your tailnet via [tsnet](https://pkg.go.dev/tailscale.com/tsnet).

## Install

```bash
brew install rajsinghtech/tap/tsk9s
```

Or with Go:
```bash
go install github.com/rajsinghtech/tsk9s@latest
```

Or with Docker:
```bash
docker run -v tsk9s-state:/data ghcr.io/rajsinghtech/tsk9s:latest --endpoints "cluster1.example.ts.net"
```

Requires [k9s](https://k9scli.io/) on `$PATH` (except Docker, which bundles it).

## Usage

```
TS_AUTHKEY=tskey-auth-... tsk9s --endpoints "cluster1.example.ts.net,cluster2.example.ts.net"
```

Then open `http://tsk9s.<your-tailnet>.ts.net` in a browser.

### Flags

```
--endpoints    comma-separated API server proxy FQDNs
--state-dir    tsnet state directory (default: ./tsk9s-state)
--hostname     tailnet hostname (default: tsk9s)
```

`TS_AUTHKEY` is only needed on first run. State is persisted in `--state-dir`.

## How it works

- Joins your tailnet as a tsnet node
- Starts a local reverse proxy per cluster endpoint (dials through tsnet)
- Generates a merged kubeconfig pointing at the local proxies
- Serves a web terminal (xterm.js) running k9s on port 80
