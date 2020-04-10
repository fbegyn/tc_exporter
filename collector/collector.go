package tccollector

import (
	"net"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "tc"

type TcCollector struct {
	logger     log.Logger
	netns      map[int][]*net.Interface
	Collectors []prometheus.Collector
}

func NewTcCollector(netns map[int][]*net.Interface, logger log.Logger) (prometheus.Collector, error) {
	collectors := []prometheus.Collector{}
	// Setup Qdisc collector for interface
	qColl, err := NewQdiscCollector(netns, logger)
	if err != nil {
		return nil, err
	}
	collectors = append(collectors, qColl)
	// Setup Class collector for interface
	cColl, err := NewClassCollector(netns, logger)
	if err != nil {
		return nil, err
	}
	collectors = append(collectors, cColl)
	// Setup Service Curve collector for interface
	scColl, err := NewServiceCurveCollector(netns, logger)
	if err != nil {
		return nil, err
	}
	collectors = append(collectors, scColl)

	return &TcCollector{
		logger:     logger,
		netns:      netns,
		Collectors: collectors,
	}, nil
}

func (t TcCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, col := range t.Collectors {
		col.Describe(ch)
	}
}

func (t TcCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(t.Collectors))
	t.logger.Log("msg", "processing scrape")
	for _, coll := range t.Collectors {
		go func(c prometheus.Collector) {
			c.Collect(ch)
			wg.Done()
		}(coll)
	}
	wg.Wait()
}
