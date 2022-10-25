package infra

import (
	"fmt"
	"time"

	"github.com/GwanWingYan/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

type Observer struct {
	client peer.Deliver_DeliverFilteredClient
}

func NewObserver() (*Observer, error) {
	deliverer, err := CreateDeliverFilteredClient()
	if err != nil {
		return nil, err
	}

	seek, err := CreateSignedDeliverNewestEnv()
	if err != nil {
		return nil, err
	}

	if err = deliverer.Send(seek); err != nil {
		return nil, err
	}

	// drain first response
	if _, err = deliverer.Recv(); err != nil {
		return nil, err
	}

	return &Observer{
		client: deliverer,
	}, nil
}

// Start starts observing
func (o *Observer) Start() {
	logger.Infof("Start observer\n")

	deliverCh := make(chan *peer.DeliverResponse_FilteredBlock)

	// Process FilteredBlock
	go func() {
		var validTxNum int32 = 0
		for {
			select {
			case fb := <-deliverCh:
				endTime := time.Now().UnixNano()
				validTxNum = validTxNum + int32(len(fb.FilteredBlock.FilteredTransactions))
				for _, tx := range fb.FilteredBlock.FilteredTransactions {
					txid := tx.GetTxid()
					printCh <- fmt.Sprintf("End: %d %d %s %s", endTime, txid2id[txid], txid, tx.TxValidationCode)
				}
				if validTxNum+Metric.Abort >= int32(config.TxNum) {
					close(finishCh)
					return
				}
			case <-time.After(20 * time.Second):
				close(finishCh)
				return
			}
		}
	}()

	// Receive FilteredBlock
	go func() {
		for {
			r, err := o.client.Recv()
			if err != nil {
				errorCh <- err
			}
			if r == nil {
				errorCh <- errors.Errorf("received nil message, but expect a valid block instead. You could look into your peer logs for more info")
				return
			}

			switch t := r.Type.(type) {
			case *peer.DeliverResponse_FilteredBlock:
				deliverCh <- t
			case *peer.DeliverResponse_Status:
				logger.Infoln("Status:", t.Status)
			default:
				logger.Infoln("Please check the return type manually")
			}
		}
	}()
}
