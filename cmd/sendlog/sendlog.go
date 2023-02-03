package main

import (
	"flag"
	"fmt"
	"log/syslog"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/nxadm/tail"
)

func mustSendLog(filename string, writer *syslog.Writer) {
	t, err := tail.TailFile(filename, tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		panic(err)
	}
	for line := range t.Lines {
		fmt.Fprintf(writer, line.Text)
	}
}

func main() {
	defaultLogURL := os.Getenv("LAZ_SYSLOGURL")
	if defaultLogURL == "" {
		defaultLogURL = "udp://localhost:514"
	}
	logURL := flag.String("syslogurl", defaultLogURL, "syslog url to send the logs to: (tcp|udp)://host:port, default to udp://localhost:514")
	logFile := flag.String("log", os.Getenv("LAZ_LOGFILE"), "log file to analyze")
	flag.Parse()
	sigint := make(chan os.Signal)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigint
		os.Exit(1)
	}()
	u, err := url.Parse(*logURL)
	if err != nil {
		panic(err)
	}
	if u.Scheme != "tcp" && u.Scheme != "udp" {
		panic(fmt.Errorf("syslog URL scheme must be either tcp or udp, you put %s", u.Scheme))
	}
	if u.Hostname() == "" {
		panic(fmt.Errorf("syslog URL host can't be empty"))
	}
	if u.Port() == "" {
		panic(fmt.Errorf("syslog URL port can't be empty"))
	}
	logger, err := syslog.Dial(u.Scheme, u.Host, syslog.LOG_NOTICE, "log-analyzer")
	if err != nil {
		panic(err)
	}
	mustSendLog(*logFile, logger)
}
