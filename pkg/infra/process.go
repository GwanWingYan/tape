package infra

import (
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	CH_MAX_CAPACITY     = 100010
	endorsementFilename = "ENDORSEMENT.txt"
)

var (
	txid2id  map[string]int
	config   *Config
	identity *Crypto
	logger   *log.Logger
	printCh  = make(chan string, CH_MAX_CAPACITY)
	printWG  = sync.WaitGroup{}
)

var (
	done     chan struct{}
	finishCh chan struct{}
	errorCh  chan error
)

func Process(c *Config, l *log.Logger) error {
	txid2id = make(map[string]int)
	config = c
	logger = l

	logger.Info("End To End")
	return End2End()

	// if config.End2End {
	// 	fmt.Println("e2e")
	// 	return e2e(config, logger)
	// } else {
	// 	if _, err := os.Stat(endorsementFilename); err == nil {
	// 		fmt.Println("phase2")
	// 		// phase2: broadcast transactions to order
	// 		return breakdown_phase2(config, logger)
	// 	} else {
	// 		fmt.Println("phase1")
	// 		// phase1: send proposals to endorsers {
	// 		return breakdown_phase1(config, logger)
	// 	}

	// }
}

func printBenchmark(logPath string, done <-chan struct{}) {
	printWG.Add(1)
	defer printWG.Done()
	f, err := os.Create(logPath)
	if err != nil {
		logger.Fatalf("Failed to create log file %s: %v\n", logPath, err)
	}
	defer f.Close()

	// Receive following types of elements:
	// 		Start: timestamp txid-index txid  endorser-id, connection-id, client-id
	// 		Proposal: timestamp txid-index txid  endorser-id, connection-id, client-id
	// 		Broadcast: timestamp txid-index txid  broadcaster-id
	// 		End: timestamp txid-index txid [VALID/MVCC]
	// 		Number of all transactions: total-transaction-num
	// 		Number of VALID transactions: valid-transaction-num
	// 		Number of ABORTED transactions: aborted-transaction-num
	// 		Abort rate: abort-rate
	// 		Duration: duration
	// 		TPS: throughput
	for {
		select {
		case res := <-printCh:
			f.WriteString(res + "\n")
		case <-done:
			for len(printCh) > 0 {
				f.WriteString(<-printCh + "\n")
			}
			return
		}
	}
}

// End2End executes end-to-end test on HLF
// An Element (i.e. a transaction) will go through the following channels
// unsignedCh -> signedCh -> endorsedCh -> integratedCh
func End2End() error {
	var err error
	burst := int(config.Burst)

	identity, err = config.LoadCrypto() // loads the client identity
	if err != nil {
		return err
	}

	unsignedCh := make(chan *Element, burst) // all unsigned transactions

	// 'signedCh' are a set of channels
	// one channel for one endorser and stores all signed but not yet endorsed transactions
	signedCh := make([]chan *Element, config.EndorserNum)
	for i := 0; i < config.EndorserNum; i++ {
		signedCh[i] = make(chan *Element, burst)
	}

	// all endorsed but not yet extracted transactions
	endorsedCh := make(chan *Element, burst)

	// all endorsed transactions in envelope format
	integratedCh := make(chan *Element, burst)

	done = make(chan struct{})
	finishCh = make(chan struct{})
	errorCh = make(chan error, burst)

	go printBenchmark(config.LogPath, done)

	initiator, err := NewInitiator(unsignedCh)
	if err != nil {
		return err
	}

	signers, err := NewSigners(unsignedCh, signedCh)
	if err != nil {
		return err
	}

	proposers, err := NewProposers(signedCh, endorsedCh)
	if err != nil {
		return err
	}

	integrators, err := NewIntegrators(endorsedCh, integratedCh)
	if err != nil {
		return err
	}

	broadcasters, err := NewBroadcasters(integratedCh)
	if err != nil {
		return err
	}

	observer, err := NewObserver()
	if err != nil {
		return err
	}

	proposers.Start()

	integrators.Start()

	broadcasters.Start()

	observer.Start()

	// Wait for all raw proposals to be inserted in the 'raw' channel
	initiator.Start()
	time.Sleep(10 * time.Second)

	// Timing starts
	start := time.Now()
	signers.Start()

	// Wait for the end of execution
	for {
		select {
		case err = <-errorCh:
			return err
		case <-finishCh:
			duration := time.Since(start)
			logger.Infof("Completed processing transactions.")
			printCh <- fmt.Sprintf("Number of all transactions: %d", config.TxNum)
			printCh <- fmt.Sprintf("Number of VALID transactions: %d", int32(config.TxNum)-Metric.Abort)
			printCh <- fmt.Sprintf("Number of ABORTED transactions: %d", Metric.Abort)
			printCh <- fmt.Sprintf("Abort rate: %f", float64(Metric.Abort)/float64(config.TxNum)*100)
			printCh <- fmt.Sprintf("Duration: %+v", duration)
			printCh <- fmt.Sprintf("TPS: %f", float64(config.TxNum)*1e9/float64(duration.Nanoseconds()))

			// Closing 'done', a channel which is never sent an element, is a common technique to notify ending in Golang
			// More information: https://go101.org/article/channel-use-cases.html#check-closed-status
			close(done)

			// Wait for printBenchmark to return
			printWG.Wait()

			return nil
		}
	}
}

