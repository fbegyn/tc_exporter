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
	Collectors []prometheus.Collector
}

// NewTcCollector create a new TcCollector given a network interface
func NewTcCollector(netns map[string][]rtnetlink.LinkMessage, collectorEnables map[string]bool, logger *slog.Logger) (prometheus.Collector, error) {
	collectors := []prometheus.Collector{}

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
	// Setup Service Curve collector for interface
	scColl, err := NewServiceCurveCollector(netns, logger)
	if err != nil {
		return nil, err
	}
	collectors = append(collectors, scColl)

	// add additional collectors
	for collector, enabled := range collectorEnables {
		if enabled {
			switch collector {
	                case "cbq":
				coll, err := NewCBQCollector(netns, logger)
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
				coll, err := NewFqCodelCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
	                case "hfsc":
				coll, err := NewHfscCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
	                case "htb":
				coll, err := NewHTBCollector(netns, logger)
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
				coll, err := NewSFBCollector(netns, logger)
				if err != nil {
					return nil, err
				}
				collectors = append(collectors, coll)
	                case "sfq":
				coll, err := NewSFQCollector(netns, logger)
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
	t.logger.Info("processing scrape, fetching data structures")

	qdiscs, classes, filters := make([]tc.Object,0),make([]tc.Object,0),make([]tc.Object,0)
	for ns, devices := range t.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qds, err := getQdiscs(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}
			qdiscs = append(qdiscs, qds...)

			// fetch all the the qdisc for this interface
			cls, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get classes", "interface", interf.Attributes.Name, "err", err)
			}
			classes = append(classes, cls...)

			// fetch all the the qdisc for this interface
			fls, err := getFilters(uint32(interf.Index), ns)
			if err != nil {
				t.logger.Error("failed to get filters", "interface", interf.Attributes.Name, "err", err)
			}
			filters = append(filters, fls...)
		}
	}

	t.logger.Info("data fetched, passing it along to the exports")
	for _, coll := range t.Collectors {
		coll.CollectWithData(ch, qdiscs)
	}
	t.logger.Info("scrape complete")
}
