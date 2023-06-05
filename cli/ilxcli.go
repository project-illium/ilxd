// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/rpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
)

type options struct {
	ShowVersion bool   `short:"v" long:"version" description:"Display version information and exit"`
	AuthToken   string `short:"t" long:"authtoken" description:"The ilxd node gRPC authentican token if needed"`
	ServerAddr  string `short:"a" long:"serveraddr" description:"The address of the ilxd gRPC server (in multiaddr format)" default:"/ip4/127.0.0.1/tcp/5001"`
	RPCCert     string `long:"rpccert" description:"A path to the SSL certificate to use with gRPC (this is only need if using a self-signed cert)" default:"~/.ilxd/rpc.cert"`
}

func main() {
	var opts options
	parser := flags.NewNamedParser("ilxcli", flags.HelpFlag)
	parser.AddGroup("Connection options", "Configuration options for connecting to ilxd", &opts)
	parser.AddCommand("getblockchaininfo", "todo", "todo", &GetBlockchainInfo{})
	if _, err := parser.Parse(); err != nil {
		log.Fatal(err)
	}
}

func makeBlockchainClient() (pb.BlockchainServiceClient, error) {
	var opts options
	parser := flags.NewParser(&opts, flags.HelpFlag)
	if _, err := parser.Parse(); err != nil {
		return nil, err
	}
	certFile := repo.CleanAndExpandPath(opts.RPCCert)

	creds, err := credentials.NewClientTLSFromFile(certFile, "localhost")
	if err != nil {
		return nil, err
	}
	ma, err := multiaddr.NewMultiaddr(opts.ServerAddr)
	if err != nil {
		return nil, err
	}

	netAddr, err := manet.ToNetAddr(ma)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(netAddr.String(), grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return pb.NewBlockchainServiceClient(conn), nil
}
