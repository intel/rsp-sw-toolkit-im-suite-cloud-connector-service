package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/app/config"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/app/routes"
	"github.impcloud.net/Responsive-Retail-Core/cloud-connector-service/pkg/healthcheck"
	"github.impcloud.net/Responsive-Retail-Core/utilities/configuration"
	"github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics"
	reporter "github.impcloud.net/Responsive-Retail-Core/utilities/go-metrics-influxdb"
	golog "log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	// Load config variables
	if err := config.InitConfig(configChangedCallback); err != nil {
		log.WithFields(log.Fields{
			"Method": "config.InitConfig",
			"Action": "Load config",
		}).Fatal(err.Error())
	}

	isHealthyPtr := flag.Bool("isHealthy", false, "a bool, runs a healthcheck")
	flag.Parse()

	if *isHealthyPtr {
		os.Exit(healthcheck.Healthcheck(config.AppConfig.Port))
	}

	// Initialize metrics reporting
	initMetrics()

	setLoggingLevel(config.AppConfig.LoggingLevel)

	log.WithFields(log.Fields{
		"Method": "main",
		"Action": "Start",
	}).Info("Starting application...")

	// Start Webserver
	router := routes.NewRouter()

	// Create a new server and set timeout values.
	server := http.Server{
		Addr:           ":" + config.AppConfig.Port,
		Handler:        router,
		ReadTimeout:    900 * time.Second,
		WriteTimeout:   900 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// We want to report the listener is closed.
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the listener.
	go func() {
		log.Infof("%s running!", config.AppConfig.ServiceName)
		log.Infof("Listener closed : %v", server.ListenAndServe())
		wg.Done()
	}()

	// Listen for an interrupt signal from the OS.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt)

	// Wait for a signal to shutdown.
	<-osSignals

	// Create a context to attempt a graceful 5 second shutdown.
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Attempt the graceful shutdown by closing the listener and
	// completing all inflight requests.
	if err := server.Shutdown(ctx); err != nil {

		log.WithFields(log.Fields{
			"Method":  "main",
			"Action":  "shutdown",
			"Timeout": timeout,
			"Message": err.Error(),
		}).Error("Graceful shutdown did not complete")

		// Looks like we timedout on the graceful shutdown. Kill it hard.
		if err := server.Close(); err != nil {
			log.WithFields(log.Fields{
				"Method":  "main",
				"Action":  "shutdown",
				"Message": err.Error(),
			}).Error("Error killing server")
		}
	}

	// Wait for the listener to report it is closed.
	wg.Wait()
	log.WithField("Method", "main").Info("Completed.")
}

func configChangedCallback(changeDetails []configuration.ChangeDetails) {
	if len(changeDetails) == 1 && changeDetails[0].Name == "loggingLevel" {
		setLoggingLevel(changeDetails[0].Value.(string))
		return
	}

	log.Info("config has changed with some field that requires restarting")
	os.Exit(0)
}

func setLoggingLevel(loggingLevel string) {
	if loggingLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}

	// Not using filtered func (Info, etc ) so that message is always logged
	golog.Printf("Logging level set to %s\n", loggingLevel)
}

func initMetrics() {
	// setup metrics reporting
	if config.AppConfig.TelemetryEndpoint != "" {
		go reporter.InfluxDBWithTags(
			metrics.DefaultRegistry,
			time.Second*10, //cfg.ReportingInterval,
			config.AppConfig.TelemetryEndpoint,
			config.AppConfig.TelemetryDataStoreName,
			"",
			"",
			nil,
		)
	}
}
