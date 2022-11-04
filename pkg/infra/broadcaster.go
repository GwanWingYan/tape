package infra

import (
	"fmt"
	"io"
	"time"

	"github.com/GwanWingYan/fabric-protos-go/common"
	"github.com/GwanWingYan/fabric-protos-go/orderer"
)

type Broadcasters struct {
	broadcasters []*Broadcaster
	tokenCh      chan struct{}
}

func NewBroadcasters(inCh <-chan *Element) *Broadcasters {
	bs := &Broadcasters{
		broadcasters: make([]*Broadcaster, config.BroadcasterNum),
		tokenCh:      make(chan struct{}, int(config.Burst)),
	}

	// The expect throughput for each broadcaster
	expectTPS := float64(config.Rate) / float64(config.BroadcasterNum)

	for i := 0; i < config.BroadcasterNum; i++ {
		client, err := CreateBroadcastClient(config.Orderer)
		if err != nil {
			logger.Fatalf("Fail to create connection for the No. %d broadcaster: %v", i, err)
		}

		bs.broadcasters[i] = &Broadcaster{
			client:           client,
			broadcasterIndex: i,
			expectTPS:        expectTPS,
			inCh:             inCh,
			tokenCh:          bs.tokenCh,
		}
	}

	return bs
}

// StartAsync starts a goroutine for every broadcaster
func (bs *Broadcasters) StartAsync() {
	// Use a token bucket to throttle the sending of envelopes
	go func() {
		if config.Rate == 0 {
			for {
				bs.tokenCh <- struct{}{}
			}
		} else {
			interval := 1e9 / config.Rate
			for {
				bs.tokenCh <- struct{}{}
				time.Sleep(time.Duration(interval) * time.Nanosecond)
			}
		}
	}()

	// Start multiple goroutines to send envelopes
	for _, b := range bs.broadcasters {
		//TODO?
		go b.StartDraining()
		go b.Start()
	}
}

type Broadcaster struct {
	client           orderer.AtomicBroadcast_BroadcastClient
	broadcasterIndex int
	expectTPS        float64
	inCh             <-chan *Element
	tokenCh          chan struct{}
}

func (b *Broadcaster) getToken() {
	<-b.tokenCh
}

// Start collects and send envelopes to the orderer
func (b *Broadcaster) Start() {
	logger.Infof("Start broadcasting\n")

	// count := 0
	// time1 := time.Now().UnixNano()
	// var time2 int64

	for {
		select {
		case element := <-b.inCh:
			// todo
			b.getToken()
			broadcastTime := time.Now().UnixNano()
			printCh <- fmt.Sprintf("Broadcast: %d %d %s %d", broadcastTime, txid2id[element.Txid], element.Txid, b.broadcasterIndex)
			err := b.client.Send(element.Envelope)
			if err != nil {
				logger.Fatalln(err)
			}

			// time2 = time.Now().UnixNano()
			// count += 1

		case <-doneCh:
			return
		}

		// //TODO: preserve and drop?
		// if count >= 20 {
		// 	realTPS := count * 1e9 / int(time2-time1)
		// 	logger.Infof("Broadcaster(%d) realTPS: %f expectTPS: %f", realTPS, b.expectTPS)
		// 	// reset
		// 	count = 0
		// 	time1 = time.Now().UnixNano()
		// }
	}
}

func (b *Broadcaster) StartDraining() {
	for {
		res, err := b.client.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			logger.Errorf("recieve broadcast error: %+v, status: %+v\n", err, res)
			return
		}

		if res.Status != common.Status_SUCCESS {
			logger.Fatalf("Receive error status %s", res.Status)
		}
	}
}
