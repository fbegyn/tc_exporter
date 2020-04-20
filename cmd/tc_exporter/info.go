package main

import (
	"fmt"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
)

// Build information. Populated at build-time.
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion = runtime.Version()
)

// NewCollector returns a collector that exports metrics about current version
// information.
func NewVersionCollector(program string) prometheus.Collector {
	return prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: program,
			Name:      "build_info",
			Help: fmt.Sprintf(
				"A metric with a constant '1' value labeled by version, revision, branch, and goversion from which %s was built.",
				program,
			),
			ConstLabels: prometheus.Labels{
				"version":   Version,
				"revision":  Revision,
				"branch":    Branch,
				"goversion": GoVersion,
			},
		},
		func() float64 { return 1 },
	)
}
