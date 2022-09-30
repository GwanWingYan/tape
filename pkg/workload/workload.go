package workload

import (
	"fmt"

	"github.com/GwanWingYan/tape/pkg/metrics"
	"github.com/GwanWingYan/tape/pkg/workload/smallbank"
	"github.com/spf13/viper"
)

type Provider interface {
	ForEachClient(i int, session string) smallbank.GeneratorT
	Start()
}

type WorkloadProvider struct {
	Provider
}

func NewWorkloadProvider(provider metrics.Provider, resub chan string) *WorkloadProvider {
	wlp := &WorkloadProvider{}
	workload := viper.GetString("workload")
	switch workload {
	case "smallbank":
		wlp.Provider = smallbank.NewSmallBank(provider, resub)
	case "kv":
		panic("TODO")
	default:
		panic(fmt.Sprintf("Unknown workload type: %s", workload))
	}
	return wlp
}
