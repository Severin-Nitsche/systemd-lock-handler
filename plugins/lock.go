package main

import (
  "log"
  "os/user"
  
  dbus "github.com/godbus/dbus/v5"
)

func Hardcode() (
  func(), 
  func(*dbus.Conn, *dbus.Signal) bool, 
  func(), 
  func(),
) {
  user, err := user.Current()
  if err != nil {
    log.Fatalln("Failed to get username:", err)
  }
  log.Println("Running for user:", user.Username)

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
