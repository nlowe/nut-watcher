# nut-watcher

For some reason, [`nut-driver`](https://networkupstools.org/) on the
Raspberry Pi likes to break. Every few days it stalls out trying to
talk to my UPS. `nut-watcher` is a simple Go service that polls the
driver and if it fails to poll 3 times in a row, it instructs
systemd to restart `nut-driver.service`.

## Building

To build for a raspberry pi:

```
GOOS=linux GOARCH=arm64 go build -o nut-watcher -v main.go
```

An example systemd unit:

```ini
[Unit]
Description=Network UPS Tools Watchdog
After=network.target nut-server.service nut-driver.service nut-monitor.service

[Service]
Environment=NUT_EXPORTER_PASSWORD=***
ExecStart=/usr/local/bin/nut-watcher --username=***
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```