// func breakdown_phase1(config Config, logger *log.Logger) error {
// 	crypto, err := config.LoadCrypto()
// 	if err != nil {
// 		return err
// 	}
// 	raw := make(chan *Element, burst)
// 	signed := make([]chan *Element, config.EndorserNum)
// 	processed := make(chan *Element, burst)
// 	done := make(chan struct{})
// 	errorCh := make(chan error, burst)
// 	assembler := &Assembler{Signer: crypto, EndorserGroups: config.EndorserGroupNum, Conf: config}
// 	go printBenchmark(config.LogPath, done)

// 	for i := 0; i < config.EndorserNum; i++ {
// 		signed[i] = make(chan *Element, burst)
// 	}

// 	proposers, err := NewProposers(config.ConnNum, config.ClientPerConnNum, config.Endorsers, burst, logger)
// 	if err != nil {
// 		return err
// 	}

// 	StartCreateProposal(config, crypto, raw, errorCh, logger)
// 	time.Sleep(10 * time.Second)
// 	start := time.Now()

// 	for i := 0; i < config.SignerNum; i++ {
// 		go assembler.StartSigner(raw, signed, errorCh, done)
// 	}
// 	proposers.Start(signed, processed, done, config)

// 	// phase1: send proposals to endorsers
// 	var cnt int32 = 0
// 	var buffer [][]byte
// 	var txids []string
// 	for i := 0; i < config.TxNum; i++ {
// 		select {
// 		case err = <-errorCh:
// 			return err
// 		case tx := <-processed:
// 			res, err := assembler.Assemble(tx)
// 			if err != nil {
// 				fmt.Println("error: assemble endorsement to envelop")
// 				return err
// 			}
// 			bytes, err := json.Marshal(res.Envelope)
// 			if err != nil {
// 				fmt.Println("error: marshal envelop")
// 				return err
// 			}
// 			cnt += 1
// 			buffer = append(buffer, bytes)
// 			txids = append(txids, tx.Txid)
// 			if cnt+assembler.Abort >= int32(config.TxNum) {
// 				break
// 			}
// 		}
// 	}
// 	duration := time.Since(start)

// 	logger.Infof("Completed endorsing transactions.")
// 	printCh <- fmt.Sprintf("tx: %d, duration: %+v, tps: %f", config.TxNum, duration, float64(config.TxNum)/duration.Seconds())
// 	printCh <- fmt.Sprintf("abort rate because of the different ledger height: %d %.2f%%", assembler.Abort, float64(assembler.Abort)/float64(config.TxNum)*100)
// 	close(done)
// 	printWG.Wait()
// 	// persistency
// 	mfile, _ := os.Create(endorsementFilename)
// 	defer mfile.Close()
// 	mw := bufio.NewWriter(mfile)
// 	for i := range buffer {
// 		mw.Write(buffer[i])
// 		mw.WriteByte('\n')
// 		mw.WriteString(txids[i])
// 		mw.WriteByte('\n')
// 	}
// 	mw.Flush()
// 	return nil
// }
// func breakdown_phase2(config Config, logger *log.Logger) error {
// 	crypto, err := config.LoadCrypto()
// 	if err != nil {
// 		return err
// 	}
// 	envs := make(chan *Element, burst)
// 	done := make(chan struct{})
// 	finishCh := make(chan struct{})
// 	errorCh := make(chan error, burst)
// 	go printBenchmark(config.LogPath, done)

// 	broadcaster, err := NewBroadcasters(config.OrdererClients, config.Orderer, burst, logger)
// 	if err != nil {
// 		return err
// 	}

// 	observer, err := NewObserver(config.Channel, config.Committer, crypto, logger)
// 	if err != nil {
// 		return err
// 	}

// 	mfile, _ := os.Open(endorsementFilename)
// 	defer mfile.Close()
// 	mscanner := bufio.NewScanner(mfile)
// 	var txids []string
// 	TXs := make([]common.Envelope, config.TxNum)
// 	i := 0
// 	for mscanner.Scan() {
// 		bytes := mscanner.Bytes()
// 		json.Unmarshal(bytes, &TXs[i])
// 		if mscanner.Scan() {
// 			txid := mscanner.Text()
// 			txids = append(txids, txid)
// 		}
// 		i++
// 	}
// 	items := make([]Element, config.TxNum)
// 	for i := 0; i < len(txids); i++ {
// 		var item Element
// 		item.Envelope = &TXs[i]
// 		item.Txid = txids[i]
// 		txid2id[item.Txid] = i
// 		items[i] = item
// 	}

// 	start := time.Now()
// 	go func() {
// 		for i := 0; i < len(items); i++ {
// 			envs <- &items[i]
// 		}
// 	}()
// 	broadcaster.Start(envs, config.Rate, errorCh, done)
// 	var temp0 int32 = 0
// 	go observer.Start(int32(len(txids)), errorCh, finishCh, start, &temp0)
// 	for {
// 		select {
// 		case err = <-errorCh:
// 			return err
// 		case <-finishCh:
// 			duration := time.Since(start)
// 			logger.Infof("Completed processing transactions.")
// 			printCh <- fmt.Sprintf("tx: %d, duration: %+v, tps: %f", config.TxNum, duration, float64(config.TxNum)/duration.Seconds())
// 			close(done)
// 			printWG.Wait()
// 			return nil
// 		}
// 	}
// }
