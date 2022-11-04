package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/GwanWingYan/fabric-protos-go/peer"
)

type Proposers struct {
	proposers [][]*Proposer
	tokenCh   chan struct{}
}

func NewProposers(inCh []chan *Element, outCh chan *Element) *Proposers {
	// connNum connections for one peer
	// one Proposer for one connection

	var proposers [][]*Proposer

	tokenCh := make(chan struct{}, int(config.Burst))

	// The expect throughput for each client
	expectTPS := float64(config.Rate) / float64(config.ConnNum*config.ClientPerConnNum*config.EndorserGroupNum)

	for i, endorser := range config.Endorsers {
		proposersForNode := make([]*Proposer, config.ConnNum)
		for j := 0; j < config.ConnNum; j++ {
			client, err := CreateEndorserClient(endorser)
			if err != nil {
				logger.Fatalf("Fail to create No. %d connection for endorser %s: %v", j, endorser.Address, err)
			}

			proposersForNode[j] = &Proposer{
				endorserIndex: i,
				connIndex:     j,
				expectTPS:     expectTPS,
				client:        client,
				address:       endorser.Address,
				inCh:          inCh[i],
				outCh:         outCh,
				tokenCh:       tokenCh,
			}
		}
		proposers = append(proposers, proposersForNode)
	}

	return &Proposers{
		proposers: proposers,
		tokenCh:   tokenCh,
	}
}

// StartAsync starts a goroutine as proposer per client per connection per endorser
func (ps *Proposers) StartAsync() {
	logger.Infof("Start sending transactions")

	// Use a token bucket to throttle the sending of proposals
	go func() {
		if config.Rate == 0 {
			for {
				ps.tokenCh <- struct{}{}
			}
		} else {
			interval := 1e9 / float64(config.Rate) * float64(config.EndorserNum) / float64(config.EndorserGroupNum)
			for {
				time.Sleep(time.Duration(interval) * time.Nanosecond)
				ps.tokenCh <- struct{}{}
			}
		}

	}()

	for i := 0; i < config.EndorserNum; i++ {
		for j := 0; j < config.ConnNum; j++ {
			go ps.proposers[i][j].Start()
		}
	}
}

type Proposer struct {
	endorserIndex int
	connIndex     int
	expectTPS     float64
	client        peer.EndorserClient
	address       string
	inCh          chan *Element
	outCh         chan *Element
	tokenCh       chan struct{}
}

func (p *Proposer) getToken() {
	<-p.tokenCh
}

// Start serves as the k-th client of the j-th connection to the endorser specified by channel 'signed'.
// It collects signed proposals and send them to the endorser
func (p *Proposer) Start() {
	for k := 0; k < config.ClientPerConnNum; k++ {
		go p.startClient(k)
	}
}

func (p *Proposer) startClient(clientIndex int) {
	// count := 0
	// time1 := time.Now().UnixNano()
	// var time2 int64

	for {
		select {
		case element := <-p.inCh:
			// Send sign proposal to peer for endorsement

			p.getToken()

			sendTime := time.Now().UnixNano()
			printCh <- fmt.Sprintf("Start: %d %d %s %d %d %d", sendTime, txid2id[element.Txid], element.Txid, p.endorserIndex, p.connIndex, clientIndex)

			// send proposal
			resp, err := p.client.ProcessProposal(context.Background(), element.SignedProp)
			if err != nil || resp.Response.Status < 200 || resp.Response.Status >= 400 {
				if resp == nil {
					logger.Errorf("Error processing proposal: %v, status: unknown, address: %s \n", err, p.address)
				} else {
					logger.Errorf("Error processing proposal: %v, status: %d, message: %s, address: %s \n", err, resp.Response.Status, resp.Response.Message, p.address)
				}
				continue
			}

			// collect for endorsement
			element.lock.Lock()
			element.Responses = append(element.Responses, resp)
			if len(element.Responses) >= config.EndorserNum {
				// Collect enough endorsement for this transaction
				p.outCh <- element
				proposalTime := time.Now().UnixNano()
				printCh <- fmt.Sprintf("Proposal: %d %d %s %d %d %d", proposalTime, txid2id[element.Txid], element.Txid, p.endorserIndex, p.connIndex, clientIndex)
			}
			element.lock.Unlock()

			// time2 = time.Now().UnixNano()
			// count += 1

		case <-doneCh:
			return
		}

		// //TODO: preserve and drop?
		// if count >= 10 {
		// 	realTPS := float64(count*1e9) / float64(time2-time1)
		// 	p.logger.Infof("Proposer(%d,%d,%d) realTPS: %f expectTPS: %f", p.endorserIndex, p.connIndex, clientIndex, realTPS, p.expectTPS)
		// 	// Reset
		// 	time1 = time.Now().UnixNano()
		// 	count = 0
		// }
	}
}
