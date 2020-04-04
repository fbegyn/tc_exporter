prefix = /usr/sbin

all: build

build:
	go build -o bin/tc_exporter -ldflags "-w -s" ./cmd/tc_exporter

clean:
	rm -rf bin/tc_exporter

package: build
	nfpm pkg --target tc_exporter.deb
