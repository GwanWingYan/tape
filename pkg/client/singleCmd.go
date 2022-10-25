package client

import (
	"fmt"

	"github.com/GwanWingYan/tape/pkg/workload"
	log "github.com/sirupsen/logrus"

	"github.com/GwanWingYan/tape/pkg/operations"
	"github.com/spf13/viper"
)

func RunSingleCmd(config Config, txn []string) {
	metricsSystem = operations.NewSystem(operations.Options{
		ListenAddress: viper.GetString("metricsAddr"),
		Provider:      "disabled",
	})
	metric := NewMetrics(metricsSystem.Provider)
	crypto, err := config.LoadCrypto()
	if err != nil {
		panic(fmt.Sprintf("load crypto failed: %v", err))
	}

	e2eCh := make(chan *Tracker, CH_MAX_CAPACITY)

	resub := make(chan string, 10000)
	viper.SetDefault("clientsNumber", len(config.Endorsers)*viper.GetInt("clientsPerEndorser"))

	workload := workload.NewWorkloadProvider(metricsSystem.Provider, resub)
	cm := NewClientManager(
		e2eCh,
		config.Endorsers,
		config.Orderer,
		crypto,
		viper.GetInt("clientsPerEndorser"),
		workload.Provider,
		metric,
		resub,
	)
	node := viper.GetInt("queryNode")
	res := cm.Execute(node, txn)
	log.Println("txn result:", res)
}
