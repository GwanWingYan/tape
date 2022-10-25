package infra

import (
	"io/ioutil"

	"github.com/GwanWingYan/fabric-protos-go/msp"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	noSuchItemError = errors.New("No such item")
)

type Config struct {
	// Network
	Endorsers []Node `yaml:"endorsers"` // peers
	Committer Node   `yaml:"committer"` // the peer chosen to observe blocks from
	Orderer   Node   `yaml:"orderer"`   // orderer
	Channel   string `yaml:"channel"`   // name of the channel to be operated on

	// Chaincode
	Chaincode string   `yaml:"chaincode"` // chaincode name
	Version   string   `yaml:"version"`   // chaincode version
	Args      []string `yaml:"args"`      // chaincode arguments

	// Client identity
	MSPID      string `yaml:"mspid"`      // the MSP the client belongs
	PrivateKey string `yaml:"privateKey"` // client's private key
	SignCert   string `yaml:"signCert"`   // client's certificate

	End2End bool `yaml:"e2e"` // running mode

	Rate  float64 `yaml:"rate"`  // average speed of transaction generation
	Burst float64 `yaml:"burst"` // maximum speed of transaction generation

	TxNum  int    `yaml:"txNum"`  // number of transactions
	TxTime int    `yaml:"txTime"` // maximum execution time
	TxType string `yaml:"txType"` // transaction type ['put', 'conflict']

	ConnNum          int `yaml:"connNum"`          // number of connection
	ClientPerConnNum int `yaml:"clientPerConnNum"` // number of client per connection
	SignerNum        int `yaml:"signerNum"`        // number of signer
	IntegratorNum    int `yaml:"integratorNum"`    // number of integrator
	BroadcasterNum   int `yaml:"broadcasterNum"`   // number of orderer client
	EndorserNum      int // number of endorsers
	EndorserGroupNum int `yaml:"endorserGroupNum"` // number of endorser group

	// If true, let the protoutil generate txid automatically
	// If false, encode the txid by us
	// WARNING: Must modify the code in core/endorser/msgvalidation.go:Validate() and
	// protoutil/proputils.go:ComputeTxID (v2) before compilation
	CheckTxID bool `yaml:"checkTxID"`

	// If true, print the read set and write set to STDOUT
	CheckRWSet bool `yaml:"checkRWSet"`

	LogPath string `yaml:"logPath"` // path of the log file
	Seed    int    `yaml:"seed"`    // random seed
}

type Node struct {
	Address       string `yaml:"address"`
	TLSCACert     string `yaml:"tlsCACert"`
	TLSCAKey      string `yaml:"tlsCAKey"`
	TLSCARoot     string `yaml:"tlsCARoot"`
	TLSCACertByte []byte
	TLSCAKeyByte  []byte
	TLSCARootByte []byte
}

func LoadConfigFile(config *Config, f string) error {

	raw, err := ioutil.ReadFile(f)
	if err != nil {
		return errors.Wrapf(err, "error loading %s", f)
	}
	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return errors.Wrapf(err, "error unmarshal %s", f)
	}

	for i := range config.Endorsers {
		err = config.Endorsers[i].loadConfig()
		if err != nil {
			return err
		}
	}
	config.EndorserNum = len(config.Endorsers)

	err = config.Committer.loadConfig()
	if err != nil {
		return err
	}

	err = config.Orderer.loadConfig()
	if err != nil {
		return err
	}

	return nil
}

// LoadCrypto loads the client specified in the configuration file
func (c Config) LoadCrypto() (*Crypto, error) {
	cc := CryptoConfig{
		MSPID:    c.MSPID,
		PrivKey:  c.PrivateKey,
		SignCert: c.SignCert,
	}

	priv, err := GetPrivateKey(cc.PrivKey)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading priv key")
	}

	cert, certBytes, err := GetCertificate(cc.SignCert)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading certificate")
	}

	id := &msp.SerializedIdentity{
		Mspid:   cc.MSPID,
		IdBytes: certBytes,
	}

	name, err := proto.Marshal(id)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting msp id")
	}

	return &Crypto{
		Creator:  name,
		PrivKey:  priv,
		SignCert: cert,
	}, nil
}

func GetTLSCACerts(file string) ([]byte, error) {
	if len(file) == 0 {
		return nil, noSuchItemError
	}

	in, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading %s", file)
	}

	return in, nil
}

//TODO
func (n *Node) loadConfig() error {
	certByte, err := GetTLSCACerts(n.TLSCACert)
	if err != nil && err != noSuchItemError {
		return errors.Wrapf(err, "fail to load TLS CA Cert %s", n.TLSCACert)
	}

	keyByte, err := GetTLSCACerts(n.TLSCAKey)
	if err != nil && err != noSuchItemError {
		return errors.Wrapf(err, "fail to load TLS CA Key %s", n.TLSCAKey)
	}

	rootByte, err := GetTLSCACerts(n.TLSCARoot)
	if err != nil && err != noSuchItemError {
		return errors.Wrapf(err, "fail to load TLS CA Root %s", n.TLSCARoot)
	}

	n.TLSCACertByte = certByte
	n.TLSCAKeyByte = keyByte
	n.TLSCARootByte = rootByte

	return nil
}
