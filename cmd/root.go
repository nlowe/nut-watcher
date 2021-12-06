package cmd

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/nlowe/nut-watcher/watcher"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	unit := "nut-driver"

	server := "127.0.0.1"
	var username string

	threshold := 3
	interval := 10 * time.Second

	metricsAddr := ":9100"
	verbosity := "info"

	result := &cobra.Command{
		Use:  "nut-watcher.service",
		Long: "nut-driver on a Pi4 seems to lock up sometimes. Restart the driver if it locks up",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			lvl, err := logrus.ParseLevel(verbosity)
			if err != nil {
				return err
			}

			logrus.SetLevel(lvl)

			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			metrics := &http.Server{
				Addr:         metricsAddr,
				Handler:      promhttp.Handler(),
				WriteTimeout: 10 * time.Second,
				ReadTimeout:  10 * time.Second,
			}

			logrus.WithFields(logrus.Fields{
				"nut-server": server,
				"unit":       unit,
				"metrics":    fmt.Sprintf("http://%s/metrics", metricsAddr),
			}).Infof("Watching Driver, after %d failures it will be restarted", threshold)

			password := os.Getenv("NUT_EXPORTER_PASSWORD")
			if username != "" && password == "" {
				return fmt.Errorf("username set but NUT_EXPORTER_PASSWORD is not set or is empty")
			}

			w := watcher.NewWatcherFor(ctx, unit, server, threshold, interval, username, password)
			go w.Watch()

			go func() {
				if err := metrics.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logrus.WithError(err).Fatal("failure serving metrics")
				}
			}()

			<-ctx.Done()
			logrus.Info("Shutting Down")

			shutdown, timeout := context.WithTimeout(context.Background(), 10*time.Second)
			defer timeout()

			return metrics.Shutdown(shutdown)
		},
	}

	flags := result.PersistentFlags()

	flags.StringVar(&unit, "unit", unit, "systemd unit to restart when monitoring fails")

	flags.StringVarP(&server, "server", "s", server, "nut-server to connect to")
	flags.StringVar(&username, "username", "", "If set, will authenticate with this username to the server. Password must be set in NUT_EXPORTER_PASSWORD environment variable.")

	flags.IntVarP(&threshold, "failure-threshold", "f", threshold, "restart the driver after this many failures in a row")
	flags.DurationVar(&interval, "interval", interval, "how frequently should the driver be checked")

	flag.StringVar(&metricsAddr, "metrics-addr", metricsAddr, "listen on this address for serving metrics")

	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Logging Verbosity [trace, debug, info, warn, error, fatal]")

	return result
}
