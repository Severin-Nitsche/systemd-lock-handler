{
  lib,
  buildGoModule,
}:

let
  fs = lib.fileset;
  sourceFiles = fs.unions [
    ./main.go
    ./go.mod
    ./go.sum
    ./dbus-systemd-dispatcher.service
  ];
in buildGoModule {
  pname = "dbus-systemd-dispatcher";
  version = "3.0.0";

  src = fs.toSource {
    root = ./.;
    fileset = sourceFiles;
  };

  vendorHash = "sha256-8FjviQT/d+H/ClbK8a2eYzvgFkA+U/auZ0jE1FFOgHI=";

  postInstall = ''
    install -Dm644 dbus-systemd-dispatcher.service $out/lib/systemd/user/dbus-systemd-dispatcher.service
    install -Dm644 dbus-systemd-dispatcher.service $out/lib/systemd/system/dbus-systemd-dispatcher.service
  '';

  preInstall = ''
    substituteInPlace dbus-systemd-dispatcher.service \
      --replace /usr/lib $out/bin/
  '';

  meta = {
    description = "Translates D-Bus events into systemd targets";
    license = lib.licenses.isc;
    platforms = lib.platforms.linux;
  };
}
