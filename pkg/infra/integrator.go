package infra

type Integrators struct {
	integrators []*Integrator
}

func NewIntegrators(inCh chan *Element, outCh chan *Element) (*Integrators, error) {
	itegratorArray := make([]*Integrator, config.IntegratorNum)
	for i := 0; i < config.IntegratorNum; i++ {
		itegratorArray[i] = &Integrator{
			inCh:  inCh,
			outCh: outCh,
		}
	}

	return &Integrators{integrators: itegratorArray}, nil
}

func (its *Integrators) Start() {
	// Start multiple goroutines to extract responses and integrate it into envelope
	for _, it := range its.integrators {
		go it.Start()
	}
}

type Integrator struct {
	inCh  chan *Element
	outCh chan *Element
}

// integrate extracts responses and generates an envelope
func (it *Integrator) Integrate(e *Element) (*Element, error) {
	env, err := CreateSignedTx(e.Proposal, e.Responses)
	if err != nil {
		return nil, err
	}
	e.Envelope = env
	return e, nil
}

// StartIntegrator tries to extract enough response from endorsed transaction and integrate them into an envelope
func (it *Integrator) Start() {
	for {
		select {
		case element := <-it.inCh:
			// Try to generate an envelope
			envelope, err := it.Integrate(element)
			if err != nil {
				// Abort directly because of the different endorsement
				Metric.AddAbort()
				continue
			}
			it.outCh <- envelope
		case <-done:
			return
		}
	}
}
