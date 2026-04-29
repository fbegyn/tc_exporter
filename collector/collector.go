package tccollector

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "tc"

// Factory builds an ObjectCollector for the given netns map and logger.
type Factory func(netns map[string][]rtnetlink.LinkMessage, logger *slog.Logger) (ObjectCollector, error)

var (
	factories        = map[string]Factory{}
	collectorState   = map[string]*bool{}
	forcedCollectors = map[string]bool{}
)

// registerCollector wires a collector into the registry: it defines the
// --collector.<name> kingpin flag, records whether the user explicitly set the
// flag (via the action), and stores the factory for later instantiation.
func registerCollector(name string, isDefaultEnabled bool, factory Factory) {
	helpDefault := "disabled"
	if isDefaultEnabled {
		helpDefault = "enabled"
	}
	flag := kingpin.Flag(
		fmt.Sprintf("collector.%s", name),
		fmt.Sprintf("Enable the %s collector (default: %s).", name, helpDefault),
	).Default(fmt.Sprintf("%v", isDefaultEnabled)).
		Action(func(*kingpin.ParseContext) error {
			forcedCollectors[name] = true
			return nil
		}).Bool()
	collectorState[name] = flag
	factories[name] = factory
}

// DisableDefaultCollectors flips every collector flag to false unless the user
// explicitly set it on the command line.
func DisableDefaultCollectors() {
	for name, st := range collectorState {
		if _, forced := forcedCollectors[name]; !forced {
			*st = false
		}
	}
}

// ApplyCollectorConfig overrides collector enable/disable state from values
// supplied by the config file. CLI-explicit flags (tracked in forcedCollectors)
// take precedence and are never overridden. Unknown collector names produce a
// WARN log and are skipped, so config typos don't silently disable a feature.
func ApplyCollectorConfig(cfg map[string]bool, logger *slog.Logger) {
	for name, enabled := range cfg {
		if _, forced := forcedCollectors[name]; forced {
			continue
		}
		state, ok := collectorState[name]
		if !ok {
			logger.Warn("unknown collector in [collector] config section", "name", name)
			continue
		}
		*state = enabled
	}
}

// TcCollector is the object that will collect TC data for the interface
type TcCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
	Collectors map[string]ObjectCollector
	Filters    FilterHolder
}

type FilterHolder struct {
	Qdisc []Filter `name:"qdisc"`
	Class []Filter `name:"class"`
}

type Filter struct {
	Kind   string `name:"kind"`
	Parent string `name:"parent"`
	Handle string `name:"handle"`
}

type ObjectCollector interface {
	Describe(chan<- *prometheus.Desc)
	CollectObject(ch chan<- prometheus.Metric, hostname, ns string, interf rtnetlink.LinkMessage, qd tc.Object)
}

// NewTcCollector creates a TcCollector with every collector whose kingpin flag
// is currently true. Call kingpin.Parse (and DisableDefaultCollectors if
// applicable) before calling this.
func NewTcCollector(netns map[string][]rtnetlink.LinkMessage, filters FilterHolder, logger *slog.Logger) (prometheus.Collector, error) {
	collectors := map[string]ObjectCollector{}
	for name, enabled := range collectorState {
		if !*enabled {
			continue
		}
		logger.Debug("registering collector", "collector", name)
		c, err := factories[name](netns, logger)
		if err != nil {
			return nil, fmt.Errorf("couldn't init collector %s: %w", name, err)
		}
		collectors[name] = c
	}

	return &TcCollector{
		logger:     *logger,
		netns:      netns,
		Collectors: collectors,
		Filters:    filters,
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
		QDISCS:
			for _, qd := range qdiscs {
				t.logger.Debug("qdisc type", "kind", qd.Kind, "handle", qd.Handle)
				for _, f := range t.Filters.Qdisc {
					kindmatch := qd.Kind == f.Kind && f.Kind != ""
					parentmatch := FmtHandleStr(qd.Parent) == f.Parent && f.Parent != ""
					if kindmatch || parentmatch {
						t.logger.Debug("skipping qdisc, it is in the filter list", "qdisc", qd)
						continue QDISCS
					}
				}
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
		CLASSES:
			for _, cl := range classes {
				t.logger.Debug("class type", "kind", cl.Kind, "classid", cl.Handle)
				for _, f := range t.Filters.Class {
					kindmatch := cl.Kind == f.Kind && f.Kind != ""
					parentmatch := FmtHandleStr(cl.Parent) == f.Parent && f.Parent != ""
					if kindmatch || parentmatch {
						t.logger.Debug("skipping class, it is in the filter list", "class", cl)
						continue CLASSES
					}
				}
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
					t.logger.Info("no specific exporter for class", "class", cl)
				}
			}
		}
	}
	t.logger.Debug("metric scrape complete")
}
