package infra

import (
	"strconv"

	"github.com/GwanWingYan/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

type Initiator struct {
	proposals []*peer.Proposal
	txids     []string
	outCh     chan *Element
}

func NewInitiator(outCh chan *Element) (*Initiator, error) {
	it := &Initiator{
		proposals: make([]*peer.Proposal, config.TxNum),
		txids:     make([]string, config.TxNum),
		outCh:     outCh,
	}

	wg := NewWorkloadGenerator()
	chaincodeCtorJSONs := wg.GenerateWorkload()
	session := getName(20)

	// Create proposal and id for all generated transactions
	for i := 0; i < config.TxNum; i++ {
		chaincodeCtorJSON := chaincodeCtorJSONs[i]

		tempTxID := ""
		if !config.CheckTxID {
			// A customized transaction id
			//TODO: is this scheme accpetable?
			tempTxID = strconv.Itoa(i) + "_+=+_" + session + "_+=+_" + getName(20)
		}

		prop, txid, err := CreateProposal(
			tempTxID,
			config.Channel,
			config.Chaincode,
			config.Version,
			chaincodeCtorJSON,
		)
		if err != nil {
			errorCh <- errors.Wrapf(err, "Error creating proposal %s", txid)
			return nil, err
		}

		txid2id[txid] = i
		it.proposals[i] = prop
		it.txids[i] = txid
	}

	return it, nil
}

func (it *Initiator) Start() {
	go func() {
		// send all unsigned transactions (raw transactions) to the channel 'raw'
		// waiting for subsequent processing
		for i := 0; i < len(it.proposals); i++ {
			it.outCh <- &Element{Proposal: it.proposals[i], Txid: it.txids[i]}
		}
	}()
}
