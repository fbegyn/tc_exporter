# TC exporter

[![builds.sr.ht status](https://builds.sr.ht/~fbegyn/tc_exporter.svg)](https://builds.sr.ht/~fbegyn/tc_exporter?) [![Go Report Card](https://goreportcard.com/badge/github.com/fbegyn/tc_exporter)](https://goreportcard.com/report/github.com/fbegyn/tc_exporter)

`tc_exporter` is an Prometheus exporter that is capable of exporting statisctics from `tc`
through the netlink library.

It was created from the need of being capable of monitoring the TC statistics that can be seen when
running `tc -s` in a modern way.

## Getting started

### Prerequisites

* Linux (netlink/`tc` support)
* Root privileges or `CAP_NET_ADMIN` (required to read netlink/tc data)
* Go 1.24+ (only needed to build from source)

### Build

```sh
make build
```

This produces the binary at `bin/tc_exporter`. Alternatively, build directly with:

```sh
go build -o bin/tc_exporter ./cmd/tc_exporter
```

### Configure

Create a `config.toml` (see [below](#configtoml)). The exporter looks for it, in order, in:

* `/etc/tc-exporter/`
* `$HOME/.tc-exporter/`
* the current working directory
* the path passed via `--config-file`

### Run

```sh
sudo ./bin/tc_exporter --config-file .
```

Metrics are then served at `http://localhost:9704/metrics` (or whichever `--listen-address` /
`listen-address` is configured).

Run `./bin/tc_exporter --help` to see all flags, including `--log-level`, `--listen-address` and
`--collector-<name>` / `--no-collector-<name>` to toggle individual qdisc collectors.

### Run as a systemd service

Packages built via `make package` (using [nfpm](https://nfpm.goreleaser.com/)) install a
`tc_exporter.service` unit and enable it automatically. To set it up manually instead, copy
`tc_exporter.service` to `/etc/systemd/system/`, install the binary and config file, then run:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now tc_exporter.service
```

## config.toml

It is possible to filter the interface to fetch data from by using a `config.toml` with the following
structure and keys

* `listen-address`: specifies the address on which the exporter will be running
* `log-level`: specifies the log level based on [slog levels](https://pkg.go.dev/log/slog#Level)
   ```go
   const (
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
   )
   ```
* `[netns.<netns name>]`: Map that specifies which network namespaces to monitor by name
  * `interfaces`: string array with the names of the interfaces that should be exported

```
listen-address = ":9704"
log-level = 0

[netns.default]
interfaces = ['dummy','eno1']

[netns.netns01]
interfaces = ['dummy01']
```
