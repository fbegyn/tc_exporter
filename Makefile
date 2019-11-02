prefix = /usr/sbin

all: build

build:
	go build -o bin/tc_exporter -ldflags "-w -s"

clean:
	rm -rf bin/tc_exporter

install: build
	install -D bin/tc_exporter $(DESTDIR)$(prefix)/tc_exporter
	install -D tc_exporter@.service $(DESTDIR)/etc/systemd/system/tc_exporter@.service
	systemctl daemon-reload

uninstall:
	rm $(DESTDIR)$(prefix)/tc_exporter
	rm $(DESTDIR)/etc/systemd/system/tc_exporter@.service
	systemctl daemon-reload
