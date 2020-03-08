package collector

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
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
	collectors := make(map[string]Collector)
	for _, interf := range interfaces {
		dc, err := NewDataCollector(interf)
		if err != nil {
			return nil, err
		}
		collectors[interf] = dc
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
	interf  *net.Interface
	sock    *tc.Tc
	Logger  log.Logger
	Qdiscs  []Collector
	Classes []Collector
}

func NewDataCollector(devName string) (Collector, error) {
	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open rtnetlink socket: %v\n", err)
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "could not close rtnetlink socket: %v\n", err)
		}
	}()

	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller, "component", "DataCollector")

	interf, err := net.InterfaceByName(devName)
	if err != nil {
		logger.Log("err", err)
	}

	coll := dataCollector{
		interf: interf,
		sock:   rtnl,
		Logger: logger,
	}

	qdiscs, classes := GetNetlinkData(coll.sock, uint32(coll.interf.Index), coll.Logger)

	for _, qd := range qdiscs {
		qc, err := NewQdiscCollector(coll.interf, qd)
		if err != nil {
			coll.Logger.Log("msg", "failed to initiate new collector", "err", err)
		}
		coll.Qdiscs = append(coll.Qdiscs, qc)
	}
	for _, cl := range classes {
		fmt.Println(cl.Kind)
	}

	return &coll, nil
}

func (d *dataCollector) Update(ch chan<- prometheus.Metric) error {
	for _, qdisc := range d.Qdiscs {
		qdisc.Update(ch)
	}
	for _, class := range d.Classes {
		class.Update(ch)
	}
	return nil
}
