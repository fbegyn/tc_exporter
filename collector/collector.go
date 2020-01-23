package collector

import (
	"sync"
	"time"

	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

const namespace = "tc"

var (
	hfscSC = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_hfsc_sc",
			Help: "Service curve for the hfsc class",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf", "sc", "param"},
	)
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"node_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"node_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

type TcCollector struct {
	Collectors map[string]Collector
}

func NewTcCollector() (*TcCollector, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}
	collectors := make(map[string]Collector)
	for _, link := range links {
		name := link.Attrs().Name
		dc, err := NewDataCollector(link)
		if err != nil {
			return nil, err
		}
		collectors[name] = dc
	}
	return &TcCollector{collectors}, nil
}

func (t TcCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

func (t TcCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(t.Collectors))
	for name, c := range t.Collectors {
		go func(name string, c Collector) {
			execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

type Collector interface {
	Update(ch chan<- prometheus.Metric) error
}

func execute(name string, c Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		logrus.Errorf("failed collection for %s: %v", name, err)
		success = 0
	} else {
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

type dataCollector struct {
	name    string
	Link    Collector
	Qdiscs  []Collector
	Classes []Collector
}

func NewDataCollector(link netlink.Link) (Collector, error) {
	data := GetData(link)
	name := link.Attrs().Name
	linkc, err := NewLinkCollector(link)
	if err != nil {
		return nil, err
	}

	var qds []Collector
	for _, qd := range *data.qdiscs {
		var col Collector
		switch qd.Type() {
		default:
			col, err = NewGenericQdiscCollector(qd, name)
			if err != nil {
				return nil, err
			}
		}
		qds = append(qds, col)
	}

	var cls []Collector
	for _, cl := range *data.classes {
		var col Collector
		switch cl.Type() {
		default:
			col, err = NewGenericClassCollector(cl, name)
			if err != nil {
				return nil, err
			}
		}
		cls = append(cls, col)
	}

	return &dataCollector{name, linkc, qds, cls}, nil
}

func (d *dataCollector) Update(ch chan<- prometheus.Metric) error {
	d.Link.Update(ch)
	for _, qdisc := range d.Qdiscs {
		qdisc.Update(ch)
	}
	for _, class := range d.Classes {
		class.Update(ch)
	}
	return nil
}