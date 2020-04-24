# TC exporter

[![builds.sr.ht status](https://builds.sr.ht/~fbegyn/tc_exporter.svg)](https://builds.sr.ht/~fbegyn/tc_exporter?) [![Go Report Card](https://goreportcard.com/badge/github.com/fbegyn/tc_exporter)](https://goreportcard.com/report/github.com/fbegyn/tc_exporter)

`tc_exporter` is an Prometheus exporter that is capable of exporting statisctics from `tc`
through the netlink library.

It was created from the need of being capable of monitoring the TC statistics that can be seen when
running `tc -s` in a modern way.

## config.toml

It is possible to filter the interface to fetch data from by using a `config.toml` with the following
structure and keys

* `listen-address`: specifies the address on which the exporter will be running
* `[netns.<netns name>]`: Map that specifies which network namespaces to monitor by name
  * `interfaces`: string array with the names of the interfaces that should be exported

```
listen-address = ":9704"

[netns.default]
interfaces = ['dummy','eno1']

[netns.netns01]
interfaces = ['dummy01']
```
