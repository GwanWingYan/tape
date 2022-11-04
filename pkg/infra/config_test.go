package infra_test

import (
	"io/ioutil"
	"os"
	"text/template"

	"github.com/GwanWingYan/tape/pkg/infra"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func generateConfigFile(FileName string, values interface{}) {
	var Text = `# Definition of nodes
org1peer0: &org1peer0
  addr: peer0.org1.example.com:7051
  tlsCACert: {{.TlsFile}}
org2peer0: &org2peer0
  addr: peer0.org2.example.com:7051
  tlsCACert: {{.TlsFile}}
org0orderer0: &org0orderer0
  addr: orderer.example.com:7050
  tlsCACert: {{.TlsFile}}

endorsers:
  - *org1peer0
  - *org2peer0
committer: *org2peer0
orderer: *org0orderer0

channel: mychannel
chaincode: mycc
args:
  - invoke
  - a
  - b
  - 1
mspid: Org1MSP
privateKey: /path/to/private.key
signCert: /path/to/sign.cert
connNum: 20
clientPerConnNum: 40
`
	tmpl, err := template.New("test").Parse(Text)
	if err != nil {
		panic(err)
	}
	file, err := os.OpenFile(FileName, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	err = tmpl.Execute(file, values)
	if err != nil {
		panic(err)
	}
}

var _ = Describe("Config", func() {

	Context("good", func() {
		It("successful loads", func() {
			tlsFile, err := ioutil.TempFile("", "dummy-*.pem")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tlsFile.Name())

			_, err = tlsFile.Write([]byte("a"))
			Expect(err).NotTo(HaveOccurred())

			f, _ := ioutil.TempFile("", "config-*.yaml")
			defer os.Remove(f.Name())

			generateConfigFile(f.Name(), struct{ TlsFile string }{tlsFile.Name()})

			c, err := infra.LoadConfigFromFile(f.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(c).To(Equal(infra.Config{
				Endorsers: []infra.Node{
					{Addr: "peer0.org1.example.com:7051", TLSCACert: tlsFile.Name(), TLSCACertByte: []byte("a")},
					{Addr: "peer0.org2.example.com:7051", TLSCACert: tlsFile.Name(), TLSCACertByte: []byte("a")},
				},
				Committer:        infra.Node{Addr: "peer0.org2.example.com:7051", TLSCACert: tlsFile.Name(), TLSCACertByte: []byte("a")},
				Orderer:          infra.Node{Addr: "orderer.example.com:7050", TLSCACert: tlsFile.Name(), TLSCACertByte: []byte("a")},
				Channel:          "mychannel",
				Chaincode:        "mycc",
				Version:          "",
				Args:             []string{"invoke", "a", "b", "1"},
				MSPID:            "Org1MSP",
				PrivateKey:       "/path/to/private.key",
				SignCert:         "/path/to/sign.cert",
				ConnNum:          20,
				ClientPerConnNum: 40,
			}))
			_, err = c.LoadCrypto()
			Expect(err).Should(MatchError(ContainSubstring("error loading priv key")))
		})
	})

	Context("bad", func() {
		It("fails to load missing config file", func() {
			_, err := infra.LoadConfigFromFile("invalid_file")
			Expect(err).Should(MatchError(ContainSubstring("invalid_file")))
		})

		It("fails to load invalid config file", func() {

			f, _ := ioutil.TempFile("", "config-*.yaml")
			defer os.Remove(f.Name())

			generateConfigFile(f.Name(), struct{ TlsFile string }{"invalid_file"})

			_, err := infra.LoadConfigFromFile(f.Name())
			Expect(err).Should(MatchError(ContainSubstring("invalid_file")))
		})
	})
})
