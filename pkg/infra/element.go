package infra

import (
	"sync"

	"github.com/GwanWingYan/fabric-protos-go/common"
	"github.com/GwanWingYan/fabric-protos-go/peer"
)

// Element contains the data for the whole lifecycle of a transaction
type Element struct {
	Proposal   *peer.Proposal
	SignedProp *peer.SignedProposal
	Responses  []*peer.ProposalResponse
	lock       sync.Mutex
	Envelope   *common.Envelope
	Txid       string
}
