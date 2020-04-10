package tccollector

import (
	"net"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/mdlayher/promtest"
)

func TestQdiscCollector(t *testing.T) {

	tests := []struct {
		name string
	}{
		{name: "dummy01"},
		{name: "dummy1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rtnl, err := SetupDummyInterface(tt.name, 1000)
			if err != nil {
				t.Fatalf("could not setup dummy interface for testing: %v", err)
			}
			defer rtnl.Close()

			interf, err := net.InterfaceByName(tt.name)
			if err != nil {
				t.Fatalf("could not get %s interface by name", tt.name)
			}

			test := make(map[int][]*net.Interface)
			test[0] = []*net.Interface{interf}

			var logger log.Logger
			logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
			logger = log.With(logger, "test", "collector")

			qc, err := NewQdiscCollector(test, logger)
			if err != nil {
				t.Fatalf("failed to create qdisc collector for %s: %v", interf.Name, err)
			}

			body := promtest.Collect(t, qc)

			if !promtest.Lint(t, body) {
				t.Errorf("one or more promlint errors found")
			}

			rtnl.Link.Delete(1000)
		})
	}

}
