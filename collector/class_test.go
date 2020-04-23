package tccollector_test

import (
	"os"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/go-kit/kit/log"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/promtest"
	"golang.org/x/sys/unix"
)

// TestClassCollector tests out the creation and polling of a ClassCollector
func TestClassCollector(t *testing.T) {
	// Define the test case
	tests := []struct {
		ns     string
		name   string
		linkid uint32
	}{
		{ns: "default", name: "dummydefault", linkid: 999},
		{ns: "testing01", name: "dummy01", linkid: 1000},
		{ns: "testing02", name: "dummy1000", linkid: 1001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup the netns for testing
			if tt.ns != "default" {
				shell(t, "ip", "netns", "add", tt.ns)
				// later on, we delete the netns since it is no longer usefull
				defer shell(t, "ip", "netns", "del", tt.ns)

				// Check if the namespace was actually created
				f, err := os.Open("/var/run/netns/" + tt.ns)
				if err != nil {
					t.Fatalf("failed to open namespace file: %v", err)
				}
				// Close the fd if we no longer need it
				defer f.Close()
			}

			// create the dummy interface in the netns
			rtnl, err := setupDummyInterface(t, tt.ns, tt.name, tt.linkid)
			if err != nil {
				t.Fatalf("could not setup %s interface for %s: %v", tt.name, tt.ns, err)
			}
			// close the returned rtnetlink connection if no longer needed
			defer rtnl.Close()

			// Fetch the test interface from the netns
			interf, err := getLinkByName(tt.ns, tt.name)
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("could not get %s interface by name", tt.name)
			}

			// Compose the test interface and netns "config"
			test := make(map[string][]rtnetlink.LinkMessage)
			// We fetcha ll devices in the default netns of the OS
			con, _ := tcexporter.GetNetlinkConn("default")
			links, _ := con.Link.List()
			test["default"] = links
			// We fetch all devices in the newly created netns
			con, _ = tcexporter.GetNetlinkConn(tt.ns)
			links, _ = con.Link.List()
			test[tt.ns] = links

			var logger log.Logger
			logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
			logger = log.With(logger, "test", "collector")

			// Create a ClassCollector with the test "config"
			qc, err := tcexporter.NewClassCollector(test, logger)
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to create class collector for %s: %v", interf.Attributes.Name, err)
			}

			// Test out the functionality of the collector
			body := promtest.Collect(t, qc)
			if !promtest.Lint(t, body) {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Errorf("one or more promlint errors found")
			}
			t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
			rtnl.Link.Delete(tt.linkid)
		})
	}

}

// TestServiceCurveCollector tests out the creation and polling of a ServiceCurveCollector. It
// rquires some more work because we need to set up a class that actually uses a Service curve
func TestServiceCurveCollector(t *testing.T) {

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
				defer shell(t, "ip", "netns", "del", tt.ns)
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

			// Add HFSC qdisc
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
				t.Fatalf("failed to add HFSC qdisc: %v", err)
			}

			// Add hfsc Class
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
			var logger log.Logger
			logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
			logger = log.With(logger, "test", "collector")

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
					logger.Log("msg", "found HFSC class", "class", cl.Kind)
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
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to create Service Curve collector for %s: %v", interf.Attributes.Name, err)
			}

			// Check if the exporter returns data on the call
			body := promtest.Collect(t, qc)

			// Check if the returned body adheres to the Prometheus style
			if !promtest.Lint(t, body) {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Errorf("one or more promlint errors found")
			}
			t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
			rtnl.Link.Delete(tt.linkid)
		})
	}

}
