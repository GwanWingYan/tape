package infra

import (
	"fmt"
	"time"

	"github.com/GwanWingYan/fabric-protos-go/peer"
)

var (
	timeKeepers TimeKeepers
)

type TimeKeepers struct {
	transactions []*TimeKeeper
}

type TimeKeeper struct {
	ProposedTime  int64
	EndorsedTime  int64
	BroadcastTime int64
	ObservedTime  int64
}

func initTimeKeepers() {
	timeKeepers = TimeKeepers{
		transactions: make([]*TimeKeeper, config.TxNum),
	}
	for i := range timeKeepers.transactions {
		timeKeepers.transactions[i] = &TimeKeeper{}
	}
}

func (tks *TimeKeepers) keepProposedTime(
	txid string,
	endorserIndex int,
	connIndex int,
	clientIndex int,
) {
	proposedTime := time.Now().UnixNano()

	id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %4d %s %d %d %d", "Proposed", proposedTime, id, txid, endorserIndex, connIndex, clientIndex)

	timeKeepers.transactions[id].ProposedTime = proposedTime
}

func (tks *TimeKeepers) keepEndorsedTime(
	txid string,
	endorserIndex int,
	connIndex int,
	clientIndex int,
) {
	endorsedTime := time.Now().UnixNano()

	id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %4d %s %d %d %d", "Endorsed", endorsedTime, id, txid, endorserIndex, connIndex, clientIndex)

	timeKeepers.transactions[id].EndorsedTime = endorsedTime
}

func (tks *TimeKeepers) keepBroadcastTime(
	txid string,
	broadcasterIndex int,
) {
	broadcastTime := time.Now().UnixNano()

	id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %4d %s %d", "Broadcast", broadcastTime, id, txid, broadcasterIndex)

	timeKeepers.transactions[id].BroadcastTime = broadcastTime
}

func (tks *TimeKeepers) keepObservedTime(
	txid string,
	validationCode peer.TxValidationCode,
) {
	observedTime := time.Now().UnixNano()

	id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %4d %s %s", "Observed", observedTime, id, txid, validationCode)

	timeKeepers.transactions[id].ObservedTime = observedTime
}
