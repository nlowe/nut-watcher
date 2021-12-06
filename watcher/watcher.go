package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	nut "github.com/robbiet480/go.nut"
	"github.com/sirupsen/logrus"
)

const unitRestartModeReplace = "replace"

var (
	driverRestartsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nut_watcher_driver_restarts_total",
	})

	driverRestartErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nut_watcher_driver_restart_errors_total",
	})
)

type Watcher struct {
	unit string

	server   string
	username string
	password string

	failCount int
	threshold int

	ctx      context.Context
	interval time.Duration

	log logrus.FieldLogger
}

func NewWatcherFor(ctx context.Context, unit, server string, threshold int, interval time.Duration, username, password string) *Watcher {
	return &Watcher{
		unit: unit,

		server:   server,
		username: username,
		password: password,

		threshold: threshold,

		ctx:      ctx,
		interval: interval,

		log: logrus.WithField("prefix", "watcher"),
	}
}

func (w *Watcher) Watch() {
	timer := time.NewTicker(w.interval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if err := w.check(); err != nil {
				w.failCount++
				w.log.WithError(err).WithFields(logrus.Fields{
					"count":     w.failCount,
					"threshold": w.threshold,
				}).Warn("UPS Seems Dead")

				if w.failCount > w.threshold {
					w.log.WithError(w.restartDriver()).Warn("Asked driver to restart")
					w.failCount = 0
				}
			} else {
				w.failCount = 0
			}
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Watcher) restartDriver() error {
	w.log.Warnf("Restarting %s", w.unit)
	ctx, cancel := context.WithTimeout(w.ctx, w.interval)
	defer cancel()

	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		driverRestartErrorsTotal.Inc()
		return fmt.Errorf("failed to connect to dbus: %w", err)
	}

	defer conn.Close()
	driverRestartsTotal.Inc()

	_, err = conn.RestartUnitContext(ctx, w.unit, unitRestartModeReplace, nil)
	if err != nil {
		driverRestartErrorsTotal.Inc()
		return fmt.Errorf("failed to schedule driver restart: %w", err)
	}

	return nil
}

func (w *Watcher) check() error {
	w.log.Debugf("Checking if UPS driver is alive")

	log := w.log.WithField("server", w.server)

	log.Trace("Connecting to driver")
	c, err := nut.Connect(w.server)
	if err != nil {
		return fmt.Errorf("failed to create NUT client: %w", err)
	}

	if w.username != "" && w.password != "" {
		log.Tracef("Authenticating with Server as %s", w.username)
		_, err = c.Authenticate(w.username, w.password)
		if err != nil {
			return fmt.Errorf("failed to authenticate to NUT server: %w", err)
		}
	}

	log.Trace("Listing Devices")
	clients, err := c.GetUPSList()
	if err != nil {
		return fmt.Errorf("failed to list UPS devices: %w", err)
	}

	if len(clients) == 0 {
		return fmt.Errorf("driver returned no clients")
	}

	log.Trace("UPS is Healthy")
	return nil
}
