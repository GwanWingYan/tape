package infra

import (
	"math/rand"
)

type Signers struct {
	Signers []*Signer
}

func NewSigners(inCh chan *Element, outCh []chan *Element) *Signers {
	signerArray := make([]*Signer, config.SignerNum)
	for i := 0; i < config.SignerNum; i++ {
		signerArray[i] = &Signer{
			inCh:  inCh,
			outCh: outCh,
		}
	}

	return &Signers{Signers: signerArray}
}

func (ss *Signers) StartAsync() {
	for _, signer := range ss.Signers {
		go signer.Start()
	}
}

type Signer struct {
	inCh  chan *Element
	outCh []chan *Element
}

// SignElement signs a transaction with the assembler's identity
func (s *Signer) SignElement(e *Element) (*Element, error) {
	sprop, err := SignProposal(e.Proposal)
	if err != nil {
		return nil, err
	}
	e.SignedProp = sprop

	return e, nil
}

// StartSigner collects an unsigned transactions, sign it, then send it to the 'signed' channel of each endorser
func (s *Signer) Start() {
	//TODO
	endorsersPerGroup := int(len(s.outCh) / config.EndorserGroupNum)
	//TODO
	rand.Seed(666)

	for {
		select {
		case r := <-s.inCh:
			// sign the raw transaction
			t, err := s.SignElement(r)
			if err != nil {
				logger.Fatalf("Fail to sign transaction %s: %v", t.Txid, err)
			}

			// Randomly select a group of endorsers to endorse the transaction
			// TODO: select endorser group based on transactions
			group := rand.Intn(config.EndorserGroupNum)
			start := int(config.EndorserGroupNum * group)
			end := start + endorsersPerGroup
			for i := start; i < end; i++ {
				s.outCh[i] <- t
			}

		case <-doneCh:
			return
		}
	}
}
