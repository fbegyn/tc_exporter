package tccollector

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	hfscLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// FqCollector is the object that will collect FQ qdisc data for the interface
type HfscCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
}

// NewFqCollector create a new QdiscCollector given a network interface
func NewHfscCollector(netns map[string][]rtnetlink.LinkMessage, fqcodellog *slog.Logger) (prometheus.Collector, error) {
	// Setup logger for qdisc collector
	fqcodellog = fqcodellog.With("collector", "fq_codel")
	fqcodellog.Info("making qdisc collector")

	return &FqCollector{
		logger: *fqcodellog,
		netns:  netns,
		gcFlows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq_codel", "gc_flows"),
			"FQ gc flow counter",
			hfscLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (qc *HfscCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (qc *HfscCollector) Collect(ch chan<- prometheus.Metric) {
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		qc.logger.Error("failed to fetch hostname", "err", err)
	}

	// iterate through the netns and devices
	for ns, devices := range qc.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qdiscs, err := getFQQdiscs(uint32(interf.Index), ns)
			if err != nil {
				qc.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}

			// iterate through all the qdiscs and sent the data to the prometheus metric channel
			for _, qd := range qdiscs {
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					qc.type,
					prometheus.CounterValue,
					float64(qd.XStats.Hfsc.Type),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
			}
		}
	}
}

// ServiceCurveCollector is the object that will collect Service Curve data for the interface. It is
// mainly used to determine the current limits imposed by the service curve
type ServiceCurveCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage
	curves map[string]*tc.ServiceCurve
	Burst  *prometheus.Desc
	Delay  *prometheus.Desc
	Rate   *prometheus.Desc
	Mutex  *sync.Mutex
}

// NewServiceCurveCollector create a new ServiceCurveCollector given a network interface
func NewServiceCurveCollector(netns map[string][]rtnetlink.LinkMessage, sclog *slog.Logger) (prometheus.Collector, error) {
	// Set up the logger for the service curve collector
	sclog = sclog.With("collector", "hfsc")
	sclog.Info("making SC collector")

	// We need an object to persust the different types of curves in for each HFSC class
	curves := make(map[string]*tc.ServiceCurve)

	return &ServiceCurveCollector{
		logger: *sclog,
		curves: curves,
		netns:  netns,
		Burst: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "service_curve", "burst"),
			"Burst parameter of the service curve",
			curvelabels, nil,
		),
		Delay: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "service_curve", "delay"),
			"Delay parameter of the service curve",
			curvelabels, nil,
		),
		Rate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "service_curve", "rate"),
			"Rate parameter of the service curve",
			curvelabels, nil,
		),
		Mutex: &sync.Mutex{},
	}, nil
}

// Describe implements Collector
func (c *ServiceCurveCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.Burst,
		c.Delay,
		c.Rate,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (c *ServiceCurveCollector) Collect(ch chan<- prometheus.Metric) {
	c.Mutex.Lock()
	// First we go and get the hostname of the system, so it can later be used in the labels
	host, err := os.Hostname()
	if err != nil {
		c.logger.Info("failed to get hostname", "err", err)
	}
	// start iterating over the defined namespaces and devices
	for ns, devices := range c.netns {
		// interate over each device, TODO: maybe there is a more elegant way to do this then 2 for
		// loops, I need a Go wizard to have a look at this.
		for _, interf := range devices {
			// Get all the classes for the interface
			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				c.logger.Info("failed to get classes", "interface", interf.Attributes.Name, "err", err)
			}

			// Iterate over each class
			for _, cl := range classes {s
				handleMaj, handleMin := HandleStr(cl.Handle)
				parentMaj, parentMin := HandleStr(cl.Parent)

				// If the class is a HFSC class, fetch the curve information. Otherwise continue to the next
				// class since we are only intrested in HFSC here
				if cl.Hfsc == nil {
					continue
				}
				c.curves["fsc"] = cl.Hfsc.Fsc
				c.curves["rsc"] = cl.Hfsc.Rsc
				c.curves["usc"] = cl.Hfsc.Usc

				// For each curve, report the settings of the service curve the channel
				for typ, sc := range c.curves {
					// If a certain type of curve is not defined, skip over it. It also functions as another
					// safety check so the function does not error on nil curves
					if sc == nil {
						continue
					}
					ch <- prometheus.MustNewConstMetric(
						c.Burst,
						prometheus.GaugeValue,
						float64(sc.M1),
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						typ,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						c.Delay,
						prometheus.GaugeValue,
						float64(sc.D),
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						typ,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						c.Rate,
						prometheus.GaugeValue,
						float64(sc.M2),
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						typ,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
				}
			}
		}
	}
	c.Mutex.Unlock()

}
