prefix = /usr/sbin

all: build

build:
	go build -o bin/tc_exporter -ldflags "-w -s" ./

clean:
	rm -rf bin/tc_exporter

package:
	nfpm pkg --target tc_exporter.deb
