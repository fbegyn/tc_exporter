package tccollector

import (
	"net"
	"os"
	"testing"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/go-kit/kit/log"
	"github.com/mdlayher/promtest"
	"golang.org/x/sys/unix"
)

func TestClassCollector(t *testing.T) {

	tests := []struct {
		name string
	}{
		{name: "dummy01"},
		{name: "dummy1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rtnl, err := SetupDummyInterface(tt.name)
			if err != nil {
				t.Fatalf("could not setup dummy interface for testing: %v", err)
			}
			defer rtnl.Close()

			interf, err := net.InterfaceByName(tt.name)
			if err != nil {
				t.Fatalf("could not get %s interface by name", tt.name)
			}

			var logger log.Logger
			logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
			logger = log.With(logger, "test", "collector")

			qc, err := NewClassCollector(interf, logger)
			if err != nil {
				t.Fatalf("failed to create class collector for %s: %v", interf.Name, err)
			}

			body := promtest.Collect(t, qc)

			if !promtest.Lint(t, body) {
				t.Errorf("one or more promlint errors found")
			}

			rtnl.Link.Delete(1000)
		})
	}

}

func TestServiceCurveCollector(t *testing.T) {

	tests := []struct {
		name string
	}{
		{name: "dummy01"},
		{name: "dummy1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup dummy interface for testing
			rtnl, err := SetupDummyInterface(tt.name)
			if err != nil {
				t.Fatalf("could not setup dummy interface for testing: %v", err)
			}
			defer rtnl.Close()

			// Fetch the dummy interface
			interf, err := net.InterfaceByName(tt.name)
			if err != nil {
				rtnl.Link.Delete(uint32(interf.Index))
				t.Fatalf("could not get %s interface by name", tt.name)
			}

			// Create socket for interface to get and set classes
			sock, err := tc.Open(&tc.Config{})
			if err != nil {
				rtnl.Link.Delete(uint32(interf.Index))
				t.Fatalf("could not open rtnetlink socket: %v", err)
			}
			defer func() {
				if err := sock.Close(); err != nil {
					rtnl.Link.Delete(uint32(interf.Index))
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
				rtnl.Link.Delete(uint32(interf.Index))
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
				rtnl.Link.Delete(uint32(interf.Index))
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
				rtnl.Link.Delete(uint32(interf.Index))
				t.Fatalf("failed to get classes for %s: %v", interf.Name, err)
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
				rtnl.Link.Delete(uint32(interf.Index))
				t.Fatalf("failed to find HFSC class")
			}

			// Create ServiceCurve collector for the class
			qc, err := NewServiceCurveCollector(interf, logger)
			if err != nil {
				rtnl.Link.Delete(uint32(interf.Index))
				t.Fatalf("failed to create Service Curve collector for %s: %v", interf.Name, err)
			}

			// Check if the exporter returns data on the call
			body := promtest.Collect(t, qc)
			// Check if the returned body adheres to the Prometheus style
			if !promtest.Lint(t, body) {
				t.Errorf("one or more promlint errors found")
			}

			// Clean up dummy interface
			rtnl.Link.Delete(uint32(interf.Index))
		})
	}

}
