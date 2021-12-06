package main

import (
	"os"

	"github.com/nlowe/nut-watcher/cmd"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func main() {
	logrus.SetFormatter(&prefixed.TextFormatter{
		ForceFormatting: true,
		ForceColors:     true,
		FullTimestamp:   true,
	})

	if err := cmd.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
