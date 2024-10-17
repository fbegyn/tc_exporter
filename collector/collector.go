package tccollector

import (
	"log/slog"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "tc"

// TcCollector is the object that will collect TC data for the interface
type TcCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
	Collectors []TcSubCollector
}

// NewTcCollector create a new TcCollector given a network interface
func NewTcCollector(netns map[string][]rtnetlink.LinkMessage, collectorEnables map[string]bool, logger *slog.Logger) (prometheus.Collector, error) {
	collectors := []TcSubCollector{}

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

	// add additional collectors
	for collector, enabled := range collectorEnables {
		if enabled {
			switch collector {
			case "cbq":
				coll, err := NewCbqCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "choke":
				coll, err := NewChokeCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "codel":
				coll, err := NewCodelCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "fq":
				coll, err := NewFqCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "fq_codel":
				coll, err := NewFqCodelQdiscCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "hfsc_qdisc":
				coll, err := NewHfscCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "service_curve":
				coll, err := NewServiceCurveCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "htb":
				coll, err := NewHtbCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "pie":
				coll, err := NewPieCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "red":
				coll, err := NewRedCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "sfb":
				coll, err := NewSfbCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
			case "sfq":
				coll, err := NewSfqCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
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
	t.logger.Debug("starting metrics scrape")
	t.logger.Debug("starting data collection")
	objects := make(map[string]map[string][]tc.Object)
	for ns, devices := range t.netns {
		objects[ns] = make(map[string][]tc.Object)
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qds, err := getQdiscs(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}
			objects[ns]["qdisc"] = append(objects[ns]["qdisc"], qds...)
			// fetch all the the qdisc for this interface
			cls, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get classes", "interface", interf.Attributes.Name, "err", err)
			}
			objects[ns]["class"] = append(objects[ns]["class"], cls...)
			// fetch all the the qdisc for this interface
			fls, err := getFilters(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get filters", "interface", interf.Attributes.Name, "err", err)
			}
			objects[ns]["filter"] = append(objects[ns]["filter"], fls...)
		}
	}
	t.logger.Debug("data collection complete")
	for _, coll := range t.Collectors {
		coll.Collect(ch, objects)
	}
	t.logger.Info("metric scrape complete")
}

type TcSubCollector interface {
	Describe(chan<- *prometheus.Desc)
	Collect(chan<- prometheus.Metric, map[string]map[string][]tc.Object)
}
