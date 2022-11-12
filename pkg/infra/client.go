package infra

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/GwanWingYan/fabric-protos-go/orderer"
	"github.com/GwanWingYan/fabric-protos-go/peer"
	"github.com/GwanWingYan/tape/pkg/comm"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	MAX_TRY = 3
)

func CreateGRPCClient(node Node) (*comm.GRPCClient, error) {
	var certs [][]byte
	if node.TLSCACertByte != nil {
		certs = append(certs, node.TLSCACertByte)
	}
	config := comm.ClientConfig{}
	config.Timeout = 5 * time.Second
	config.SecOpts = comm.SecureOptions{
		UseTLS:            false,
		RequireClientCert: false,
		ServerRootCAs:     certs,
	}

	if len(certs) > 0 {
		config.SecOpts.UseTLS = true
		if len(node.TLSCAKey) > 0 && len(node.TLSCARoot) > 0 {
			config.SecOpts.RequireClientCert = true
			config.SecOpts.Certificate = node.TLSCACertByte
			config.SecOpts.Key = node.TLSCAKeyByte
			if node.TLSCARootByte != nil {
				config.SecOpts.ClientRootCAs = append(config.SecOpts.ClientRootCAs, node.TLSCARootByte)
			}
		}
	}

	grpcClient, err := comm.NewGRPCClient(config)
	//to do: unit test for this error, current fails to make case for this
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to %s", node.Address)
	}

	return grpcClient, nil
}

func CreateEndorserClient(node Node) (peer.EndorserClient, error) {
	conn, err := DialConnection(node)
	if err != nil {
		return nil, err
	}
	return peer.NewEndorserClient(conn), nil
}

func CreateBroadcastClient(node Node) (orderer.AtomicBroadcast_BroadcastClient, error) {
	conn, err := DialConnection(node)
	if err != nil {
		return nil, err
	}
	return orderer.NewAtomicBroadcastClient(conn).Broadcast(context.Background())
}

func CreateDeliverFilteredClient() (peer.Deliver_DeliverFilteredClient, error) {
	conn, err := DialConnection(config.Committer)
	if err != nil {
		return nil, err
	}
	return peer.NewDeliverClient(conn).DeliverFiltered(context.Background())
}

func DialConnection(node Node) (*grpc.ClientConn, error) {
	gRPCClient, err := CreateGRPCClient(node)
	if err != nil {
		return nil, err
	}
	var connError error
	var conn *grpc.ClientConn
	for i := 1; i <= MAX_TRY; i++ {
		conn, connError = gRPCClient.NewConnection(node.Address, func(tlsConfig *tls.Config) { tlsConfig.InsecureSkipVerify = true })
		if connError == nil {
			return conn, nil
		}
	}
	return nil, errors.Wrapf(connError, "failed to dial %s: %s", node.Address)
}
