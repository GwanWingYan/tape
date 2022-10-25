package infra

import (
	"bufio"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	accountFilePath     = "ACCOUNTS.txt"
	transactionFilePath = "TRANSACTIONS.txt"
)

var (
	chs = []rune("qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890!@#$%^&*()=")
)

func getName(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chs[rand.Intn(len(chs))]
	}
	return string(b)
}

func randomId(n int) int {
	res := rand.Intn(n)
	return res
}

type WorkloadGenerator struct {
	accounts []string
}

func NewWorkloadGenerator() *WorkloadGenerator {
	return &WorkloadGenerator{}
}

// generate generates an argument list for one chaincode invocation
// currently, two types of transaction are supported:
// 		'put' for CreateAccount
//		'conflict' for SendPayment
// TODO: support other transactions
// (e.g., Amalgamate, TransactionsSavings, WriteCheck, DepositChecking)
func (wg *WorkloadGenerator) generate() []string {
	var res []string

	switch config.TxType {
	case "put":
		id := getName(64)                    // generate a random name for customer
		res = append(res, "CreateAccount")   // function name
		res = append(res, id)                // customer id
		res = append(res, id)                // customer name
		res = append(res, strconv.Itoa(1e9)) // savings balance
		res = append(res, strconv.Itoa(1e9)) // checking balance
	case "conflict":
		// randomly select 2 different accounts as sender and receiver
		src := rand.Intn(len(wg.accounts))
		dst := rand.Intn(len(wg.accounts))
		for src == dst {
			dst = rand.Intn(len(wg.accounts))
		}
		res = append(res, "SendPayment")    // function name
		res = append(res, wg.accounts[src]) // sender id
		res = append(res, wg.accounts[dst]) // receiver id
		res = append(res, "1")              // amount
	}

	return res
}

// GenerateWorkload generates TxNum of chaincode invocations
// currently, two types of transaction are supported:
// 		'put' for CreateAccount
//		'conflict' for SendPayment
// NOTE: To execute 'conflict', we have to first execute 'put' to create
// accounts.
func (wg *WorkloadGenerator) GenerateWorkload() [][]string {
	if config.Seed == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(int64(config.Seed))
	}

	// Prepare
	switch config.TxType {
	case "put":
		logger.Infof("Create %d accounts\n", config.TxNum)
	case "conflict":
		// try to load all accounts' id from file
		if _, err := os.Stat(accountFilePath); os.IsNotExist(err) {
			logger.Fatalf("Account file %s not found: %v\n", accountFilePath, err)
		}
		af, _ := os.Open(accountFilePath)
		defer af.Close()
		input := bufio.NewScanner(af)
		for input.Scan() {
			wg.accounts = append(wg.accounts, input.Text())
		}
		logger.Infof("Read %d accounts from %s\n", len(wg.accounts), accountFilePath)
	}

	// Generates TxNum of chaincode invocations argument list
	var res [][]string
	for i := 0; i < config.TxNum; i++ {
		res = append(res, wg.generate())
	}

	// Write the argument list to file
	os.Remove(transactionFilePath)
	tf, err := os.Create(transactionFilePath)
	defer tf.Close()
	if err != nil {
		logger.Fatalf("Failed to create file %s: %v\n", transactionFilePath, err)
	}
	for i := 0; i < config.TxNum; i++ {
		tf.WriteString(strconv.Itoa(i) + " " + strings.Join(res[i], " ") + "\n")
	}

	// If we are creating new accounts, we have to write all the accounts' id to file
	if config.TxType == "put" {
		af, err := os.Create(accountFilePath)
		defer af.Close()
		if err != nil {
			logger.Fatalf("Failed to create file %s: %v\n", accountFilePath, err)
		}
		for i := 0; i < config.TxNum; i++ {
			// only record the account id
			af.WriteString(res[i][1] + "\n")
		}
	}

	return res
}
