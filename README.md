# WIP TC exporter

[![builds.sr.ht status](https://builds.sr.ht/~fbegyn/tc_exporter.svg)](https://builds.sr.ht/~fbegyn/tc_exporter?)

`tc_exporter` is an InfluxDB/Prometheus exporter that is capable of exporting statisctics from `tc`
through the netlink library. Current project is a WIP and documentation will expand during the
development.

## config.toml

It is possible to filter the interface to fetch data from by using a `config.toml` with the following
structure.

```
interfaces = ['dummy-01','dummy-02']
```
