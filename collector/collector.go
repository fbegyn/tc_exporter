package tccollector

import (
	"log/slog"
	"os"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "tc"

// TcCollector is the object that will collect TC data for the interface
type TcCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
	Collectors map[string]ObjectCollector
}

type ObjectCollector interface {
	Describe(chan<- *prometheus.Desc)
	CollectObject(ch chan<- prometheus.Metric, hostname, ns string, interf rtnetlink.LinkMessage, qd tc.Object)
}

// NewTcCollector create a new TcCollector given a network interface
func NewTcCollector(netns map[string][]rtnetlink.LinkMessage, collectorEnables map[string]bool, logger *slog.Logger) (prometheus.Collector, error) {
	collectors := map[string]ObjectCollector{}

	// Setup Qdisc collector for interface
	qColl, err := NewQdiscCollector(netns, logger)
	if err != nil {
		return nil, err
	}
	collectors["qdisc"] = qColl
	// Setup Class collector for interface
	cColl, err := NewClassCollector(netns, logger)
	if err != nil {
		return nil, err
	}
	collectors["class"] = cColl

	// add additional collectors
	for collector, enabled := range collectorEnables {
		if enabled {
			switch collector {
			case "cbq":
				logger.Debug("registering collector", "collector", "cbq", "key", "cbq")
				coll, err := NewCbqCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["cbq"] = coll
			case "choke":
				logger.Debug("registering collector", "collector", "choke", "key", "choke")
				coll, err := NewChokeCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["choke"] = coll
			case "codel":
				logger.Debug("registering collector", "collector", "codel", "key", "codel")
				coll, err := NewCodelCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["codel"] = coll
			case "fq":
				logger.Debug("registering collector", "collector", "fq", "key", "fq")
				coll, err := NewFqCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["fq"] = coll
			case "fq_codel":

				logger.Debug("registering collector", "collector", "fq_codel", "key", "fq_codel")
				coll, err := NewFqCodelQdiscCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["fq_codel"] = coll
			case "hfsc_qdisc":
				logger.Debug(
					"registering collector",
					"collector", "hfsc",
					"component", "qdisc",
					"key", "hfsc_qdisc",
				)
				coll, err := NewHfscCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["hfsc_qdisc"] = coll
			case "service_curve":
				logger.Debug(
					"registering collector",
					"collector", "hfsc",
					"component", "service curve",
					"key", "service_curve",
				)
				coll, err := NewServiceCurveCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["service_curve"] = coll
			case "htb":
				logger.Debug("registering collector", "collector", "htb", "key", "htb")
				coll, err := NewHtbCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["htb"] = coll
			case "pie":
				logger.Debug("registering collector", "collector", "pie", "key", "pie")
				coll, err := NewPieCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["pie"] = coll
			case "red":
				logger.Debug("registering collector", "collector", "red", "key", "red")
				coll, err := NewRedCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["red"] = coll
			case "sfb":
				logger.Debug("registering collector", "collector", "sfb", "key", "sfb")
				coll, err := NewSfbCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["sfb"] = coll
			case "sfq":
				logger.Debug("registering collector", "collector", "sfq", "key", "sfq")
				coll, err := NewSfqCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["sfq"] = coll
			}
		}
	}

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
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		t.logger.Error("failed to fetch hostname", "err", err)
	}

	t.logger.Debug("starting metrics scrape")
	// iterate through the netns and devices
	for ns, devices := range t.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qdiscs, err := getQdiscs(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}
			for _, qd := range qdiscs {
				t.logger.Debug("qdisc type", "kind", qd.Kind)
				t.logger.Debug("passing qdisc to qdisc collector", "qdisc", qd)
				t.Collectors["qdisc"].CollectObject(ch, host, ns, interf, qd)
				if qd.XStats == nil {
					t.logger.Debug("XStats struct is empty for this qdisc", "qdisc", qd, "interface", interf.Attributes.Name)
					continue
				}
				switch qd.Kind {
				case "cbq":
					t.logger.Debug("passing qdisc to cbq collector", "qdisc", qd)
					t.Collectors["cbq"].CollectObject(ch, host, ns, interf, qd)
				case "choke":
					t.logger.Debug("passing qdisc to cbq collector", "qdisc", qd)
					t.Collectors["choke"].CollectObject(ch, host, ns, interf, qd)
				case "codel":
					t.logger.Debug("passing qdisc to codel collector", "qdisc", qd)
					t.Collectors["codel"].CollectObject(ch, host, ns, interf, qd)
				case "fq":
					t.logger.Debug("passing qdisc to fq collector", "qdisc", qd)
					t.Collectors["fq"].CollectObject(ch, host, ns, interf, qd)
				case "fq_codel":
					t.logger.Debug("passing qdisc to fq_codel collector", "qdisc", qd)
					t.Collectors["fq_codel"].CollectObject(ch, host, ns, interf, qd)
				case "hfsc_qdisc":
					t.logger.Debug("passing qdisc to hfsc collector", "qdisc", qd)
					t.Collectors["hfsc_qdisc"].CollectObject(ch, host, ns, interf, qd)
				case "service_curve":
					t.logger.Debug("passing qdisc to serivce curve collector", "qdisc", qd)
					t.Collectors["service_curve"].CollectObject(ch, host, ns, interf, qd)
				case "htb":
					t.logger.Debug("passing qdisc to htb collector", "qdisc", qd)
					t.Collectors["htb"].CollectObject(ch, host, ns, interf, qd)
				case "pie":
					t.logger.Debug("passing qdisc to pie collector", "qdisc", qd)
					t.Collectors["pie"].CollectObject(ch, host, ns, interf, qd)
				case "red":
					t.logger.Debug("passing qdisc to red collector", "qdisc", qd)
					t.Collectors["red"].CollectObject(ch, host, ns, interf, qd)
				case "sfb":
					t.logger.Debug("passing qdisc to sfb collector", "qdisc", qd)
					t.Collectors["sfb"].CollectObject(ch, host, ns, interf, qd)
				case "sfq":
					t.logger.Debug("passing qdisc to sfq collector", "qdisc", qd)
					t.Collectors["sfq"].CollectObject(ch, host, ns, interf, qd)
				default:
					t.logger.Debug("no specific exporter for qdisc", "qdisc", qd)
				}
			}

			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}
			for _, cl := range classes {
				t.logger.Debug("class type", "kind", cl.Kind)
				t.logger.Debug("passing class to class collector", "class", cl)
				t.Collectors["class"].CollectObject(ch, host, ns, interf, cl)
				if cl.XStats == nil {
					t.logger.Debug("XStats struct is empty for this class", "class", cl, "interface", interf.Attributes.Name)
					continue
				}
				switch cl.Kind {
				case "hfsc":
					t.logger.Debug("passing class to hfsc collector", "class", cl)
					t.Collectors["hfsc"].CollectObject(ch, host, ns, interf, cl)
					t.logger.Debug("passing class to hfsc service curve collector", "class", cl)
					t.Collectors["service_curve"].CollectObject(ch, host, ns, interf, cl)
				default:
					t.logger.Debug("no specific exporter for class", "class", cl)
				}
			}
		}
	}
	// for _, coll := range t.Collectors {
	// 	coll.Collect(ch)
	// }
	t.logger.Debug("metric scrape complete")
}
