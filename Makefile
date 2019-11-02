all: build

build:
	go build -o bin/tc_exporter -ldflags "-w -s"

clean:
	rm -rf bin/tc_exporter

install:
	cp bin/tc_exporter /usr/sbin/tc_exporter

uninstall:
	rm /usr/sbin/tc_exporter
