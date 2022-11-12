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
	txid2id map[string]int
	config  *Config
	logger  *log.Logger
)

var (
	logCh         chan string
	reportCh      chan string
	unsignedCh    chan *Element
	signedChs     []chan *Element
	endorsedCh    chan *Element
	integratedCh  chan *Element
	observerEndCh chan struct{}
	doneCh        chan struct{}
)

// isBreakdownPhase1 returns true if this round is phase 1,
// false if this round is phase 2
func isBreakdownPhase1() bool {
	_, err := os.Stat(endorsementFilename)
	return err != nil
}

func Process(c *Config, l *log.Logger) {
	txid2id = make(map[string]int)
	config = c
	logger = l

	if config.End2End {
		logger.Info("Test Mode: End To End")
		End2End()
	} else {
		if isBreakdownPhase1() {
			logger.Info("Test Mode: Breakdown Phase 1")
			BreakdownPhase1(config, logger)
		} else {
			logger.Info("Test Mode: Breakdown Phase 2")
			BreakdownPhase2(config, logger)
		}
	}
}

// WriteLogToFile receives and write the following types of log to file:
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
func WriteLogToFile(printWG *sync.WaitGroup) {
	printWG.Add(1)
	defer printWG.Done()

	logFile, err := os.Create(config.LogPath)
	if err != nil {
		logger.Fatalf("Failed to create log file %s: %v\n", config.LogPath, err)
	}
	defer logFile.Close()

	reportFile, err := os.Create(config.ReportPath)
	if err != nil {
		logger.Fatalf("Failed to create report file %s: %v\n", config.ReportPath, err)
	}
	defer reportFile.Close()

	for {
		select {
		case s := <-logCh:
			logFile.WriteString(s + "\n")
		case s := <-reportCh:
			reportFile.WriteString(s + "\n")
		case <-doneCh:
			for len(logCh) > 0 {
				logFile.WriteString(<-logCh + "\n")
			}
			return
		}
	}
}

func NewLogChannel() chan string {
	logCh := make(chan string, CH_MAX_CAPACITY)
	return logCh
}

func NewReportChannel() chan string {
	reportCh := make(chan string, CH_MAX_CAPACITY)
	return reportCh
}

func NewUnsignedChannel() chan *Element {
	// unsignedCh stores all unsigned transactions
	// Sender: initiator
	// Receiver: signers
	unsignedCh := make(chan *Element, config.Burst)
	return unsignedCh
}

func NewSignedChannel() []chan *Element {
	// signedChs are a set of channels, each of which is for one endorser
	// and stores all signed but not yet endorsed transactions
	// Sender: signers
	// Receiver: proposers
	signedChs := make([]chan *Element, config.EndorserNum)
	for i := 0; i < config.EndorserNum; i++ {
		signedChs[i] = make(chan *Element, config.Burst)
	}
	return signedChs
}

func NewEndorsedChannel() chan *Element {
	// endorsedCh stores all endorsed but not yet extracted transactions
	// Sender: proposers
	// Receiver: integrators
	endorsedCh := make(chan *Element, config.Burst)
	return endorsedCh
}

func NewIntegratedChannel() chan *Element {
	// integratedCh stores all endorsed envelope-format transactions
	// Sender: integrators
	// Receiver: broadcasters
	integratedCh := make(chan *Element, config.Burst)
	return integratedCh
}

func NewObserverEndChannel() chan struct{} {
	observerEndCh := make(chan struct{})
	return observerEndCh
}

func initDoneChannel() chan struct{} {
	doneCh := make(chan struct{})
	return doneCh
}

func initChannels() {
	logCh = NewLogChannel()
	reportCh = NewReportChannel()
	unsignedCh = NewUnsignedChannel()
	signedChs = NewSignedChannel()
	endorsedCh = NewEndorsedChannel()
	integratedCh = NewIntegratedChannel()
	observerEndCh = NewObserverEndChannel()
	doneCh = initDoneChannel()
}

