package tccollector

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	hfscLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// HfscCollector is the object that will collect hfsc qdisc data for the interface
type HfscCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage
	level  *prometheus.Desc
	period *prometheus.Desc
	rtWork *prometheus.Desc
	work   *prometheus.Desc
}

// NewHfscCollector create a new QdiscCollector given a network interface
func NewHfscCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "hfsc")
	log.Info("making hfsc collector")

	return &HfscCollector{
		logger: *log,
		netns:  netns,
		level: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "hfsc", "level"),
			"hfsc level xstat",
			hfscLabels, nil,
		),
		period: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "hfsc", "period"),
			"hfsc period xstat",
			hfscLabels, nil,
		),
		rtWork: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "hfsc", "rt_work"),
			"hfsc rtwork xstat",
			hfscLabels, nil,
		),
		work: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "hfsc", "work"),
			"hfsc work xstat",
			hfscLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *HfscCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.level,
		col.period,
		col.rtWork,
		col.work,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *HfscCollector) Collect(ch chan<- prometheus.Metric) {
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		col.logger.Error("failed to fetch hostname", "err", err)
	}

	// iterate through the netns and devices
	for ns, devices := range col.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qdiscs, err := getQdiscs(uint32(interf.Index), ns)
			if err != nil {
				col.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}

			// iterate through all the qdiscs and sent the data to the prometheus metric channel
			for _, qd := range qdiscs {
				if qd.Hfsc == nil || qd.XStats == nil {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.level,
					prometheus.CounterValue,
					float64(qd.XStats.Hfsc.Level),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.period,
					prometheus.CounterValue,
					float64(qd.XStats.Hfsc.Period),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.rtWork,
					prometheus.CounterValue,
					float64(qd.XStats.Hfsc.RtWork),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.work,
					prometheus.CounterValue,
					float64(qd.XStats.Hfsc.Work),
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

// CollectObject fetches and updates the data the collector is exporting
func (col *HfscCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	if qd.XStats == nil {
		return
	}

	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.level,
		prometheus.CounterValue,
		float64(qd.XStats.Hfsc.Level),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.period,
		prometheus.CounterValue,
		float64(qd.XStats.Hfsc.Period),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.rtWork,
		prometheus.CounterValue,
		float64(qd.XStats.Hfsc.RtWork),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.work,
		prometheus.CounterValue,
		float64(qd.XStats.Hfsc.Work),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
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
func NewServiceCurveCollector(netns map[string][]rtnetlink.LinkMessage, sclog *slog.Logger) (ObjectCollector, error) {
	// Set up the logger for the service curve collector
	sclog = sclog.With("collector", "service_curve")
	sclog.Info("making hfsc service curve collector")

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
	c.Mutex.Unlock()

}

// CollectObject fetches and updates the data the collector is exporting
func (c *ServiceCurveCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, cl tc.Object) {
	handleMaj, handleMin := HandleStr(cl.Handle)
	parentMaj, parentMin := HandleStr(cl.Parent)

	// If the class is a HFSC class, fetch the curve information. Otherwise continue to the next
	// class since we are only intrested in HFSC here
	if cl.Hfsc == nil {
		return
	}

	c.Mutex.Lock()
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
	c.Mutex.Unlock()
}
