package tccollector

import (
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/mdlayher/promtest"
)

func TestTcCollector(t *testing.T) {
	rtnl, err := SetupDummyInterface("dummy01")
	if err != nil {
		t.Fatalf("could not setup dummy interface for testing: %v", err)
	}
	defer rtnl.Close()

	tests := []struct {
		name    string
		devices []string
	}{
		{name: "dummy01", devices: []string{"dummy01"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logger log.Logger
			logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
			logger = log.With(logger, "test", "collector")
			coll, err := NewTcCollector(tt.devices, logger)
			if err != nil {
				t.Fatalf("failed to create TC collector: %v", err)
			}
			body := promtest.Collect(t, coll)

			if !promtest.Lint(t, body) {
				t.Errorf("one or more promlint errors found")
			}

		})
	}
	rtnl.Link.Delete(1000)
}
