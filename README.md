# Outline ss-server

![Build Status](https://myoss/actions/workflows/go.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/myoss)](https://goreportcard.com/report/myoss)
[![Go Reference](https://pkg.go.dev/badge/myoss.svg)](https://pkg.go.dev/myoss)

[![Mattermost](https://badgen.net/badge/Mattermost/Outline%20Community/blue)](https://community.internetfreedomfestival.org/community/channels/outline-community)
[![Reddit](https://badgen.net/badge/Reddit/r%2Foutlinevpn/orange)](https://www.reddit.com/r/outlinevpn/)

This repository has the Shadowsocks service used by Outline servers. It was inspired by [go-shadowsocks2](https://github.com/shadowsocks/go-shadowsocks2), and adds a number of improvements to meet the needs of the Outline users.

The Outline Shadowsocks service allows for:
- Multiple users on a single port.
  - Does so by trying all the different credentials until one succeeds.
- Multiple ports
- Whitebox monitoring of the service using [prometheus.io](https://prometheus.io)
  - Includes traffic measurements and other health indicators.
- Live updates via config change + SIGHUP
- Replay defense (add `--replay_history 10000`).  See [PROBES](service/PROBES.md) for details.

![Graphana Dashboard](https://user-images.githubusercontent.com/113565/44177062-419d7700-a0ba-11e8-9621-db519692ff6c.png "Graphana Dashboard")


## Try it!

Fetch dependencies for this demo:
```
GO111MODULE=off go get github.com/prometheus/prometheus/cmd/...
```
If that doesn't work, download the [prometheus](https://prometheus.io/download/) binary directly.


### Run the server
On Terminal 1, from the repository directory, build and start the SS server:
```
go run ./cmd/outline-ss-server -config cmd/outline-ss-server/config_example.yml -metrics localhost:9091 --replay_history=10000
```
In production, you may want to specify `-ip_country_db` to get per-country metrics. See [how the Outline Server calls outline-ss-server](https://github.com/Jigsaw-Code/outline-server/blob/master/src/shadowbox/server/outline_shadowsocks_server.ts).


### Run the Prometheus scraper for metrics collection
On Terminal 2, start prometheus scraper for metrics collection:
```
$(go env GOPATH)/bin/prometheus --config.file=cmd/outline-ss-server/prometheus_example.yml
```

### Run the SOCKS-to-Shadowsocks client
On Terminal 3, start the SS client:
```
go run github.com/shadowsocks/go-shadowsocks2 -c ss://chacha20-ietf-poly1305:Secret0@:9000 -verbose  -socks localhost:1080
```

### Fetch a page over Shadowsocks
On Terminal 4, fetch a page using the SS client:
```
curl --proxy socks5h://localhost:1080 example.com
```

Stop and restart the client on Terminal 3 with "Secret1" as the password and try to fetch the page again on Terminal 4.

### Check the metrics
Open http://localhost:9091/metrics and see the exported Prometheus variables.

Open http://localhost:9090/ and see the Prometheus server dashboard.


## Performance Testing

Start the iperf3 server (runs on port 5201 by default):
```
iperf3 -s
```

Start the SS server (listening on port 9000):
```
go run ./cmd/outline-ss-server -config cmd/outline-ss-server/config_example.yml
```

Start the SS tunnel to redirect port 8000 -> localhost:5201 via the proxy on 9000:
```
go run github.com/shadowsocks/go-shadowsocks2 -c ss://chacha20-ietf-poly1305:Secret0@:9000 -tcptun ":8000=localhost:5201" -udptun ":8000=localhost:5201" -verbose
```

Test TCP upload (client -> server):
```
iperf3 -c localhost -p 8000
```

Test TCP download (server -> client):
```
iperf3 -c localhost -p 8000 --reverse
```

Test UDP upload:
```
iperf3 -c localhost -p 8000 --udp -b 0
```

Test UDP download:
```
iperf3 -c localhost -p 8000 --udp -b 0 --reverse
```

### Compare to go-shadowsocks2

Run the commands above, but start the SS server with
```
go run github.com/shadowsocks/go-shadowsocks2 -s ss://chacha20-ietf-poly1305:Secret0@:9000 -verbose
```


### Compare to shadowsocks-libev 

Start the SS server (listening on port 10001):
```
ss-server -s localhost -p 10001 -m chacha20-ietf-poly1305 -k Secret1 -u -v
```

Start the SS tunnel to redirect port 10002 -> localhost:5201 via the proxy on 10001:
```
ss-tunnel -s localhost -p 10001 -m chacha20-ietf-poly1305 -k Secret1 -l 10002 -L localhost:5201 -u -v
```

Run the iperf3 client tests listed above on port 10002.

You can mix and match the libev and go servers and clients.

## Tests and Benchmarks

To run the tests and benchmarks, call:
```
make test
```

You can benchmark the cipher finding code with
```
go test -cpuprofile cpu.prof -memprofile mem.prof -bench . -benchmem -run=^$ myoss/shadowsocks
```

You can inspect the CPU or memory profiles with `go tool pprof cpu.prof` or `go tool pprof mem.prof`, and then enter `web` on the prompt.

## Release

We use [GoReleaser](https://goreleaser.com/) to build and upload binaries to our [GitHub releases](https://myoss/releases).

Summary:
- Test the build locally:
  ```
  make release-local
  ```
- Export an environment variable named `GITHUB_TOKEN` with a temporary repo-scoped GitHub token ([create one here](https://github.com/settings/tokens/new)):
  ```bash
  export GITHUB_TOKEN=yournewtoken
  ```
- Create a new tag and push it to GitHub e.g.:
  ```bash
  git tag v1.0.0
  git push origin v1.0.0
  ```
- Build and upload:
  ```bash
  make release
  ```
- Go to https://myoss/releases, review and publish the release.

- Delete the Github token you created for the release on the [Personal Access Tokens page](https://github.com/settings/tokens).

Full instructions in [GoReleaser's Quick Start](https://goreleaser.com/quick-start) (jump to the section starting "You’ll need to export a GITHUB_TOKEN environment variable").
