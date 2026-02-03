dbus-systemd-dispatcher
====================
This is derivative work based on `systemd-lock-handler` by Hugo Osvaldo Barrera.
Check out the original source
[here](https://git.sr.ht/~whynothugo/systemd-lock-handler)

The whole thing is based on the idea that some events such as (un)locking the
session or going into sleep are announced via the D-Bus but are not easily
accessible to the user space.

This software provides an interface to hook systemd targets onto these events.
It is fully configurable via yaml and provides the ability to even hook custom
logic into the dispatcher via the loading of dynamic libraries.

This allows for example the provided `sleep.target`-hook to hold on to a sleep
inhibitor lock to ensure the execution of dependant units.

Installation
------------

I don't know why you would install this manually.
When time comes, I plan however on writing a nice nix-module for this.

## Manual

You can (probably) manually build and install:

    cd dbus-systemd-dispatcher
    make build
    sudo make install

Usage
-----

The service itself must be enabled for the current user:

    systemctl --user enable --now dbus-systemd-dispatcher.service

Additionally, target files must be available for all configured targets and the
specified `.so` files for custom logic must be locateable.

Configuration
-------------

The program is configured via `configuration.yaml`.
All targets must be configured under the `targets` key as list entries with the
following keys:

|  key   |     description     |
|--------|---------------------|
| target | The target to start |
| dlib   | The dynamic library for custom logic |
| toggle | Whether to toggle the target upon repetition of the event |
| start  | Whether to (initially) start or stop the target |
| system | Whether to run the target as a system or user target |
| dbus   | The D-Bus [match rules](https://dbus.freedesktop.org/doc/dbus-specification.html#message-bus-routing-match-rules) for the event |

Consider this example for further reference:

```yaml
# config.yaml
targets:
  - target: "sleep.target"
    dlib: "bin/sleep.so"
    toggle: true
    start: true
    system: false
    dbus:
      path: "/org/freedesktop/login1"
      interface: "org.freedesktop.login1.Manager"
      member: "PrepareForSleep"
```

Custom Logic
------------

The `dlib` of a target, if provided provides flexible means of custom validation
logic. It is realized via go [plugins](https://pkg.go.dev/plugin), so watch out
with which version of the toolchain you compile your plugins vs. which version
of the program you use.

The symbol searched for in the export is `Hardcode`. The type signature should
be:

```go
func() (
  func(),
  func(*dbus.Conn, *dbus.Signal) bool,
  func(),
  func(),
)
```

where `dbus` refers to the [dbus
package](https://pkg.go.dev/github.com/godbus/dbus/v5).

The `Hardcode`-symbol will be called for every target that specified its
corresponding dynamic library. The return values of `Hardcode` are coined `init,
verify, before, after` in that order.

`init` will be executed once before the program listens to the dbus.
`verify` will be executed everytime a matching signal arrives. The corresponding
unit will only be executed/stopped if verify passes.

Considering the `before` and `after`, they have to be explained with the
`toggle` configuration.

If `toggle` is set to `false`, that is the target is configured to always
stop/start depending on `start`, `before` is executed everytime before waiting
for a verified D-Bus signal but not inbetween signals that got rejected by
`verify`. In this case `after` is never called.

If `toggle` is set to `true`, `after` will be called instead of `before` on
every second invocation. That is before waiting for the dbus signal that
initiates the action opposite of the one defined via `start` and *after* the
action defined via `start` was performed.

Changelog
---------

## 2.4.1

- Fixed regression introduced in 2.4.0.

## 2.4.0

- Sleeping will be now inhibited when `systemd-lock-handler` starts. This
  ensure that there is enough time to react before the system actually goes to
  sleep. See [this article] for some background on how this. See also the
  updated example in the README to ensure that your screen locker has actually
  locked the screen before sleeping continues.

[this article]: https://whynothugo.nl/journal/2022/10/26/systemd-locking-and-sleeping/

## 2.3.0

- `sleep.target` now requires `lock.target` itself. So for any services that
  should be started when either locking or suspending the system, specifying
  `WantedBy=lock.target` is enough.
- Fixed a bug where lock some services wouldn't be stopped after waking up
  and then unlocking a system.

## 2.2.0

- Also handle unlock events (and translate those to unlock.target).

## 2.1.0

- Minor bugfixes.
- Run as `Type=notify`.

## 2.0.0

- Rewrite in go.
- Move binary into /usr/lib.

## 1.1.0

- Use newer logind API.
- Events for other sessions are now correctly ignored.

## 1.0.0

Also handle sleep target.

## 0.1.0

Initial release.

LICENCE
-------

systemd-lock-handler is licensed under the ISC licence. See LICENCE for details.