func WaitObserverEnd(startTime time.Time, printWG *sync.WaitGroup) {
	select {
	case <-observerEndCh:
		duration := time.Since(startTime)
		logger.Infof("Finish processing transactions")

		reportCh <- fmt.Sprintf("Number of ALL Transactions: %d", config.TxNum)
		reportCh <- fmt.Sprintf("Number of VALID Transactions: %d", int32(config.TxNum)-Metric.Abort)
		reportCh <- fmt.Sprintf("Number of ABORTED Transactions: %d", Metric.Abort)
		reportCh <- fmt.Sprintf("Duration: %.3fs", float64(duration.Milliseconds())/float64(1e3))
		reportCh <- fmt.Sprintf("TPS: %f", float64(config.TxNum)*1e9/float64(duration.Nanoseconds()))
		reportCh <- fmt.Sprintf("Abort Rate: %.3f%%", float64(Metric.Abort)/float64(config.TxNum)*100)

		reportCh <- fmt.Sprintf("id    endorse(ms) integrate(ms) order&commit(ms)")
		for i, tk := range timeKeepers.transactions {
			endorsementDuration := float64(tk.EndorsedTime-tk.ProposedTime) / float64(1e6)
			if endorsementDuration < 0.0 {
				endorsementDuration = 0.0
			}

			integrationDuration := float64(tk.BroadcastTime-tk.EndorsedTime) / float64(1e6)
			if integrationDuration < 0.0 {
				integrationDuration = 0.0
			}

			orderingDuration := float64(tk.ObservedTime-tk.BroadcastTime) / float64(1e6)
			if orderingDuration < 0.0 {
				orderingDuration = 0.0
			}

			reportCh <- fmt.Sprintf("%-5d %11.2f %13.2f %16.2f",
				i,
				endorsementDuration,
				integrationDuration,
				orderingDuration,
			)
		}

		// Closing 'doneCh', a channel which is never sent an element, is a common technique to notify ending in Golang
		// More information: https://go101.org/article/channel-use-cases.html#check-closed-status
		close(doneCh)

		// Wait for WriteLogToFile() to return
		printWG.Wait()
	}
}

// End2End executes end-to-end benchmark on HLF
// An Element (i.e. a transaction) will go through the following channels
// unsignedCh -> signedCh -> endorsedCh -> integratedCh
func End2End() {
	initChannels()
	initTimeKeepers()

	printWG := &sync.WaitGroup{}
	go WriteLogToFile(printWG)

	initiator := NewInitiator(unsignedCh)
	signers := NewSigners(unsignedCh, signedChs)
	proposers := NewProposers(signedChs, endorsedCh)
	integrators := NewIntegrators(endorsedCh, integratedCh)
	broadcasters := NewBroadcasters(integratedCh)
	observer := NewObserver()

	proposers.StartAsync()
	integrators.StartAsync()
	broadcasters.StartAsync()
	observer.StartAsync()
	initiator.StartSync() // Block until all raw transactions are ready

	startTime := time.Now()
	signers.StartAsync()

	WaitObserverEnd(startTime, printWG)
}

