// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/stake"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/project-illium/walletlib"
	"google.golang.org/protobuf/proto"
	"strings"
)

type GetBalance struct {
	opts *options
}

func (x *GetBalance) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetBalance(makeContext(x.opts.AuthToken), &pb.GetBalanceRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Balance)
	return nil
}

type GetWalletSeed struct {
	opts *options
}

func (x *GetWalletSeed) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetWalletSeed(makeContext(x.opts.AuthToken), &pb.GetWalletSeedRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.MnemonicSeed)
	return nil
}

type GetAddress struct {
	opts *options
}

func (x *GetAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetAddress(makeContext(x.opts.AuthToken), &pb.GetAddressRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Address)
	return nil
}

type GetTimelockedAddress struct {
	LockUntil int64 `short:"l" long:"lockuntil" description:"A unix timestamp to lock the coins until (in seconds)."`
	opts      *options
}

func (x *GetTimelockedAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetTimelockedAddress(makeContext(x.opts.AuthToken), &pb.GetTimelockedAddressRequest{
		LockUntil: x.LockUntil,
	})
	if err != nil {
		return err
	}
	fmt.Println(resp.Address)
	return nil
}

type GetAddresses struct {
	opts *options
}

func (x *GetAddresses) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetAddresses(makeContext(x.opts.AuthToken), &pb.GetAddressesRequest{})
	if err != nil {
		return err
	}
	for _, addr := range resp.Addresses {
		fmt.Println(addr)
	}
	return nil
}

type GetNewAddress struct {
	opts *options
}

func (x *GetNewAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetNewAddress(makeContext(x.opts.AuthToken), &pb.GetNewAddressRequest{})
	if err != nil {
		return err
	}
	fmt.Println(resp.Address)
	return nil
}

type GetAddrInfo struct {
	Address string `short:"a" long:"addr" description:"The address to get the info for"`
	opts    *options
}

