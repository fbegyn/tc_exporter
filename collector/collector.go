package collector

import (
	"sync"
	"time"

	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const namespace = "tc"

var (
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

func NewTcCollector(interfaces []string) (*TcCollector, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	collectors := make(map[string]Collector)
	for _, link := range links {
		name := link.Attrs().Name
		if checkArray(name, interfaces) {
			dc, err := NewDataCollector(link)
			if err != nil {
				return nil, err
			}
			collectors[name] = dc
		}
	}
	return &TcCollector{collectors}, nil
}

func checkArray(str1 string, str2 []string) bool {
	for _, a := range str2 {
		if a == str1 {
			return true
		}
	}
	return false
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
		case "fq_codel":
			col, err = NewFqcodelQdiscCollector(qd, name)
			if err != nil {
				return nil, err
			}
		case "hfsc":
			col, err = NewHfscQdiscCollector(qd, name)
			if err != nil {
				return nil, err
			}
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
		case "hfsc":
			col, err = NewHfscClassCollector(cl, name)
			if err != nil {
				return nil, err
			}
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
