package tccollector

import (
	"log/slog"
	"sync"

	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "tc"

// TcCollector is the object that will collect TC data for the interface
type TcCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
	Collectors []prometheus.Collector
}

// NewTcCollector create a new TcCollector given a network interface
func NewTcCollector(netns map[string][]rtnetlink.LinkMessage, logger *slog.Logger) (prometheus.Collector, error) {
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
		logger:     *logger,
		netns:      netns,
		Collectors: collectors,
	}, nil
}

// Describe implements Collector
func (t TcCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, col := range t.Collectors {
		col.Describe(ch)
	}
}

// Collect fetches and updates the data the collector is exporting
func (t TcCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(t.Collectors))
	t.logger.Info("processing scrape")
	for _, coll := range t.Collectors {
		go func(c prometheus.Collector) {
			c.Collect(ch)
			wg.Done()
		}(coll)
	}
	wg.Wait()
}
