package infra

import (
	"fmt"
	"time"

	"github.com/GwanWingYan/fabric-protos-go/peer"
)

type Observer struct {
	client peer.Deliver_DeliverFilteredClient
}

func NewObserver() *Observer {
	deliverer, err := CreateDeliverFilteredClient()
	if err != nil {
		logger.Fatalf("Fail to create DeliverFilteredClient: %v", err)
	}

	seek, err := CreateSignedDeliverNewestEnv()
	if err != nil {
		logger.Fatalf("Fail to create SignedEnvelope: %v", err)
	}

	if err = deliverer.Send(seek); err != nil {
		logger.Fatalf("Fail to send SignedEnvelope: %v", err)
	}

	// drain first response
	if _, err = deliverer.Recv(); err != nil {
		logger.Fatalf("Fail to receive the first response: %v", err)
	}

	return &Observer{
		client: deliverer,
	}
}

// StartAsync starts observing
func (o *Observer) StartAsync() {
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
					close(observerEndCh)
					return
				}
			case <-time.After(20 * time.Second):
				close(observerEndCh)
				return
			}
		}
	}()

	// Receive FilteredBlock
	go func() {
		for {
			r, err := o.client.Recv()
			if err != nil {
				//TODO
				logger.Fatalln(err)
			}
			if r == nil {
				//TODO
				logger.Fatalln("received nil message, but expect a valid block instead. You could look into your peer logs for more info")
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
