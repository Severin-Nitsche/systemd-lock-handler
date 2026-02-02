package main

import (
  "context"
  "fmt"
  "log"
  "os/user"
  "os"

  "github.com/coreos/go-systemd/v22/daemon"
  systemd "github.com/coreos/go-systemd/v22/dbus"
  "github.com/coreos/go-systemd/v22/login1"
  dbus "github.com/godbus/dbus/v5"
)

// Starts/Stops a systemd unit and blocks until the job is completed.
func HandleSystemdUnit(unitName string, start bool, system bool) error {
  var conn *systemd.Conn
  var err error

  if system {
    conn, err = systemd.NewSystemConnectionContext(context.Background())
  } else {
    conn, err = systemd.NewUserConnectionContext(context.Background())
  }

  if err != nil {
    return fmt.Errorf("failed to connect to systemd session (system: %v): %v", system, err)
  }

  ch := make(chan string, 1)

  if start {
    _, err = conn.StartUnitContext(context.Background(), unitName, "replace", ch)
  } else {
    _, err = conn.StopUnitContext(context.Background(), unitName, "replace", ch)
  }
  if err != nil {
    return fmt.Errorf("failed to handle unit (start: %v): %v", start, err)
  }

  result := <-ch
  if result == "done" {
    log.Printf("Handled systemd unit: %v (system: %v, start: %v)", unitName, system, start)
  } else {
    return fmt.Errorf("failed to handle unit %v (system: %v, start: %v): %v", unitName, system, start, result)
  }

  return nil
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

func hardcodeLock(user *user.User) (
  func(), 
  func(*dbus.Conn, *dbus.Signal) bool, 
  func(), 
  func(),
) {
  verifySession := func (conn *dbus.Conn, v *dbus.Signal) bool {
    // Get the (un)locked session
    obj := conn.Object("org.freedesktop.login1", v.Path)

    name, err := obj.GetProperty("org.freedesktop.login1.Session.Name")
    if err != nil {
      log.Println("WARNING: Could not obtain details for session:", err)
      return false
    }

    if name.Value() != user.Username {
      return false
    }
    log.Println("Session signal for current user:", v.Name)

    return true
  }

  noop := func () {}

  return noop, verifySession, noop, noop
}


func ListenFor(
  target string, 
  toggle bool, 
  start bool, 
  system bool, 
  hardcode func() (
    func(), 
    func(*dbus.Conn, *dbus.Signal) bool, 
    func(), 
    func(),
  ), 
  options ...dbus.MatchOption,
) {
  conn, err := dbus.ConnectSystemBus()
  if err != nil {
    log.Fatalln("Could not connect to the system D-Bus", err)
  }

  err = conn.AddMatchSignal(options...)
  if err != nil {
    log.Fatalf("Failed to listen for %v signals: %v", target, err)
  }

  init, verify, before, after := hardcode();

  c := make(chan *dbus.Signal, 10)

  // Hardcode: Initialize logind for sleep.target
  init()

  waitFor := func (start bool) {
    v := <-c

    // Hardcode: verify user session for (un)lock.target
    // Hardcode: verify want==got for sleep.target
    if !verify(conn, v) {
      return
    }

    err = HandleSystemdUnit(target, start, system)
    if err != nil {
      log.Println("Error handling target:", target, err)
    }
  }

  go func() {
    for {
      // Hardcode: Inhibit Sleep for sleep.target
      before()

      waitFor(start)

      if !toggle {
	continue
      }

      // Hardcode: Uninhibit Sleep for sleep.target
      after()

      waitFor(!start)

    }
  }()

  conn.Signal(c)
  log.Printf("Listening for %v signals", target);
}

func main() {
  log.SetFlags(log.Lshortfile)

  user, err := user.Current()
  if err != nil {
    log.Fatalln("Failed to get username:", err)
  }
  log.Println("Running for user:", user.Username)

  lockCode := func () (
    func(), 
    func(*dbus.Conn, *dbus.Signal) bool, 
    func(), 
    func(),
  ){
    return hardcodeLock(user)
  }

  // Listen for sleep
  ListenFor(
    "sleep.target", // target
    true, // toggle
    true, // start
    false, // system
    hardcodeSleep,
    dbus.WithMatchObjectPath("/org/freedesktop/login1"),
    dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
    dbus.WithMatchMember("PrepareForSleep"),
  )

  // Listen for Lock
  ListenFor(
    "lock.target",
    false, // toggle
    true, // start
    false, // system
    lockCode,
    dbus.WithMatchInterface("org.freedesktop.login1.Session"),
    dbus.WithMatchSender("org.freedesktop.login1"),
    dbus.WithMatchMember("Lock"),
  )

  ListenFor(
    "unlock.target",
    false, // toggle
    true, // start
    false, // system
    lockCode,
    dbus.WithMatchInterface("org.freedesktop.login1.Session"),
    dbus.WithMatchSender("org.freedesktop.login1"),
    dbus.WithMatchMember("Unlock"),
  )
  
  log.Println("Initialization complete.")

  sent, err := daemon.SdNotify(true, daemon.SdNotifyReady)
  if !sent {
    log.Println("Couldn't call sd_notify. Not running via systemd?")
  }
  if err != nil {
    log.Println("Call to sd_notify failed:", err)
  }

  select {}
}
