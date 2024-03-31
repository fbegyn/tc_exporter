package tccollector_test

import (
	"log/slog"
	"os"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/jsimonetti/rtnetlink"
)

func TestTcCollector(t *testing.T) {
	// TODO: rework the test so it makes use of an array of tests (easier to expand an adjust)
	// setup the netns for testing
	shell(t, "ip", "netns", "add", "testing01")
	defer shell(t, "ip", "netns", "del", "testing01")
	shell(t, "ip", "netns", "add", "testing02")
	defer shell(t, "ip", "netns", "del", "testing02")

	rtnl1, err := setupDummyInterface(t, "testing01", "dummy01", 1000)
	if err != nil {
		t.Fatalf("could not setup dummy interface for testing: %v", err)
	}
	rtnl2, err := setupDummyInterface(t, "testing02", "dummy02", 1001)
	if err != nil {
		t.Fatalf("could not setup dummy interface for testing: %v", err)
	}
	defer rtnl1.Link.Delete(1001)
	defer rtnl2.Link.Delete(1002)
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
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			logger = logger.With("test", "collector")

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
			_ = coll
			if err != nil {
				t.Fatalf("failed to create TC collector: %v", err)
			}
			// body := promtest.Collect(t, coll)

			// if !promtest.Lint(t, body) {
			// 	t.Errorf("one or more promlint errors found")
			// }

		})
	}
}
