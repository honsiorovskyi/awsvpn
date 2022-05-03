SOURCES := $(shell find . -name "*.go") go.mod
PREFIX := /usr/local

awsvpn.bin: $(SOURCES)
	go build -o awsvpn.bin main.go

.PHONY: install

install: awsvpn.bin
	install -m755 awsvpn.bin $(PREFIX)/bin/awsvpn

clean:
	rm -f awsvpn.bin