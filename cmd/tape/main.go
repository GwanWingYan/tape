package main

import (
	"fmt"
	"os"

	"github.com/GwanWingYan/tape/pkg/infra"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	logLevelEnv = "TAPE_LOGLEVEL"
)

var (
	logger  *log.Logger
	config  *infra.Config
	fullCmd string
)

var (
	app = kingpin.New("tape", "A performance test tool for Hyperledger Fabric")

	run     = app.Command("run", "Start the tape program").Default()
	version = app.Command("version", "Show version information")

	configFile = run.Flag("config", "Path to config file").Required().Short('c').String()
)

func newLogger() *log.Logger {
	logger = log.New()
	logger.SetLevel(log.InfoLevel)
	if value, ok := os.LookupEnv(logLevelEnv); ok {
		if level, err := log.ParseLevel(value); err == nil {
			logger.SetLevel(level)
		}
	}
	return logger
}

func loadConfig() *infra.Config {
	config = &infra.Config{}
	err := infra.LoadConfigFile(config, *configFile)
	if err != nil {
		log.Fatalf("load config error: %v\n", err)
	}
	return config
}

func checkArgs() {
	if config.Rate < 0 {
		logger.Errorf("tape: error: rate %f is not a zero (unlimited) or positive number\n", config.Rate)
		os.Exit(1)
	}

	if config.Burst < 1 {
		logger.Errorf("tape: error: burst %d is not greater than 1\n", config.Burst)
		os.Exit(1)
	}

	if config.Rate > config.Burst {
		fmt.Printf("rate %f is bigger than burst %f, so set rate to burst\n", config.Rate, config.Burst)
		config.Rate = config.Burst
	}
}

func main() {
	var err error

	fullCmd = kingpin.MustParse(app.Parse(os.Args[1:]))

	logger = newLogger()

	config = loadConfig()

	switch fullCmd {
	case run.FullCommand():
		checkArgs()
		err = infra.Process(config, logger)
	case version.FullCommand():
		fmt.Printf(infra.GetVersionInfo())
	default:
		err = errors.Errorf("invalid command: %s", fullCmd)
	}

	if err != nil {
		logger.Errorln(err)
		os.Exit(1)
	}
	os.Exit(0)
}
