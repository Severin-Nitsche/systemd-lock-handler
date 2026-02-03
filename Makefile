DESTDIR?=/
PREFIX=/usr

dbus-systemd-dispatcher: main.go
	mkdir -p bin
	go build -ldflags '-s'
	mv dbus-systemd-dispatcher bin

sleep-plugin: plugins/sleep.go
	mkdir -p bin
	go build -buildmode=plugin plugins/sleep.go
	mv sleep.so bin

lock-plugin: plugins/lock.go
	mkdir -p bin
	go build -buildmode=plugin plugins/lock.go
	mv lock.so bin

plugins: sleep-plugin lock-plugin

build: dbus-systemd-dispatcher

install: build
	@install -Dm755 dbus-systemd-dispatcher ${DESTDIR}${PREFIX}/lib/dbus-systemd-dispatcher
	@install -Dm644 dbus-systemd-dispatcher.service ${DESTDIR}${PREFIX}/lib/systemd/user/dbus-systemd-dispatcher.service

.PHONY: build install plugins
