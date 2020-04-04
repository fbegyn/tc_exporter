package tccollector

import (
	"net"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "tc"

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"node_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
)

type TcCollector struct {
	logger     log.Logger
	Collectors map[string][]prometheus.Collector
}

func NewTcCollector(interfaces []string, logger log.Logger) (prometheus.Collector, error) {
	// setup the logger for the collector
	collectors := make(map[string][]prometheus.Collector)
	for _, interf := range interfaces {
		device, err := net.InterfaceByName(interf)
		if err != nil {
			return nil, err
		}
		// Setup Qdisc collector for interface
		qColl, err := NewQdiscCollector(device, logger)
		if err != nil {
			return nil, err
		}
		collectors[interf] = append(collectors[interf], qColl)
		// Setup Class collector for interface
		cColl, err := NewClassCollector(device, logger)
		if err != nil {
			return nil, err
		}
		collectors[interf] = append(collectors[interf], cColl)
		// Setup Service Curve collector for interface
		scColl, err := NewServiceCurveCollector(device, logger)
		if err != nil {
			return nil, err
		}
		collectors[interf] = append(collectors[interf], scColl)
	}

	return &TcCollector{
		logger:     logger,
		Collectors: collectors,
	}, nil
}

func (t TcCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc

	for _, interf := range t.Collectors {
		for _, col := range interf {
			col.Describe(ch)
		}
	}
}

func (t TcCollector) Collect(ch chan<- prometheus.Metric) {

	collectors := 0
	for _, t := range t.Collectors {
		collectors += len(t)
	}

	wg := sync.WaitGroup{}
	wg.Add(collectors)
	for name, colls := range t.Collectors {
		for _, coll := range colls {
			go func(name string, c prometheus.Collector) {
				c.Collect(ch)
				wg.Done()
			}(name, coll)
		}
	}
	wg.Wait()
}
