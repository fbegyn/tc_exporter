package tccollector

import (
	"fmt"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/log"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

var (
	classlabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
	curvelabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// ClassCollector is the object that will collect Class data for the interface
// It is a basic reperesentation of the Stats and Stats2 struct of iproute
type ClassCollector struct {
	logger     log.Logger
	netns      map[string][]rtnetlink.LinkMessage
	bytes      *prometheus.Desc
	packets    *prometheus.Desc
	bps        *prometheus.Desc
	pps        *prometheus.Desc
	backlog    *prometheus.Desc
	drops      *prometheus.Desc
	overlimits *prometheus.Desc
	qlen       *prometheus.Desc
	requeues   *prometheus.Desc
}

// NewClassCollector create a new ClassCollector given a network interface
func NewClassCollector(netns map[string][]rtnetlink.LinkMessage, clog log.Logger) (prometheus.Collector, error) {
	// Setup logger for the class collector
	clog = log.With(clog, "collector", "class")
	clog.Log("msg", "making class collector")

	return &ClassCollector{
		logger: clog,
		netns:  netns,
		bytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bytes_total"),
			"Class counter",
			classlabels, nil,
		),
		packets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "packets_total"),
			"Class packet counter",
			classlabels, nil,
		),
		bps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bps"),
			"Class byte rate",
			classlabels, nil,
		),
		pps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "pps"),
			"Class packet rate",
			classlabels, nil,
		),
		backlog: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "backlog_total"),
			"Class queue backlog",
			classlabels, nil,
		),
		drops: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "drops_total"),
			"Class queue drops",
			classlabels, nil,
		),
		overlimits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "overlimits_total"),
			"Class queue overlimits",
			classlabels, nil,
		),
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "qlen_total"),
			"Class queue length",
			classlabels, nil,
		),
		requeues: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "requeque_total"),
			"Class requeque counter",
			classlabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (cc *ClassCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		cc.backlog,
		cc.bps,
		cc.bytes,
		cc.packets,
		cc.drops,
		cc.overlimits,
		cc.pps,
		cc.qlen,
		cc.requeues,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (cc *ClassCollector) Collect(ch chan<- prometheus.Metric) {
	// First we go and get the hostname of the system, so it can later be used in the labels
	host, err := os.Hostname()
	if err != nil {
		cc.logger.Log("msg", "failed to get hostname", "err", err)
	}

	// start iterating over the defined namespaces and devices
	for ns, devices := range cc.netns {
		// interate over each device, TODO: maybe there is a more elegant way to do this then 2 for
		// loops, I need a Go wizard to have a look at this.
		for _, interf := range devices {
			// Get all TC classes  for the specified device
			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				cc.logger.Log("msg", "failed to get classes", "interface", interf.Attributes.Name, "err", err)
			}

			// Range over each class and report the statisctics of the class to the channel for Prometheus
			// metrics. Note that we print the handle with %x, so the hexadecimal notation. This way the
			// reported labels match the output from `tc -s show class ...`
			for _, cl := range classes {
				handleMaj, handleMin := HandleStr(cl.Handle)
				parentMaj, parentMin := HandleStr(cl.Parent)

				ch <- prometheus.MustNewConstMetric(
					cc.bytes,
					prometheus.CounterValue,
					float64(cl.Stats.Bytes),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.packets,
					prometheus.CounterValue,
					float64(cl.Stats.Packets),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.bps,
					prometheus.GaugeValue,
					float64(cl.Stats.Bps),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.pps,
					prometheus.GaugeValue,
					float64(cl.Stats.Pps),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.backlog,
					prometheus.CounterValue,
					float64(cl.Stats.Backlog),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.drops,
					prometheus.CounterValue,
					float64(cl.Stats.Drops),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.overlimits,
					prometheus.CounterValue,
					float64(cl.Stats.Overlimits),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.qlen,
					prometheus.CounterValue,
					float64(cl.Stats.Qlen),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.requeues,
					prometheus.CounterValue,
					float64(cl.Stats2.Requeues),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
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
	logger log.Logger
	netns  map[string][]rtnetlink.LinkMessage
	curves map[string]*tc.ServiceCurve
	Burst  *prometheus.Desc
	Delay  *prometheus.Desc
	Rate   *prometheus.Desc
}

// NewServiceCurveCollector create a new ServiceCurveCollector given a network interface
func NewServiceCurveCollector(netns map[string][]rtnetlink.LinkMessage, sclog log.Logger) (prometheus.Collector, error) {
	// Set up the logger for the service curve collector
	sclog = log.With(sclog, "collector", "hfsc")
	sclog.Log("msg", "making SC collector")

	// We need an object to persust the different types of curves in for each HFSC class
	curves := make(map[string]*tc.ServiceCurve)

	return &ServiceCurveCollector{
		logger: sclog,
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
	// First we go and get the hostname of the system, so it can later be used in the labels
	host, err := os.Hostname()
	if err != nil {
		c.logger.Log("msg", "failed to get hostname", "err", err)
	}

	// start iterating over the defined namespaces and devices
	for ns, devices := range c.netns {
		// interate over each device, TODO: maybe there is a more elegant way to do this then 2 for
		// loops, I need a Go wizard to have a look at this.
		for _, interf := range devices {
			// Get all the classes for the interface
			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				c.logger.Log("msg", "failed to get classes", "interface", interf.Attributes.Name, "err", err)
			}

			// Iterate over each class
			for _, cl := range classes {
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

}

// getClass fetches all the class of an interface in a certain network namespace (ns)
func getClasses(devid uint32, ns string) ([]tc.Object, error) {
	// Open a netlink TC socket for a specfied ns
	sock, err := GetTcConn(ns)
	if err != nil {
		return nil, err
	}
	// when the socket is no longer used, we close it
	defer sock.Close()

	// Get all classes for a specified interface
	classes, err := sock.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: devid,
	})
	if err != nil {
		return nil, err
	}

	// This is acutal a little hack. Since we return all classes for the interface, and it is possible
	// that HFSC class is used. Each flow through the HFSC class creates a fq_codel class with a random
	// handle, the cardinality of these metrics becomes very high in Prometheus
	// TODO: figure out a better solution for this, there has to be one
	var cl []tc.Object
	for _, class := range classes {
		if class.Ifindex == devid && class.Kind != "fq_codel" {
			cl = append(cl, class)
		}
	}
	return cl, nil
}
