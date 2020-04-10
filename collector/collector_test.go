package tccollector

import (
	"net"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/mdlayher/promtest"
)

func TestTcCollector(t *testing.T) {
	rtnl1, err := SetupDummyInterface("dummy01", 1000)
	rtnl2, err := SetupDummyInterface("dummy02", 1001)
	if err != nil {
		t.Fatalf("could not setup dummy interface for testing: %v", err)
	}
	defer rtnl1.Close()
	defer rtnl2.Close()

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

			test := make(map[int][]*net.Interface)
			for _, device := range tt.devices {

				interf, err := net.InterfaceByName(device)
				if err != nil {
					t.Fatalf("could not get %s interface by name", tt.name)
				}
				test[0] = append(test[0], interf)
			}

			coll, err := NewTcCollector(test, logger)
			if err != nil {
				t.Fatalf("failed to create TC collector: %v", err)
			}
			body := promtest.Collect(t, coll)

			if !promtest.Lint(t, body) {
				t.Errorf("one or more promlint errors found")
			}

		})
	}
	rtnl1.Link.Delete(1000)
	rtnl2.Link.Delete(1001)
}
