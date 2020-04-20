package tccollector_test

import (
	"os"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/go-kit/kit/log"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/promtest"
)

func TestTcCollector(t *testing.T) {

	// setup the netns for testing
	shell(t, "ip", "netns", "add", "testing01")
	defer shell(t, "ip", "netns", "del", "testing01")
	shell(t, "ip", "netns", "add", "testing02")
	defer shell(t, "ip", "netns", "del", "testing02")

	rtnl1, err := setupDummyInterface(t, "testing01", "dummy01", 1000)
	rtnl2, err := setupDummyInterface(t, "testing02", "dummy02", 1001)
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

			test := make(map[string][]rtnetlink.LinkMessage)
			con, _ := tcexporter.GetNetlinkConn("default")
			links, _ := con.Link.List()
			test["default"] = links
			con, _ = tcexporter.GetNetlinkConn("testing01")
			links, _ = con.Link.List()
			test["testing01"] = links
			con, _ = tcexporter.GetNetlinkConn("testing02")
			links, _ = con.Link.List()
			test["testing02"] = links

			coll, err := tcexporter.NewTcCollector(test, logger)
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
