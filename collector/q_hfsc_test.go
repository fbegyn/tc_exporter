package tccollector_test

import (
	"log/slog"
	"os"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/jsimonetti/rtnetlink"
	"golang.org/x/sys/unix"
)

// TestServiceCurveCollector tests out the creation and polling of a ServiceCurveCollector. It
// rquires some more work because we need to set up a class that actually uses a Service curve
func TestServiceCurveCollector(t *testing.T) {
	// Define the test cases for the service collector
	tests := []struct {
		ns     string
		name   string
		linkid uint32
	}{
		{ns: "default", name: "dummydefault", linkid: 1000},
		{ns: "testing01", name: "dummy01", linkid: 1001},
		{ns: "testing02", name: "dummy1000", linkid: 1002},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup the netns for testing
			if tt.ns != "default" {
				shell(t, "ip", "netns", "add", tt.ns)
				// delete the netns when it is no longer needed
				defer shell(t, "ip", "netns", "del", tt.ns)
				// check if the netns was actually created
				f, err := os.Open("/var/run/netns/" + tt.ns)
				if err != nil {
					t.Fatalf("failed to open namespace file: %v", err)
				}
				defer f.Close()
			}

			// Setup dummy interface for testing
			rtnl, err := setupDummyInterface(t, tt.ns, tt.name, tt.linkid)
			if err != nil {
				t.Fatalf("could not setup %s interface for %s: %v", tt.name, tt.ns, err)
			}
			defer rtnl.Close()

			// Fetch the dummy interface
			interf, err := getLinkByName(tt.ns, tt.name)
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("could not get %s interface by name", tt.name)
			}

			// create the test "config"
			test := make(map[string][]rtnetlink.LinkMessage)
			con, _ := tcexporter.GetNetlinkConn("default")
			links, _ := con.Link.List()
			test["default"] = links

			// Create socket for interface to get and set classes
			sock, err := tcexporter.GetTcConn(tt.ns)
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("could not open rtnetlink socket: %v", err)
			}
			defer func() {
				if err := sock.Close(); err != nil {
					t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
					rtnl.Link.Delete(tt.linkid)
					t.Fatalf("could not close rtnetlink socket: %v", err)
				}
			}()

			// Since the interface spawns with the default class, we need to add an HFSC class to the
			// interface for this to work. To add an HFSC class, we first need to add an HFSC qdisc
			qmsg := tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(interf.Index),
				Handle:  core.BuildHandle(0x1, 0x0),
				Parent:  tc.HandleRoot,
				Info:    0,
			}
			err = sock.Qdisc().Add(&tc.Object{
				Msg: qmsg,
				Attribute: tc.Attribute{
					Kind: "hfsc",
					HfscQOpt: &tc.HfscQOpt{
						DefCls: 1,
					},
					Stab: &tc.Stab{
						Base: &tc.SizeSpec{
							CellLog:   0,
							SizeLog:   0,
							CellAlign: 0,
							Overhead:  0,
							LinkLayer: 1,
							MPU:       0,
							MTU:       1500,
							TSize:     0,
						},
					},
				},
			})
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to create HFSC qdisc: %v", err)
			}

			// After the HFSC qdisc has been created, we can add the HFSC class
			cmsg := tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(interf.Index),
				Handle:  core.BuildHandle(0x1, 0x1),
				Parent:  core.BuildHandle(0x1, 0x0),
				Info:    0,
			}
			err = sock.Class().Add(&tc.Object{
				Msg: cmsg,
				Attribute: tc.Attribute{
					Kind: "hfsc",
					Hfsc: &tc.Hfsc{
						Rsc: &tc.ServiceCurve{
							M1: 0,
							D:  0,
							M2: 10e6,
						},
						Fsc: &tc.ServiceCurve{
							M1: 0,
							D:  0,
							M2: 10e6,
						},
						Usc: &tc.ServiceCurve{
							M1: 0,
							D:  0,
							M2: 10e6,
						},
					},
				},
			})
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to add HFSC class: %v", err)
			}

			// Setup a logger for the test collector
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			logger = logger.With("test", "class")

			// Fetch classes and select a HFSC class
			classes, err := sock.Class().Get(&tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(interf.Index),
			})
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to get classes for %s: %v", interf.Attributes.Name, err)
			}

			// Filter out an HFSC class
			var cl tc.Object
			found := false
			for _, c := range classes {
				if c.Kind == "hfsc" {
					found = true
					cl = c
					logger.Info("found HFSC class", "class", cl.Kind)
					break
				}
			}

			if !found {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to find HFSC class")
			}

			// Create ServiceCurve collector for the class
			qc, err := tcexporter.NewServiceCurveCollector(test, logger)
			_ = qc
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to create Service Curve collector for %s: %v", interf.Attributes.Name, err)
			}

			// // Check if the exporter returns data on the call
			// body := promtest.Collect(t, qc)

			// // Check if the returned body adheres to the Prometheus style
			// if !promtest.Lint(t, body) {
			// 	t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
			// 	rtnl.Link.Delete(tt.linkid)
			// 	t.Errorf("one or more promlint errors found")
			// }
			t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
			rtnl.Link.Delete(tt.linkid)
		})
	}

}
