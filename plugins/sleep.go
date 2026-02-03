package main

import(
  "os"
  "log"

  "github.com/coreos/go-systemd/v22/login1"
  dbus "github.com/godbus/dbus/v5"
)


func Hardcode() (
  func(), 
  func(*dbus.Conn, *dbus.Signal) bool, 
  func(), 
  func(),
) {
  var logind *login1.Conn
  var lock *os.File
  want := true

  initLogind := func () {
    var err error
    logind, err = login1.New()
    if err != nil {
      log.Fatalln("Failed to connect to logind")
    }
  }

  inhibitSleep := func () {
    var err error
    lock, err = logind.Inhibit(
      "sleep",
      "systemd-lock-handler",
      "Start pre-sleep target",
      "delay",
    )
    if err != nil {
      log.Fatalln("Failed to grab sleep inhibitor lock", err)
    }
    log.Println("Got lock on sleep inhibitor")
  }

  uninhibitSleep := func () {
    err := lock.Close()
    if err != nil {
      log.Fatalln("Error releasing inhibitor lock:", err)
    }
  }

  verify := func (conn *dbus.Conn, v *dbus.Signal) bool {
    if len(v.Body) == 0 {
      log.Fatalln("empty signal arguments:", v)
    }

    got, ok := v.Body[0].(bool)
    if !ok {
      log.Fatalln("active argument not a bool:", v.Body[0])
    }

    if want != got {
      log.Fatalf("expected PrepareForSleep(%v), got %v", want, got)
    }

    want = !want

    return true
  }

  return initLogind, verify, inhibitSleep, uninhibitSleep
}

func hardcodeSleep() (
  func(), 
  func(*dbus.Conn, *dbus.Signal) bool, 
  func(), 
  func(),
) {
  var logind *login1.Conn
  var lock *os.File
  want := true

  initLogind := func () {
    var err error
    logind, err = login1.New()
    if err != nil {
      log.Fatalln("Failed to connect to logind")
    }
  }

  inhibitSleep := func () {
    var err error
    lock, err = logind.Inhibit(
      "sleep",
      "systemd-lock-handler",
      "Start pre-sleep target",
      "delay",
    )
    if err != nil {
      log.Fatalln("Failed to grab sleep inhibitor lock", err)
    }
    log.Println("Got lock on sleep inhibitor")
  }

  uninhibitSleep := func () {
    err := lock.Close()
    if err != nil {
      log.Fatalln("Error releasing inhibitor lock:", err)
    }
  }

  verify := func (conn *dbus.Conn, v *dbus.Signal) bool {
    if len(v.Body) == 0 {
      log.Fatalln("empty signal arguments:", v)
    }

    got, ok := v.Body[0].(bool)
    if !ok {
      log.Fatalln("active argument not a bool:", v.Body[0])
    }

    if want != got {
      log.Fatalf("expected PrepareForSleep(%v), got %v", want, got)
    }

    want = !want

    return true
  }

  return initLogind, verify, inhibitSleep, uninhibitSleep
}
