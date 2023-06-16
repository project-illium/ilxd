// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/project-illium/ilxd/rpc/pb"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/standard"
	"github.com/project-illium/walletlib"
	"google.golang.org/protobuf/proto"
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
	}
	txs := make([]tx, 0, len(resp.Txs))
	for _, rtx := range resp.Txs {
		txs = append(txs, tx{
			Txid:     rtx.Transaction_ID,
			NetCoins: rtx.NetCoins,
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
		Address    string             `json:"address"`
		Commitment types.HexEncodable `json:"commitment"`
		Amount     uint64             `json:"amount"`
		WatchOnly  bool               `json:"watchOnly"`
		Staked     bool               `json:"staked"`
	}
	utxos := make([]utxo, 0, len(resp.Utxos))
	for _, ut := range resp.Utxos {
		utxos = append(utxos, utxo{
			Address:    ut.Address,
			Commitment: ut.Commitment,
			Amount:     ut.Amount,
			WatchOnly:  ut.WatchOnly,
			Staked:     ut.Staked,
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
	UnlockingScript  string `short:"u" long:"unlockingscript" description:"The unlocking script for the address. Serialized as hex string"`
	ViewPrivateKey   string `short:"k" long:"viewkey" description:"The view private key for the address. Serialized as hex string."`
	Rescan           bool   `short:"r" long:"rescan" description:"Whether or not to rescan the blockchain to try to detect transactions for this address."`
	RescanFromHeight uint32 `short:"h" long:"rescanheight" description:"The height of the chain to rescan from. Selecting a height close to the address birthday saves resources."`
	opts             *options
}

func (x *ImportAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	unlockingScriptBytes, err := hex.DecodeString(x.UnlockingScript)
	if err != nil {
		return err
	}
	privKeyBytes, err := hex.DecodeString(x.ViewPrivateKey)
	if err != nil {
		return err
	}

	_, err = client.ImportAddress(makeContext(x.opts.AuthToken), &pb.ImportAddressRequest{
		Address:          x.Address,
		UnlockingScript:  unlockingScriptBytes,
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
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.CreateMultisigSpendKeypair(makeContext(x.opts.AuthToken), &pb.CreateMultisigSpendKeypairRequest{})
	if err != nil {
		return err
	}

	kp := struct {
		PrivateKey types.HexEncodable `json:"privateKey"`
		PublicKey  types.HexEncodable `json:"publicKey"`
	}{
		PrivateKey: resp.Privkey,
		PublicKey:  resp.Pubkey,
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
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	resp, err := client.CreateMultisigViewKeypair(makeContext(x.opts.AuthToken), &pb.CreateMultisigViewKeypairRequest{})
	if err != nil {
		return err
	}

	kp := struct {
		PrivateKey types.HexEncodable `json:"privateKey"`
		PublicKey  types.HexEncodable `json:"publicKey"`
	}{
		PrivateKey: resp.Privkey,
		PublicKey:  resp.Pubkey,
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
	opts       *options
}

func (x *CreateMultisigAddress) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	pubkeys := make([][]byte, 0, len(x.Pubkeys))
	for _, p := range x.Pubkeys {
		keyBytes, err := hex.DecodeString(p)
		if err != nil {
			return err
		}
		pubkeys = append(pubkeys, keyBytes)
	}

	viewKey, err := hex.DecodeString(x.ViewPubKey)
	if err != nil {
		return err
	}

	resp, err := client.CreateMultisigAddress(makeContext(x.opts.AuthToken), &pb.CreateMultisigAddressRequest{
		Pubkeys:    pubkeys,
		Threshold:  x.Threshold,
		ViewPubkey: viewKey,
	})
	if err != nil {
		return err
	}
	fmt.Println(resp.Address)
	return nil
}

type CreateMultiSignature struct {
	Tx         string `short:"t" long:"tx" description:"A transaction to sign. Serialized as hex string. Use this or sighash."`
	SigHash    string `short:"h" long:"sighash" description:"A sighash to sign. Serialized as hex string. Use this or tx."`
	PrivateKey string `short:"k" long:"privkey" description:"A spend private key. Serialized as hex string."`
	opts       *options
}

func (x *CreateMultiSignature) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	req := &pb.CreateMultiSignatureRequest{
		TxOrSighash: nil,
		PrivateKey:  nil,
	}

	if x.Tx != "" {
		txBytes, err := hex.DecodeString(x.Tx)
		if err != nil {
			return err
		}
		var tx transactions.Transaction
		if err := proto.Unmarshal(txBytes, &tx); err != nil {
			return err
		}
		req.TxOrSighash = &pb.CreateMultiSignatureRequest_Tx{
			Tx: &tx,
		}
	} else if x.SigHash != "" {
		sigHash, err := hex.DecodeString(x.SigHash)
		if err != nil {
			return err
		}
		req.TxOrSighash = &pb.CreateMultiSignatureRequest_Sighash{
			Sighash: sigHash,
		}
	} else {
		return errors.New("tx or sighash required")
	}

	resp, err := client.CreateMultiSignature(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}

	fmt.Println(hex.EncodeToString(resp.Signature))
	return nil
}

type ProveMultisig struct {
	Tx         string   `short:"t" long:"tx" description:"The transaction to prove. Serialized as hex string."`
	Signatures []string `short:"s" long:"sig" description:"A signature covering the tranaction's sighash. Use this option more than once to add more signatures.'"`
	opts       *options
}

func (x *ProveMultisig) Execute(args []string) error {
	client, err := makeWalletClient(x.opts)
	if err != nil {
		return err
	}

	txBytes, err := hex.DecodeString(x.Tx)
	if err != nil {
		return err
	}
	var tx pb.RawTransaction
	if err := proto.Unmarshal(txBytes, &tx); err != nil {
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

	resp, err := client.ProveMultisig(makeContext(x.opts.AuthToken), &pb.ProveMultisigRequest{
		Tx:   &tx,
		Sigs: sigs,
	})
	if err != nil {
		return err
	}
	txBytes, err = proto.Marshal(resp.ProvedTx)
	if err != nil {
		return err
	}
	fmt.Println(hex.EncodeToString(txBytes))
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
		}{}
		if err := json.Unmarshal([]byte(out), &output); err != nil {
			return err
		}
		req.Outputs = append(req.Outputs, &pb.CreateRawTransactionRequest_Output{
			Address: output.Address,
			Amount:  output.Amount,
		})
	}

	resp, err := client.CreateRawTransaction(makeContext(x.opts.AuthToken), req)
	if err != nil {
		return err
	}
	if x.Serialize {
		ser, err := proto.Marshal(resp.Tx)
		if err != nil {
			return err
		}
		fmt.Println(hex.EncodeToString(ser))
	} else {
		out, err := json.MarshalIndent(resp.Tx, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}

	return nil
}

type ProveRawTransaction struct {
	Tx        string   `short:"t" long:"tx" description:"The transaction to prove. Serialized as hex string or JSON."`
	Privkeys  []string `short:"k" long:"privkey" description:"The private key to use to sign each input. These must be in order if the transaction inputs. If you need to include more than one use this option more than once."`
	Serialize bool     `short:"s" long:"serialize" description:"Serialize the output as a hex string. If false it will be JSON."`
	opts      *options
}

func (x *ProveRawTransaction) Execute(args []string) error {
	keys := make([]crypto.PrivKey, 0, len(x.Privkeys))
	for _, k := range x.Privkeys {
		keyBytes, err := hex.DecodeString(k)
		if err == nil {
			privKey, err := crypto.UnmarshalPrivateKey(keyBytes)
			if err != nil {
				return err
			}
			keys = append(keys, privKey)
			continue
		}
		privKey, err := walletlib.DecodePrivateKey(k)
		if err != nil {
			return err
		}
		keys = append(keys, privKey)
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
	////////////////////////////
	standardTx := rawTx.Tx.GetStandardTransaction()
	if standardTx == nil {
		return errors.New("standard tx is nil")
	}

	sigHash, err := standardTx.SigHash()
	if err != nil {
		return err
	}

	if len(keys) != len(standardTx.Nullifiers) {
		return errors.New("not enough private keys")
	}

	// Create the transaction zk proof
	privateParams := &standard.PrivateParams{
		Inputs:  make([]standard.PrivateInput, 0, len(rawTx.Inputs)),
		Outputs: make([]standard.PrivateOutput, 0, len(rawTx.Outputs)),
	}

	for i, in := range rawTx.Inputs {
		privKey := keys[i]

		switch k := privKey.(type) {
		case *crypto.Ed25519PrivateKey:
		case *walletlib.WalletPrivateKey:
			privKey = k.SpendKey()
		default:
			return errors.New("unknown private key type")
		}

		sig, err := privKey.Sign(sigHash)
		if err != nil {
			return err
		}
		privIn := standard.PrivateInput{
			Amount:          in.Amount,
			CommitmentIndex: in.TxoProof.Index,
			InclusionProof: standard.InclusionProof{
				Hashes:      in.TxoProof.Hashes,
				Flags:       in.TxoProof.Flags,
				Accumulator: in.TxoProof.Accumulator,
			},
			ScriptCommitment: in.ScriptCommitment,
			ScriptParams:     in.ScriptParams,
			UnlockingParams:  [][]byte{sig},
		}
		copy(privIn.Salt[:], in.Salt)
		copy(privIn.AssetID[:], in.Asset_ID)
		copy(privIn.State[:], in.State)

		privateParams.Inputs = append(privateParams.Inputs, privIn)
	}

	for _, out := range rawTx.Outputs {
		privOut := standard.PrivateOutput{
			ScriptHash: out.ScriptHash,
			Amount:     out.Amount,
		}
		copy(privOut.Salt[:], out.Salt)
		copy(privOut.AssetID[:], out.Asset_ID)
		copy(privOut.State[:], out.State)
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
		return err
	}

	standardTx.Proof = proof

	////////////////////////////
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
	return nil
}