//TODO
// BreakdownPhase1 sends proposals to endorsers
func BreakdownPhase1(config *Config, logger *log.Logger) {
	// crypto, err := config.LoadCrypto()
	// if err != nil {
	// 	return err
	// }
	// raw := make(chan *Element, burst)
	// signed := make([]chan *Element, config.EndorserNum)
	// processed := make(chan *Element, burst)
	// done := make(chan struct{})
	// errorCh := make(chan error, burst)
	// assembler := &Assembler{Signer: crypto, EndorserGroups: config.EndorserGroupNum, Conf: config}
	// go printBenchmark(config.LogPath, done)

	// for i := 0; i < config.EndorserNum; i++ {
	// 	signed[i] = make(chan *Element, burst)
	// }

	// proposers, err := NewProposers(config.ConnNum, config.ClientPerConnNum, config.Endorsers, burst, logger)
	// if err != nil {
	// 	return err
	// }

	// StartCreateProposal(config, crypto, raw, errorCh, logger)
	// time.Sleep(10 * time.Second)
	// start := time.Now()

	// for i := 0; i < config.SignerNum; i++ {
	// 	go assembler.StartSigner(raw, signed, errorCh, done)
	// }
	// proposers.Start(signed, processed, done, config)

	// // phase1: send proposals to endorsers
	// var cnt int32 = 0
	// var buffer [][]byte
	// var txids []string
	// for i := 0; i < config.TxNum; i++ {
	// 	select {
	// 	case err = <-errorCh:
	// 		return err
	// 	case tx := <-processed:
	// 		res, err := assembler.Assemble(tx)
	// 		if err != nil {
	// 			fmt.Println("error: assemble endorsement to envelop")
	// 			return err
	// 		}
	// 		bytes, err := json.Marshal(res.Envelope)
	// 		if err != nil {
	// 			fmt.Println("error: marshal envelop")
	// 			return err
	// 		}
	// 		cnt += 1
	// 		buffer = append(buffer, bytes)
	// 		txids = append(txids, tx.Txid)
	// 		if cnt+assembler.Abort >= int32(config.TxNum) {
	// 			break
	// 		}
	// 	}
	// }
	// duration := time.Since(start)

	// logger.Infof("Completed endorsing transactions.")
	// printCh <- fmt.Sprintf("tx: %d, duration: %+v, tps: %f", config.TxNum, duration, float64(config.TxNum)/duration.Seconds())
	// printCh <- fmt.Sprintf("abort rate because of the different ledger height: %d %.2f%%", assembler.Abort, float64(assembler.Abort)/float64(config.TxNum)*100)
	// close(done)
	// printWG.Wait()
	// // persistency
	// mfile, _ := os.Create(endorsementFilename)
	// defer mfile.Close()
	// mw := bufio.NewWriter(mfile)
	// for i := range buffer {
	// 	mw.Write(buffer[i])
	// 	mw.WriteByte('\n')
	// 	mw.WriteString(txids[i])
	// 	mw.WriteByte('\n')
	// }
	// mw.Flush()
	// return nil
}

//TODO
// BreakdownPhase2 Broadcast transactions to order
func BreakdownPhase2(config *Config, logger *log.Logger) {
	// crypto, err := config.LoadCrypto()
	// if err != nil {
	// 	return err
	// }
	// envs := make(chan *Element, burst)
	// done := make(chan struct{})
	// finishCh := make(chan struct{})
	// errorCh := make(chan error, burst)
	// go printBenchmark(config.LogPath, done)

	// broadcaster, err := NewBroadcasters(config.OrdererClients, config.Orderer, burst, logger)
	// if err != nil {
	// 	return err
	// }

	// observer, err := NewObserver(config.Channel, config.Committer, crypto, logger)
	// if err != nil {
	// 	return err
	// }

	// mfile, _ := os.Open(endorsementFilename)
	// defer mfile.Close()
	// mscanner := bufio.NewScanner(mfile)
	// var txids []string
	// TXs := make([]common.Envelope, config.TxNum)
	// i := 0
	// for mscanner.Scan() {
	// 	bytes := mscanner.Bytes()
	// 	json.Unmarshal(bytes, &TXs[i])
	// 	if mscanner.Scan() {
	// 		txid := mscanner.Text()
	// 		txids = append(txids, txid)
	// 	}
	// 	i++
	// }
	// items := make([]Element, config.TxNum)
	// for i := 0; i < len(txids); i++ {
	// 	var item Element
	// 	item.Envelope = &TXs[i]
	// 	item.Txid = txids[i]
	// 	txid2id[item.Txid] = i
	// 	items[i] = item
	// }

	// start := time.Now()
	// go func() {
	// 	for i := 0; i < len(items); i++ {
	// 		envs <- &items[i]
	// 	}
	// }()
	// broadcaster.Start(envs, config.Rate, errorCh, done)
	// var temp0 int32 = 0
	// go observer.Start(int32(len(txids)), errorCh, finishCh, start, &temp0)
	// for {
	// 	select {
	// 	case err = <-errorCh:
	// 		return err
	// 	case <-finishCh:
	// 		duration := time.Since(start)
	// 		logger.Infof("Completed processing transactions.")
	// 		printCh <- fmt.Sprintf("tx: %d, duration: %+v, tps: %f", config.TxNum, duration, float64(config.TxNum)/duration.Seconds())
	// 		close(done)
	// 		printWG.Wait()
	// 		return nil
	// 	}
	// }
}
