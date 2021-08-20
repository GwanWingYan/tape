package main

import (
	"fmt"
	"os"

	"tape/pkg/infra"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	loglevel = "TAPE_LOGLEVEL"
)

var (
	app = kingpin.New("tape", "A performance test tool for Hyperledger Fabric")

	run             = app.Command("run", "Start the tape program").Default()
	con             = run.Flag("config", "Path to config file").Required().Short('c').String()
	num             = run.Flag("number", "Number of tx for shot").Required().Short('n').Int()
	rate            = run.Flag("rate", "[Optional] Creates tx rate, default 0 as unlimited").Default("0").Float64()
	burst           = run.Flag("burst", "[Optional] Burst size for Tape, should bigger than rate").Default("1000").Int()
	e2e             = run.Flag("e2e", "end2end").Required().Bool()
	hot_rate        = run.Flag("hrate", "hot rate").Default("1").Float64()
	txtype          = run.Flag("txtype", "transaction type [create, create_random, transfer, null]").Default("null").String()
	contention_rate = run.Flag("crate", "contention rate").Default("1").Float64()
	nd_rate         = run.Flag("ndrate", "nondeterminism rate").Default("0").Float64()
	num_of_conn     = run.Flag("num_of_conn", "num of conn").Default("16").Int()
	client_per_conn = run.Flag("client_per_conn", "client per conn").Default("16").Int()
	groups          = run.Flag("groups", "endorser groups").Default("10").Int()
	orderer_client  = run.Flag("orderer_client", "orderer client").Default("20").Int()
	version         = app.Command("version", "Show version information")
)

func main() {
	var err error

	logger := log.New()
	logger.SetLevel(log.WarnLevel)
	if customerLevel, customerSet := os.LookupEnv(loglevel); customerSet {
		if lvl, err := log.ParseLevel(customerLevel); err == nil {
			logger.SetLevel(lvl)
		}
	}

	fullCmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch fullCmd {
	case version.FullCommand():
		fmt.Printf(infra.GetVersionInfo())
	case run.FullCommand():
		checkArgs(rate, burst, logger)
		err = infra.Process(*con, *num, *burst, *rate, *e2e, *hot_rate, *contention_rate, *nd_rate, *txtype, *num_of_conn, *client_per_conn, *groups, *orderer_client, logger)
	default:
		err = errors.Errorf("invalid command: %s", fullCmd)
	}

	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func checkArgs(rate *float64, burst *int, logger *log.Logger) {
	if *rate < 0 {
		os.Stderr.WriteString("tape: error: rate must be zero (unlimited) or positive number\n")
		os.Exit(1)
	}
	if *burst < 1 {
		os.Stderr.WriteString("tape: error: burst at least 1\n")
		os.Exit(1)
	}

	if int64(*rate) > int64(*burst) {
		fmt.Printf("As rate %d is bigger than burst %d, real rate is burst\n", int64(*rate), int64(*burst))
	}

	logger.Infof("Will use rate %f as send rate\n", *rate)
	logger.Infof("Will use %d as burst\n", burst)
}
