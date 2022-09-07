prefix = /usr/sbin

all: package

build:
	go build -o bin/tc_exporter -ldflags "-w -s\
		-X main.Branch=$(shell git rev-parse --abbrev-ref HEAD)\
		-X main.Revision=$(shell git rev-list -1 HEAD)\
		-X main.Version=$(shell cat ./VERSION)" \
		./cmd/tc_exporter

clean:
	rm -rf bin/tc_exporter

package: build
	nfpm pkg --target tc_exporter.deb
	nfpm pkg --target tc_exporter.rpm
