// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package repo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/gcash/bchutil"
	"github.com/jessevdk/go-flags"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultLogFilename    = "ilxd.log"
	defaultConfigFilename = "ilxd.conf"
	defaultGrpcPort       = "5001"
)

var (
	DefaultHomeDir    = AppDataDir("ilxd", false)
	defaultConfigFile = filepath.Join(DefaultHomeDir, defaultConfigFilename)

	defaultListenAddrs = []string{
		"/ip4/0.0.0.0/tcp/9001",
		"/ip6/::/tcp/9001",
		"/ip4/0.0.0.0/udp/9001/quic",
		"/ip6/::/udp/9001/quic",
	}
)

// Config defines the configuration options for the node.
//
// See LoadConfig for details on the configuration load process.
type Config struct {
	ShowVersion       bool     `short:"v" long:"version" description:"Display version information and exit"`
	ConfigFile        string   `short:"C" long:"configfile" description:"Path to configuration file"`
	DataDir           string   `short:"d" long:"datadir" description:"Directory to store data"`
	LogDir            string   `long:"logdir" description:"Directory to log output."`
	LogLevel          string   `short:"l" long:"loglevel" description:"Set the logging level [debug, info, notice, error, alert, critical, emergency]." default:"info"`
	SeedAddrs         []string `long:"seedaddr" description:"Override the default seed addresses with the provided values"`
	ListenAddrs       []string `long:"listenaddr" description:"Override the default listen addresses with the provided values"`
	Testnet           bool     `short:"t" long:"testnet" description:"Use the test network"`
	Regest            bool     `short:"r" long:"regtest" description:"Use regression testing mode"`
	DisableNATPortMap bool     `long:"noupnp" description:"Disable use of upnp"`
	UserAgent         string   `long:"useragent" description:"A custom user agent to advertise to the network"`

	RPCOpts RPCOptions `group:"RPC Options"`
}

type RPCOptions struct {
	RPCCert       string   `long:"rpccert" description:"A path to the SSL certificate to use with gRPC"`
	RPCKey        string   `long:"rpckey" description:"A path to the SSL key to use with gRPC"`
	ExternalIPs   []string `long:"externalips" description:"This option should be used to specify the external IP address if using the auto-generated SSL certificate"`
	GrpcListeners []string `long:"grpclisten" description:"Add an interface/port to listen for experimental gRPC connections (default port:5001)"`
	GrpcAuthToken string   `long:"grpcauthtoken" description:"Set a token here if you want to enable client authentication with gRPC"`
}

// LoadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
// 	1) Start with a default config with sane settings
// 	2) Pre-parse the command line to check for an alternative config file
// 	3) Load configuration file overwriting defaults with any specified options
// 	4) Parse CLI options and overwrite/add any specified options
//
// The above results in proper functionality without any config settings
// while still allowing the user to override settings with config files and
// command line options.  Command line options always take precedence.
func LoadConfig() (*Config, error) {
	// Default config.
	cfg := Config{
		DataDir:    DefaultHomeDir,
		ConfigFile: defaultConfigFile,
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	preCfg := cfg
	preParser := flags.NewParser(&cfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, err
		}
	}
	if cfg.DataDir != "" {
		preCfg.ConfigFile = filepath.Join(cfg.DataDir, defaultConfigFilename)
	}

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", VersionString())
		os.Exit(0)
	}

	// Load additional config from file.
	var configFileError error
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := os.Stat(preCfg.ConfigFile); os.IsNotExist(err) {
		err := createDefaultConfigFile(preCfg.ConfigFile, cfg.Testnet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating a "+
				"default config file: %v\n", err)
		}
	}

	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, err
		}
		configFileError = err
	}

	if cfg.Testnet && cfg.Regest {
		return nil, errors.New("invalid combination of testnet and regtest")
	}

	netStr := "mainnet"
	if cfg.Testnet {
		netStr = "testnet"
	}

	if cfg.LogDir == "" {
		cfg.LogDir = cleanAndExpandPath(path.Join(cfg.DataDir, "logs", netStr))
	}

	// Warn about missing config file only after all other configuration is
	// done.  This prevents the warning on help messages and invalid
	// options.  Note this should go directly before the return.
	if configFileError != nil {
		log.Errorf("%v", configFileError)
	}

	// Default RPC to listen on localhost only.
	if len(cfg.RPCOpts.GrpcListeners) == 0 {
		addrs, err := net.LookupHost("localhost")
		if err != nil {
			return nil, err
		}
		cfg.RPCOpts.GrpcListeners = make([]string, 0, len(addrs))
		for _, addr := range addrs {
			addr = net.JoinHostPort(addr, defaultGrpcPort)
			cfg.RPCOpts.GrpcListeners = append(cfg.RPCOpts.GrpcListeners, addr)
		}
	}

	if cfg.RPCOpts.RPCCert == "" && cfg.RPCOpts.RPCKey == "" {
		cfg.RPCOpts.RPCCert = path.Join(cfg.DataDir, "rpc.cert")
		cfg.RPCOpts.RPCKey = path.Join(cfg.DataDir, "rpc.key")
	}

	cfg.DataDir = cleanAndExpandPath(path.Join(cfg.DataDir, netStr))
	if !fileExists(cfg.RPCOpts.RPCKey) && !fileExists(cfg.RPCOpts.RPCCert) {
		err := genCertPair(cfg.RPCOpts.RPCCert, cfg.RPCOpts.RPCKey, cfg.RPCOpts.ExternalIPs)
		if err != nil {
			return nil, err
		}
	}

	cfg.UserAgent = "/ilxd/" + VersionString() + "/" + cfg.UserAgent

	return &cfg, nil
}

// createDefaultConfig copies the sample-obcrawler.conf content to the given destination path,
// and populates it with some randomly generated RPC username and password.
func createDefaultConfigFile(destinationPath string, testnet bool) error {
	// Create the destination directory if it does not exists
	err := os.MkdirAll(filepath.Dir(destinationPath), 0700)
	if err != nil {
		return err
	}

	sampleBytes, err := Asset("sample-ilxd.conf")
	if err != nil {
		return err
	}
	src := bytes.NewReader(sampleBytes)

	dest, err := os.OpenFile(destinationPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	// We copy every line from the sample config file to the destination,
	// only replacing the bootstrap addrs.
	reader := bufio.NewReader(src)
	for err != io.EOF {
		var line string
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if _, err := dest.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(DefaultHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// genCertPair generates a key/cert pair to the paths provided.
func genCertPair(certFile, keyFile string, externalIPs []string) error {
	log.Infof("Generating TLS certificates...")

	org := "obx autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := bchutil.NewTLSCertPair(org, validUntil, externalIPs)
	if err != nil {
		return err
	}

	// Write cert and key files.
	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	log.Infof("Done generating TLS certificates")
	return nil
}

// filesExists reports whether the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
