package main

import (
  "context"
  "fmt"
  "log"
  "plugin"

  "github.com/coreos/go-systemd/v22/daemon"
  systemd "github.com/coreos/go-systemd/v22/dbus"
  dbus "github.com/godbus/dbus/v5"

  "github.com/ilyakaznacheev/cleanenv"
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

type Hardcode = func() (
  func(), 
  func(*dbus.Conn, *dbus.Signal) bool, 
  func(), 
  func(),
)

func ListenFor(
  target string, 
  toggle bool, 
  start bool, 
  system bool, 
  hardcode Hardcode, 
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
    for {
      v := <-c

      // Hardcode: verify user session for (un)lock.target
      // Hardcode: verify want==got for sleep.target
      if verify(conn, v) {
	break
      }
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

type Config struct {
  Targets [] struct {
    Target string `yaml:"target"`
    Dlib string `yaml:"dlib"`
    Toggle bool `yaml:"toggle"`
    Start bool `yaml:"start"`
    System bool `yaml:"system"`
    MatchOptions map[string]string `yaml:"dbus"`
  } `yaml:"targets"`
}

func main() {
  log.SetFlags(log.Lshortfile)

  var cfg Config
  err := cleanenv.ReadConfig("config.yml", &cfg)
  if err != nil {
    log.Fatalln("Failed to read config:", err)
  }

  for _, target := range cfg.Targets {
    // Convert the matchOptions map to []MatchOption
    matchOptions := make([]dbus.MatchOption, 0, len(target.MatchOptions))
    for key, value := range target.MatchOptions {
      matchOptions = append(matchOptions, dbus.WithMatchOption(key, value))
    }
    
    // Load dynamic library
    dlib, err := plugin.Open(target.Dlib)
    if err != nil {
      log.Fatalf("Failed to load dynamic library %v for target %v: %v", target.Dlib, target.Target, err)
    }

    symbol, err := dlib.Lookup("Hardcode")
    if err != nil {
      log.Fatalf("Failed to locate symbol 'Hardcode' in dynamic library %v for target %v: %v", target.Dlib, target.Target, err)
    }

    hardcode, ok := symbol.(Hardcode)
    if !ok {
      log.Fatalf("Unexpected signature for symbol 'Hardcode' in dynamic library %v for target %v", target.Dlib, target.Target)
    }

    // Dispatch Listener
    ListenFor(
      target.Target,
      target.Toggle,
      target.Start,
      target.System,
      hardcode,
      matchOptions...,
    )
  }

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
