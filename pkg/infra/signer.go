package infra

import (
	"math/rand"
)

type Signers struct {
	Signers []*Signer
}

func NewSigners(inCh chan *Element, outCh []chan *Element) *Signers {
	signerList := make([]*Signer, config.SignerNum)
	for i := 0; i < config.SignerNum; i++ {
		signerList[i] = &Signer{
			inCh:  inCh,
			outCh: outCh,
		}
	}

	return &Signers{Signers: signerList}
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

// Start collects an unsigned transactions from the 'raw' channel,
// sign it, then send it to the 'signed' channel of each endorser
func (s *Signer) Start() {
	endorsersPerGroup := int(len(s.outCh) / config.EndorserGroupNum)

	for {
		select {
		case e := <-s.inCh:
			// sign the raw transaction
			err := s.SignElement(e)
			if err != nil {
				logger.Fatalf("Fail to sign transaction %s: %v", e.Txid, err)
			}

			// Randomly select a groupIndex of endorsers to endorse the transaction
			groupIndex := rand.Intn(config.EndorserGroupNum)
			endorserStartIndex := int(config.EndorserGroupNum * groupIndex)
			endorserEndIndex := endorserStartIndex + endorsersPerGroup
			for i := endorserStartIndex; i < endorserEndIndex; i++ {
				s.outCh[i] <- e
			}

		case <-doneCh:
			return
		}
	}
}

// SignElement signs a transaction with the assembler's identity
func (s *Signer) SignElement(e *Element) error {
	signedProposal, err := SignProposal(e.Proposal)
	if err != nil {
		return err
	}
	e.SignedProposal = signedProposal

	return nil
}
