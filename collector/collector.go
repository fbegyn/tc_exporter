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
			case "hfsc_class":
				logger.Debug(
					"registering collector",
					"collector", "hfsc",
					"component", "class",
					"key", "hfsc_class",
				)
				coll, err := NewHfscCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["hfsc_class"] = coll
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
				logger.Debug("registering collector", "collector", "htb", "key", "htb_qdisc")
				coll, err := NewHtbCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["htb_qdisc"] = coll
				logger.Debug("registering collector", "collector", "htb", "key", "htb_class")
				coll, err = NewHtbCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors["htb_class"] = coll
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
				qcol, found := t.Collectors["qdisc"]
				if !found {
					t.logger.Error("qdisc collector is not running")
					continue
				}
				qcol.CollectObject(ch, host, ns, interf, qd)
				if qd.XStats == nil {
					t.logger.Debug("XStats struct is empty for this qdisc", "qdisc", qd, "interface", interf.Attributes.Name)
					continue
				}
				t.logger.Debug("passing qdisc to qdisc collector", "qdisc", qd)
				switch qd.Kind {
				case "cbq":
					col, found := t.Collectors["cbq"]
					if !found {
						t.logger.Error("cbq qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to cbq collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "choke":
					col, found := t.Collectors["choke"]
					if !found {
						t.logger.Error("choke qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to choke collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "codel":
					col, found := t.Collectors["codel"]
					if !found {
						t.logger.Error("codel qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to codel collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "fq":
					col, found := t.Collectors["fq"]
					if !found {
						t.logger.Error("fq qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to fq collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "fq_codel":
					col, found := t.Collectors["fq_codel"]
					if !found {
						t.logger.Error("fq_codel qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to fq_codel collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "hfsc":
					col, found := t.Collectors["hfsc_qdisc"]
					if !found {
						t.logger.Error("hfsc qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to hfsc collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "service_curve":
					col, found := t.Collectors["service_curve"]
					if !found {
						t.logger.Error("service_curve qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to serivce curve collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "htb":
					col, found := t.Collectors["htb_qdisc"]
					if !found {
						t.logger.Error("htb qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to htb collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "pie":
					col, found := t.Collectors["pie"]
					if !found {
						t.logger.Error("pie qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to pie collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "red":
					col, found := t.Collectors["red"]
					if !found {
						t.logger.Error("red qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to red collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "sfb":
					col, found := t.Collectors["sfb"]
					if !found {
						t.logger.Error("sfb qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to sfb collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				case "sfq":
					col, found := t.Collectors["sfq"]
					if !found {
						t.logger.Error("sfq qdisc collector is not running")
						continue
					}
					t.logger.Debug("passing qdisc to sfq collector", "qdisc", qd)
					col.CollectObject(ch, host, ns, interf, qd)
				default:
					t.logger.Info("no specific exporter for qdisc", "qdisc", qd)
				}
			}

			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}
			for _, cl := range classes {
				t.logger.Debug("class type", "kind", cl.Kind)
				ccol, found := t.Collectors["class"]
				if !found {
					t.logger.Error("class collector is not running")
					continue
				}
				ccol.CollectObject(ch, host, ns, interf, cl)
				if cl.XStats == nil {
					t.logger.Debug("XStats struct is empty for this class", "class", cl, "interface", interf.Attributes.Name)
					continue
				}
				t.logger.Debug("passing class to class collector", "class", cl)
				switch cl.Kind {
				case "htb":
					col, found := t.Collectors["htb_class"]
					if !found {
						t.logger.Error("htb class collector is not running")
						continue
					}
					t.logger.Debug("passing class to htb collector", "class", cl)
					col.CollectObject(ch, host, ns, interf, cl)
				case "hfsc":
					col, found := t.Collectors["hfsc_class"]
					if !found {
						t.logger.Error("hfsc class collector is not running")
						continue
					}
					t.logger.Debug("passing class to hfsc collector", "class", cl)
					col.CollectObject(ch, host, ns, interf, cl)
					col, found = t.Collectors["service_curve"]
					if !found {
						t.logger.Error("service_curve class collector is not running")
						continue
					}
					t.logger.Debug("passing class to hfsc service curve collector", "class", cl)
					col.CollectObject(ch, host, ns, interf, cl)
				default:
				}
			}
		}
	}
	// for _, coll := range t.Collectors {
	// 	coll.Collect(ch)
	// }
	t.logger.Debug("metric scrape complete")
}