func (x *GetAddrInfo) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetAddressInfo(makeContext(x.opts.AuthToken), &pb.GetAddressInfoRequest{
		Address: x.Address,
	})
	if err != nil {
		return err
	}

	kp := struct {
		Addr           string             `json:"address"`
		LockingScript  types.HexEncodable `json:"lockingScript"`
		ViewPrivateKey types.HexEncodable `json:"viewPrivateKey"`
		WatchOnly      bool               `json:"watchOnly"`
	}{
		Addr:           resp.Address,
		LockingScript:  resp.LockingScript,
		ViewPrivateKey: resp.ViewPrivateKey,
		WatchOnly:      resp.WatchOnly,
	}
	out, err := json.MarshalIndent(&kp, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type GetTransactions struct {
	opts *options
}

func (x *GetTransactions) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetTransactions(makeContext(x.opts.AuthToken), &pb.GetTransactionsRequest{})
	if err != nil {
		return err
	}
	type tx struct {
		Txid     types.HexEncodable `json:"txid"`
		NetCoins int64              `json:"netCoins"`
		Inputs   []interface{}      `json:"inputs"`
		Outputs  []interface{}      `json:"outputs"`
	}
	txs := make([]tx, 0, len(resp.Txs))
	for _, rtx := range resp.Txs {
		txs = append(txs, tx{
			Txid:     rtx.Transaction_ID,
			NetCoins: rtx.NetCoins,
			Inputs:   pbIOtoIO(rtx.Inputs),
			Outputs:  pbIOtoIO(rtx.Outputs),
		})
	}
	out, err := json.MarshalIndent(txs, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetUtxos struct {
	opts *options
}

func (x *GetUtxos) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.GetUtxos(makeContext(x.opts.AuthToken), &pb.GetUtxosRequest{})
	if err != nil {
		return err
	}
	type utxo struct {
		Address     string             `json:"address"`
		Commitment  types.HexEncodable `json:"commitment"`
		Amount      uint64             `json:"amount"`
		WatchOnly   bool               `json:"watchOnly"`
		Staked      bool               `json:"staked"`
		LockedUntil int64              `json:"lockedUntil"`
	}
	utxos := make([]utxo, 0, len(resp.Utxos))
	for _, ut := range resp.Utxos {
		utxos = append(utxos, utxo{
			Address:     ut.Address,
			Commitment:  ut.Commitment,
			Amount:      ut.Amount,
			WatchOnly:   ut.WatchOnly,
			Staked:      ut.Staked,
			LockedUntil: ut.LockedUntill,
		})
	}
	out, err := json.MarshalIndent(utxos, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

type GetPrivateKey struct {
	Address string `short:"a" long:"addr" description:"The address to get the private key for"`
	opts    *options
}

func (x *GetPrivateKey) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	resp, err := client.GetPrivateKey(makeContext(x.opts.AuthToken), &pb.GetPrivateKeyRequest{
		Address: x.Address,
	})
	if err != nil {
		return err
	}

	key, err := crypto.UnmarshalPrivateKey(resp.SerializedKeys)
	if err != nil {
		return err
	}
	walletKey, ok := key.(*walletlib.WalletPrivateKey)
	if !ok {
		return errors.New("error decoding key")
	}

	fmt.Println(walletlib.EncodePrivateKey(walletKey))
	return nil
}

type ImportAddress struct {
	Address          string `short:"a" long:"addr" description:"The address to import"`
	LockingScript    string `short:"u" long:"lockingscript" description:"The locking script for the address. Serialized as hex string"`
	ViewPrivateKey   string `short:"k" long:"viewkey" description:"The view private key for the address. Serialized as hex string."`
	Rescan           bool   `short:"r" long:"rescan" description:"Whether or not to rescan the blockchain to try to detect transactions for this address."`
	RescanFromHeight uint32 `short:"t" long:"rescanheight" description:"The height of the chain to rescan from. Selecting a height close to the address birthday saves resources."`
	opts             *options
}

func (x *ImportAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	lockingScriptBytes, err := hex.DecodeString(x.LockingScript)
	if err != nil {
		return err
	}
	privKeyBytes, err := hex.DecodeString(x.ViewPrivateKey)
	if err != nil {
		return err
	}

	_, err = client.ImportAddress(makeContext(x.opts.AuthToken), &pb.ImportAddressRequest{
		Address:          x.Address,
		LockingScript:    lockingScriptBytes,
		ViewPrivateKey:   privKeyBytes,
		Rescan:           x.Rescan,
		RescanFromHeight: x.RescanFromHeight,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type CreateMultisigSpendKeypair struct {
	opts *options
}

func (x *CreateMultisigSpendKeypair) Execute(args []string) error {
	priv, pub, err := icrypto.GenerateNovaKey(rand.Reader)
	if err != nil {
		return err
	}
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return err
	}
	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return err
	}

	kp := struct {
		PrivateKey types.HexEncodable `json:"privateKey"`
		PublicKey  types.HexEncodable `json:"publicKey"`
	}{
		PrivateKey: privBytes,
		PublicKey:  pubBytes,
	}
	out, err := json.MarshalIndent(&kp, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type CreateMultisigViewKeypair struct {
	opts *options
}

func (x *CreateMultisigViewKeypair) Execute(args []string) error {
	priv, pub, err := icrypto.GenerateCurve25519Key(rand.Reader)
	if err != nil {
		return err
	}
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return err
	}
	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return err
	}

	kp := struct {
		PrivateKey types.HexEncodable `json:"privateKey"`
		PublicKey  types.HexEncodable `json:"publicKey"`
	}{
		PrivateKey: privBytes,
		PublicKey:  pubBytes,
	}
	out, err := json.MarshalIndent(&kp, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type CreateMultisigAddress struct {
	ViewPubKey string   `short:"k" long:"viewpubkey" description:"The view public key for the address. Serialized as hex string."`
	Pubkeys    []string `short:"p" long:"pubkey" description:"One or more public keys to use with the address. Serialized as a hex string. Use this option more than once for more than one key."`
	Threshold  uint32   `short:"t" long:"threshold" description:"The number of keys needing to sign to the spend from this address."`
	Net        string   `short:"n" long:"net" description:"Which network the address is for: [mainnet, testnet, regtest] Default: mainnet"`
	opts       *options
}

func (x *CreateMultisigAddress) Execute(args []string) error {
	pubkeys := make([][]byte, 0, len(x.Pubkeys))
	for _, p := range x.Pubkeys {
		keyBytes, err := hex.DecodeString(p)
		if err != nil {
			return err
		}

		pubkey, err := crypto.UnmarshalPublicKey(keyBytes)
		if err != nil {
			return err
		}

		novaKey, ok := pubkey.(*icrypto.NovaPublicKey)
		if !ok {
			return errors.New("pubkey is not type Nova public key")
		}
		pubX, pubY := novaKey.ToXY()
		pubkeys = append(pubkeys, pubX, pubY)
	}

	viewKeyBytes, err := hex.DecodeString(x.ViewPubKey)
	if err != nil {
		return err
	}
	viewKey, err := crypto.UnmarshalPublicKey(viewKeyBytes)
	if err != nil {
		return err
	}

	scriptCommitment, err := zk.LurkCommit(zk.MultisigScript())
	if err != nil {
		return err
	}

	threshold := make([]byte, 4)
	binary.BigEndian.PutUint32(threshold, x.Threshold)

	lockingScript := types.LockingScript{
		ScriptCommitment: types.NewID(scriptCommitment),
		LockingParams:    [][]byte{threshold},
	}
	lockingScript.LockingParams = append(lockingScript.LockingParams, pubkeys...)

	var chainParams *params.NetworkParams
	switch strings.ToLower(x.Net) {
	case "mainnet", "":
		chainParams = &params.MainnetParams
	case "testnet":
		chainParams = &params.Testnet1Params
	case "regtest":
		chainParams = &params.RegestParams
	case "alphanet":
		chainParams = &params.AlphanetParams
	default:
		return errors.New("invalid net")
	}

	addr, err := walletlib.NewBasicAddress(lockingScript, viewKey, chainParams)
	if err != nil {
		return err
	}

	kp := struct {
		Addr          string             `json:"address"`
		LockingScript types.HexEncodable `json:"lockingScript"`
	}{
		Addr:          addr.String(),
		LockingScript: lockingScript.Serialize(),
	}
	out, err := json.MarshalIndent(&kp, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

type CreateMultiSignature struct {
	Tx         string `short:"t" long:"tx" description:"A transaction to sign (either Transaction or RawTransaction). Serialized as hex string. Use this or sighash."`
	SigHash    string `short:"h" long:"sighash" description:"A sighash to sign. Serialized as hex string. Use this or tx."`
	PrivateKey string `short:"k" long:"privkey" description:"A spend private key. Serialized as hex string."`
	opts       *options
}

func (x *CreateMultiSignature) Execute(args []string) error {
	privKeyBytes, err := hex.DecodeString(x.PrivateKey)
	if err != nil {
		return err
	}
	privKey, err := crypto.UnmarshalPrivateKey(privKeyBytes)
	if err != nil {
		return err
	}

	var sigHash []byte
	if x.Tx != "" {
		txBytes, err := hex.DecodeString(x.Tx)
		if err != nil {
			return err
		}
		tx := new(transactions.Transaction)
		if err := proto.Unmarshal(txBytes, tx); err != nil {
			var raw pb.RawTransaction
			if err := proto.Unmarshal(txBytes, &raw); err != nil {
				return err
			}
			tx = raw.Tx
		}
		if tx.GetStandardTransaction() != nil {
			sigHash, err = tx.GetStandardTransaction().SigHash()
			if err != nil {
				return err
			}
		} else if tx.GetMintTransaction() != nil {
			sigHash, err = tx.GetMintTransaction().SigHash()
			if err != nil {
				return err
			}
		} else if tx.GetStakeTransaction() != nil {
			sigHash, err = tx.GetStakeTransaction().SigHash()
			if err != nil {
				return err
			}
		}

	} else if x.SigHash != "" {
		sigHash, err = hex.DecodeString(x.SigHash)
		if err != nil {
			return err
		}
	} else {
		return errors.New("tx or sighash required")
	}

	sig, err := privKey.Sign(sigHash)
	if err != nil {
		return err
	}

	fmt.Println(hex.EncodeToString(sig))
	return nil
}

type ProveMultisig struct {
	Tx         string   `short:"t" long:"tx" description:"The transaction to prove. Serialized as hex string."`
	Serialize  bool     `short:"s" long:"serialize" description:"Serialize the output as a hex string. If false it will be JSON."`
	Signatures []string `short:"s" long:"sig" description:"A signature covering the tranaction's sighash. Use this option more than once to add more signatures.'"`
	opts       *options
}

func (x *ProveMultisig) Execute(args []string) error {

	txBytes, err := hex.DecodeString(x.Tx)
	if err != nil {
		return err
	}
	var rawTx pb.RawTransaction
	if err := proto.Unmarshal(txBytes, &rawTx); err != nil {
		return err
	}

	sigs := make([][]byte, 0, len(x.Signatures))
	for _, s := range x.Signatures {
		sig, err := hex.DecodeString(s)
		if err != nil {
			return err
		}
		sigs = append(sigs, sig)
	}

	if rawTx.Tx == nil {
		return errors.New("raw transaction tx is nil")
	}

	standardTx := rawTx.Tx.GetStandardTransaction()
	if standardTx == nil {
		return errors.New("standard tx is nil")
	}

	sighash, err := standardTx.SigHash()
	if err != nil {
		return err
	}

	// Create the transaction zk proof
	privateParams := &standard.PrivateParams{
		Inputs:  []standard.PrivateInput{},
		Outputs: []standard.PrivateOutput{},
	}

	nullifiers := make([][]byte, 0, len(rawTx.Inputs))
	for _, in := range rawTx.Inputs {
		unlockingParams, err := zk.MakeMultisigUnlockingParams(in.ScriptParams[1:], sigs, sighash)
		if err != nil {
			return err
		}

		privIn := standard.PrivateInput{
			SpendNote: types.SpendNote{
				Amount: types.Amount(in.Amount),
			},
			CommitmentIndex: in.TxoProof.Index,
			InclusionProof: standard.InclusionProof{
				Hashes: in.TxoProof.Hashes,
				Flags:  in.TxoProof.Flags,
			},
			ScriptCommitment: in.ScriptCommitment,
			ScriptParams:     in.ScriptParams,
			UnlockingParams:  unlockingParams,
		}
		copy(privIn.Salt[:], in.Salt)
		copy(privIn.AssetID[:], in.Asset_ID)

		state := new(types.State)
		if err := state.Deserialize(in.State); err != nil {
			return err
		}
		privIn.State = *state

		privateParams.Inputs = append(privateParams.Inputs, privIn)

		nullifier, err := types.CalculateNullifier(in.TxoProof.Index, privIn.Salt, privIn.ScriptCommitment, privIn.ScriptParams...)
		if err != nil {
			return err
		}
		nullifiers = append(nullifiers, nullifier.Bytes())
	}
	for _, out := range rawTx.Outputs {
		privOut := standard.PrivateOutput{
			SpendNote: types.SpendNote{
				ScriptHash: types.NewID(out.ScriptHash),
				Amount:     types.Amount(out.Amount),
			},
		}
		copy(privOut.Salt[:], out.Salt)
		copy(privOut.AssetID[:], out.Asset_ID)
		state := new(types.State)
		if err := state.Deserialize(out.State); err != nil {
			return err
		}
		privOut.State = *state

		privateParams.Outputs = append(privateParams.Outputs, privOut)
	}

	publicParams := &standard.PublicParams{
		TXORoot:    standardTx.TxoRoot,
		SigHash:    sighash,
		Nullifiers: nullifiers,
		Fee:        standardTx.Fee,
	}

	for _, out := range standardTx.Outputs {
		publicParams.Outputs = append(publicParams.Outputs, standard.PublicOutput{
			Commitment: out.Commitment,
			CipherText: out.Ciphertext,
		})
	}

	proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicParams)
	if err != nil {
		return err
	}

	standardTx.Proof = proof

	tx := transactions.WrapTransaction(standardTx)
	if x.Serialize {
		ser, err := proto.Marshal(tx)
		if err != nil {
			return err
		}
		fmt.Println(hex.EncodeToString(ser))
	} else {
		out, err := json.MarshalIndent(tx, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}
	return nil
}

type WalletLock struct {
	opts *options
}

func (x *WalletLock) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.WalletLock(makeContext(x.opts.AuthToken), &pb.WalletLockRequest{})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type WalletUnlock struct {
	Passphrase string `short:"p" long:"passphrase" description:"The wallet passphrase"`
	Duration   uint32 `short:"d" long:"duration" description:"The number of seconds to unlock the wallet for"`
	opts       *options
}

func (x *WalletUnlock) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.WalletUnlock(makeContext(x.opts.AuthToken), &pb.WalletUnlockRequest{
		Passphrase: x.Passphrase,
		Duration:   x.Duration,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type SetWalletPassphrase struct {
	Passphrase string `short:"p" long:"passphrase" description:"The passphrase to set"`
	opts       *options
}

func (x *SetWalletPassphrase) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.SetWalletPassphrase(makeContext(x.opts.AuthToken), &pb.SetWalletPassphraseRequest{
		Passphrase: x.Passphrase,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type ChangeWalletPassphrase struct {
	Passphrase    string `short:"p" long:"passphrase" description:"The wallet's current passphrase"`
	NewPassphrase string `short:"n" long:"newpassphrase" description:"The passphrase to change it to"`
	opts          *options
}

func (x *ChangeWalletPassphrase) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.ChangeWalletPassphrase(makeContext(x.opts.AuthToken), &pb.ChangeWalletPassphraseRequest{
		CurrentPassphrase: x.Passphrase,
		NewPassphrase:     x.NewPassphrase,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type DeletePrivateKeys struct {
	opts *options
}

func (x *DeletePrivateKeys) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.DeletePrivateKeys(makeContext(x.opts.AuthToken), &pb.DeletePrivateKeysRequest{})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type CreateRawTransaction struct {
	InputCommitments   []string `short:"t" long:"commitment" description:"A commitment to spend as an input. Serialized as a hex string. If using this the wallet will look up the private input data. Use this or input."`
	PrivateInputs      []string `short:"i" long:"input" description:"Private input data as a JSON string. To include more than one input use this option more than once. Use this or commitment."`
	PrivateOutputs     []string `short:"o" long:"output" description:"Private output data as a JSON string. To include more than one output use this option more than once."`
	AppendChangeOutput bool     `short:"c" long:"appendchange" description:"Append a change output to the transaction. If false you'll have to manually include the change out. If true the wallet will use its most recent address for change.'"`
	FeePerKB           uint64   `short:"f" long:"feeperkb" description:"The fee per kilobyte to pay for this transaction. If zero the wallet will use its default fee."`
	Serialize          bool     `short:"s" long:"serialize" description:"Serialize the output as a hex string. If false it will be JSON."`
	opts               *options
}

func (x *CreateRawTransaction) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	req := &pb.CreateRawTransactionRequest{
		Inputs:             nil,
		Outputs:            nil,
		AppendChangeOutput: x.AppendChangeOutput,
		FeePerKilobyte:     x.FeePerKB,
	}

	if len(x.PrivateInputs) > 0 {
		for _, in := range x.PrivateInputs {
			var input pb.PrivateInput
			if err := json.Unmarshal([]byte(in), &input); err != nil {
				return err
			}
			req.Inputs = append(req.Inputs, &pb.CreateRawTransactionRequest_Input{
				CommitmentOrPrivateInput: &pb.CreateRawTransactionRequest_Input_Input{
					Input: &input,
				},
			})
		}
	} else if len(x.InputCommitments) > 0 {
		for _, commitment := range x.InputCommitments {
			commitmentBytes, err := hex.DecodeString(commitment)
			if err != nil {
				return err
			}
			req.Inputs = append(req.Inputs, &pb.CreateRawTransactionRequest_Input{
				CommitmentOrPrivateInput: &pb.CreateRawTransactionRequest_Input_Commitment{
					Commitment: commitmentBytes,
				},
			})
		}
	} else {
		return errors.New("use either input or commitment")
	}

	for _, out := range x.PrivateOutputs {
		output := struct {
			Address string `json:"address"`
			Amount  uint64 `json:"amount"`
			State   string `json:"state"`
		}{}
		if err := json.Unmarshal([]byte(out), &output); err != nil {
			return err
		}
		var state []byte
		if output.State != "" {
			state, err = hex.DecodeString(output.State)
			if err != nil {
				return err
			}
		}
		req.Outputs = append(req.Outputs, &pb.CreateRawTransactionRequest_Output{
			Address: output.Address,
			Amount:  output.Amount,
			State:   state,
		})
	}

	resp, err := client.CreateRawTransaction(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}
	if x.Serialize {
		ser, err := proto.Marshal(resp.RawTx)
		if err != nil {
			return err
		}
		fmt.Println(hex.EncodeToString(ser))
	} else {
		out, err := json.MarshalIndent(resp.RawTx, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}

	return nil
}

type CreateRawStakeTransaction struct {
	InputCommitment string `short:"t" long:"commitment" description:"A commitment to stake as an input. Serialized as a hex string. If using this the wallet will look up the private input data. Use this or input."`
	PrivateInput    string `short:"i" long:"input" description:"Private input data as a JSON string. Use this or commitment."`
	Serialize       bool   `short:"s" long:"serialize" description:"Serialize the output as a hex string. If false it will be JSON."`
	opts            *options
}

func (x *CreateRawStakeTransaction) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}
	req := &pb.CreateRawStakeTransactionRequest{
		Input: nil,
	}

	if len(x.PrivateInput) > 0 {
		var input pb.PrivateInput
		if err := json.Unmarshal([]byte(x.PrivateInput), &input); err != nil {
			return err
		}
		req.Input = &pb.CreateRawStakeTransactionRequest_Input{
			CommitmentOrPrivateInput: &pb.CreateRawStakeTransactionRequest_Input_Input{
				Input: &input,
			},
		}
	} else if len(x.InputCommitment) > 0 {
		commitmentBytes, err := hex.DecodeString(x.InputCommitment)
		if err != nil {
			return err
		}
		req.Input = &pb.CreateRawStakeTransactionRequest_Input{
			CommitmentOrPrivateInput: &pb.CreateRawStakeTransactionRequest_Input_Commitment{
				Commitment: commitmentBytes,
			},
		}
	} else {
		return errors.New("use either input or commitment")
	}

	resp, err := client.CreateRawStakeTransaction(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}
	if x.Serialize {
		ser, err := proto.Marshal(resp.RawTx)
		if err != nil {
			return err
		}
		fmt.Println(hex.EncodeToString(ser))
	} else {
		out, err := json.MarshalIndent(resp.RawTx, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}

	return nil
}

type ProveRawTransaction struct {
	Tx          string   `short:"t" long:"rawtx" description:"The transaction to prove. Serialized as hex string or JSON."`
	Serialize   bool     `short:"s" long:"serialize" description:"Serialize the output as a hex string. If false it will be JSON."`
	PrivateKeys []string `short:"k" long:"privkey" description:"An optional spend private to sign the inputs. If one is not provided this CLI will connect to the wallet and look for the key. Serialized as hex string."`
	opts        *options
}

func (x *ProveRawTransaction) Execute(args []string) error {

	var privKeys []crypto.PrivKey
	for _, k := range x.PrivateKeys {
		privKeyBytes, err := hex.DecodeString(k)
		if err != nil {
			return err
		}
		privKey, err := crypto.UnmarshalPrivateKey(privKeyBytes)
		if err != nil {
			return err
		}
		privKeys = append(privKeys, privKey)
	}

	var rawTx pb.RawTransaction
	txBytes, err := hex.DecodeString(x.Tx)
	if err == nil {
		if err := proto.Unmarshal(txBytes, &rawTx); err != nil {
			return err
		}
	} else {
		if err := json.Unmarshal([]byte(x.Tx), &rawTx); err != nil {
			return err
		}
	}

	hasUnlockingParams := false
	for _, i := range rawTx.Inputs {
		if len(i.UnlockingParams) > 0 {
			hasUnlockingParams = true
			break
		}
	}

	var tx *transactions.Transaction
	if privKeys != nil || hasUnlockingParams {
		tx, err = proveRawTransactionLocally(&rawTx, privKeys)
		if err != nil {
			return err
		}
	} else {
		client, err := makeWalletClient(x.opts)
		if err != nil {
			return err
		}

		resp, err := client.ProveRawTransaction(makeContext(x.opts.AuthToken), &pb.ProveRawTransactionRequest{
			RawTx: &rawTx,
		})
		if err != nil {
			return err
		}
		tx = resp.ProvedTx
	}

	if x.Serialize {
		ser, err := proto.Marshal(tx)
		if err != nil {
			return err
		}
		fmt.Println(hex.EncodeToString(ser))
	} else {
		out, err := json.MarshalIndent(tx, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}
	return nil
}

type Stake struct {
	Commitments []string `short:"c" long:"commitment" description:"A utxo commitment to stake. Encoded as a hex string. You can stake more than one. To do so just use this option more than once."`
	opts        *options
}

func (x *Stake) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	commitments := make([][]byte, 0, len(x.Commitments))
	for _, c := range x.Commitments {
		cBytes, err := hex.DecodeString(c)
		if err != nil {
			return err
		}
		commitments = append(commitments, cBytes)
	}
	if len(commitments) == 0 {
		return errors.New("commitment to stake must be specified")
	}

	_, err = client.Stake(makeContext(x.opts.AuthToken), &pb.StakeRequest{
		Commitments: commitments,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type SetAutoStakeRewards struct {
	Autostake bool `short:"a" long:"autostake" description:"Whether to turn on or off autostaking of rewards"`
	opts      *options
}

func (x *SetAutoStakeRewards) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	_, err = client.SetAutoStakeRewards(makeContext(x.opts.AuthToken), &pb.SetAutoStakeRewardsRequest{
		Autostake: x.Autostake,
	})
	if err != nil {
		return err
	}

	fmt.Println("success")
	return nil
}

type Spend struct {
	Address     string   `short:"a" long:"addr" description:"An address to send coins to"`
	Amount      uint64   `short:"t" long:"amount" description:"The amount to send"`
	FeePerKB    uint64   `short:"f" long:"feeperkb" description:"The fee per kilobyte to pay for this transaction. If zero the wallet will use its default fee."`
	Commitments []string `short:"c" long:"commitment" description:"Optionally specify which input commitment(s) to spend. If this field is omitted the wallet will automatically select (only non-staked) inputs commitments. Serialized as hex strings. Use this option more than once to add more than one input commitment."`
	SpendAll    bool     `long:"all" description:"If true the amount option will be ignored and all the funds will be swept from the wallet to the provided address, minus the transaction fee."`
	opts        *options
}

func (x *Spend) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	commitments := make([][]byte, 0, len(x.Commitments))
	for _, c := range x.Commitments {
		cBytes, err := hex.DecodeString(c)
		if err != nil {
			return err
		}
		commitments = append(commitments, cBytes)
	}

	if x.SpendAll {
		resp, err := client.SweepWallet(makeContext(x.opts.AuthToken), &pb.SweepWalletRequest{
			ToAddress:      x.Address,
			FeePerKilobyte: x.FeePerKB,
		})
		if err != nil {
			return err
		}

		fmt.Println(hex.EncodeToString(resp.Transaction_ID))
	} else {
		resp, err := client.Spend(makeContext(x.opts.AuthToken), &pb.SpendRequest{
			ToAddress:        x.Address,
			Amount:           x.Amount,
			FeePerKilobyte:   x.FeePerKB,
			InputCommitments: commitments,
		})
		if err != nil {
			return err
		}

		fmt.Println(hex.EncodeToString(resp.Transaction_ID))
	}

	return nil
}

type TimelockCoins struct {
	LockUntil   int64    `short:"l" long:"lockuntil" description:"A unix timestamp to lock the coins until (in seconds)."`
	Amount      uint64   `short:"t" long:"amount" description:"The amount to lockup"`
	FeePerKB    uint64   `short:"f" long:"feeperkb" description:"The fee per kilobyte to pay for this transaction. If zero the wallet will use its default fee."`
	Commitments []string `short:"c" long:"commitment" description:"Optionally specify which input commitment(s) to lock. If this field is omitted the wallet will automatically select (only non-staked) inputs commitments. Serialized as hex strings. Use this option more than once to add more than one input commitment."`
	opts        *options
}

func (x *TimelockCoins) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	commitments := make([][]byte, 0, len(x.Commitments))
	for _, c := range x.Commitments {
		cBytes, err := hex.DecodeString(c)
		if err != nil {
			return err
		}
		commitments = append(commitments, cBytes)
	}

	resp, err := client.TimelockCoins(makeContext(x.opts.AuthToken), &pb.TimelockCoinsRequest{
		LockUntil:        x.LockUntil,
		Amount:           x.Amount,
		FeePerKilobyte:   x.FeePerKB,
		InputCommitments: commitments,
	})
	if err != nil {
		return err
	}

	fmt.Println(hex.EncodeToString(resp.Transaction_ID))

	return nil
}

func proveRawTransactionLocally(rawTx *pb.RawTransaction, privKeys []crypto.PrivKey) (*transactions.Transaction, error) {
	if rawTx == nil {
		return nil, errors.New("raw tx is nil")
	}
	if rawTx.Tx == nil {
		return nil, errors.New("tx is nil")
	}

	if rawTx.Tx.GetStandardTransaction() != nil {
		standardTx := rawTx.Tx.GetStandardTransaction()
		sigHash, err := standardTx.SigHash()
		if err != nil {
			return nil, err
		}

		// Create the transaction zk proof
		privateParams := &standard.PrivateParams{
			Inputs:  []standard.PrivateInput{},
			Outputs: []standard.PrivateOutput{},
		}

		for i, in := range rawTx.Inputs {
			if in.UnlockingParams == "" {
				var privKey crypto.PrivKey
				for _, k := range privKeys {
					novaPub, ok := k.GetPublic().(*icrypto.NovaPublicKey)
					if !ok {
						return nil, errors.New("key is not type Nova")
					}
					x, y := novaPub.ToXY()
					if len(in.ScriptParams) == 2 && bytes.Equal(in.ScriptParams[0], x) && bytes.Equal(in.ScriptParams[1], y) {
						privKey = k
						break
					}
				}
				if privKey == nil {
					return nil, fmt.Errorf("private key for input %d not found", i)
				}

				sig, err := privKey.Sign(sigHash)
				if err != nil {
					return nil, err
				}

				in.UnlockingParams = fmt.Sprintf("(cons 0x%x 0x%x)", sig[:32], sig[32:])
			}
			privIn := standard.PrivateInput{
				SpendNote: types.SpendNote{
					Amount: types.Amount(in.Amount),
				},
				CommitmentIndex: in.TxoProof.Index,
				InclusionProof: standard.InclusionProof{
					Hashes: in.TxoProof.Hashes,
					Flags:  in.TxoProof.Flags,
				},
				ScriptCommitment: in.ScriptCommitment,
				ScriptParams:     in.ScriptParams,
				UnlockingParams:  []byte(in.UnlockingParams),
			}
			copy(privIn.Salt[:], in.Salt)
			copy(privIn.AssetID[:], in.Asset_ID)
			state := new(types.State)
			if err := state.Deserialize(in.State); err != nil {
				return nil, err
			}
			privIn.State = *state

			privateParams.Inputs = append(privateParams.Inputs, privIn)
		}

		for _, out := range rawTx.Outputs {
			privOut := standard.PrivateOutput{
				SpendNote: types.SpendNote{
					ScriptHash: types.ID{},
					Amount:     types.Amount(out.Amount),
				},
			}
			copy(privOut.ScriptHash[:], out.ScriptHash)
			copy(privOut.Salt[:], out.Salt)
			copy(privOut.AssetID[:], out.Asset_ID)
			state := new(types.State)
			if err := state.Deserialize(out.State); err != nil {
				return nil, err
			}
			privOut.State = *state
			privateParams.Outputs = append(privateParams.Outputs, privOut)
		}

		publicParams := &standard.PublicParams{
			TXORoot:    standardTx.TxoRoot,
			SigHash:    sigHash,
			Nullifiers: standardTx.Nullifiers,
			Fee:        standardTx.Fee,
		}

		for _, out := range standardTx.Outputs {
			publicParams.Outputs = append(publicParams.Outputs, standard.PublicOutput{
				Commitment: out.Commitment,
				CipherText: out.Ciphertext,
			})
		}

		proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicParams)
		if err != nil {
			return nil, err
		}

		standardTx.Proof = proof

		return transactions.WrapTransaction(standardTx), nil
	} else if rawTx.Tx.GetStakeTransaction() != nil {
		stakeTx := rawTx.Tx.GetStakeTransaction()
		sigHash, err := stakeTx.SigHash()
		if err != nil {
			return nil, err
		}

		if len(rawTx.Inputs) == 0 {
			return nil, errors.New("no inputs")
		}

		if rawTx.Inputs[0].UnlockingParams == "" {
			var privKey crypto.PrivKey
			for _, k := range privKeys {
				novaPub, ok := k.GetPublic().(*icrypto.NovaPublicKey)
				if !ok {
					return nil, errors.New("key is not type Nova")
				}
				x, y := novaPub.ToXY()
				if len(rawTx.Inputs[0].ScriptParams) == 2 && bytes.Equal(rawTx.Inputs[0].ScriptParams[0], x) && bytes.Equal(rawTx.Inputs[0].ScriptParams[1], y) {
					privKey = k
					break
				}
			}
			if privKey == nil {
				return nil, errors.New("private key for input not found")
			}

			sig, err := privKey.Sign(sigHash)
			if err != nil {
				return nil, err
			}

			rawTx.Inputs[0].UnlockingParams = fmt.Sprintf("(cons 0x%x 0x%x)", sig[:32], sig[32:])
		}

		// Create the transaction zk proof
		privateParams := &stake.PrivateParams{
			CommitmentIndex: rawTx.Inputs[0].TxoProof.Index,
			InclusionProof: standard.InclusionProof{
				Hashes: rawTx.Inputs[0].TxoProof.Hashes,
				Flags:  rawTx.Inputs[0].TxoProof.Flags,
			},
			ScriptCommitment: rawTx.Inputs[0].ScriptCommitment,
			ScriptParams:     rawTx.Inputs[0].ScriptParams,
			UnlockingParams:  []byte(rawTx.Inputs[0].UnlockingParams),
		}
		copy(privateParams.Salt[:], rawTx.Inputs[0].Salt)
		copy(privateParams.AssetID[:], rawTx.Inputs[0].Asset_ID)
		state := new(types.State)
		if err := state.Deserialize(rawTx.Inputs[0].State); err != nil {
			return nil, err
		}
		privateParams.State = *state

		publicParams := &stake.PublicParams{
			TXORoot:   stakeTx.TxoRoot,
			SigHash:   sigHash,
			Amount:    stakeTx.Amount,
			Nullifier: stakeTx.Nullifier,
		}

		proof, err := zk.CreateSnark(stake.StakeCircuit, privateParams, publicParams)
		if err != nil {
			return nil, err
		}

		stakeTx.Proof = proof

		return transactions.WrapTransaction(stakeTx), nil
	}
	return nil, errors.New("tx must be either standard or stake type")
}

func pbIOtoIO(ios []*pb.WalletTransaction_IO) []interface{} {
	ret := make([]interface{}, 0, len(ios))
	type txIO struct {
		Address string `json:"address"`
		Amount  uint64 `json:"amount"`
	}
	for _, io := range ios {
		if io.GetTxIo() != nil {
			ret = append(ret, &txIO{
				Address: io.GetTxIo().Address,
				Amount:  io.GetTxIo().Amount,
			})
		}
		if io.GetUnknown() != nil {
			ret = append(ret, walletlib.Unknown{})
		}
	}
	return ret
}
