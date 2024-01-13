// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	icrypto "github.com/project-illium/ilxd/crypto"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circparams"
	"time"
)

func (h *TestHarness) generateBlocks(nBlocks int) ([]*blocks.Block, map[types.Nullifier]*SpendableNote, error) {
	newBlocks := make([]*blocks.Block, 0, nBlocks)
	acc := h.acc.Clone()
	fee := uint64(1)
	nCommitments := acc.NumElements()
	bestID, bestHeight, _ := h.chain.BestBlock()

	remainingNotes := make(map[types.Nullifier]*SpendableNote)
	for k, v := range h.spendableNotes {
		remainingNotes[k] = v
	}

	for n := 0; n < nBlocks; n++ {
		outputsPerTx := h.txsPerBlock
		numTxs := h.txsPerBlock
		if len(remainingNotes) < h.txsPerBlock {
			outputsPerTx = h.txsPerBlock / len(remainingNotes)
			numTxs = len(remainingNotes)
		}

		notes := make([]*SpendableNote, 0, len(remainingNotes))
		nullifiers := make([]types.Nullifier, 0, len(remainingNotes))
		for nullifier, note := range remainingNotes {
			notes = append(notes, note)
			nullifiers = append(nullifiers, nullifier)
		}

		toDelete := make([]types.Nullifier, 0, len(remainingNotes))
		txs := make([]*transactions.Transaction, 0, len(remainingNotes))
		for i := 0; i < numTxs; i++ {
			sn := notes[i]
			inNullifier := nullifiers[i]
			commitment, err := sn.Note.Commitment()
			if err != nil {
				return nil, nil, err
			}
			inclusionProof, err := acc.GetProof(commitment[:])
			if err != nil {
				return nil, nil, err
			}

			toDelete = append(toDelete, inNullifier)

			var (
				outputs     = make([]*transactions.Output, 0, outputsPerTx)
				outputNotes = make([]*SpendableNote, 0, outputsPerTx)
			)

			for x := 0; x < outputsPerTx; x++ {
				nCommitments++
				privKey, pubKey, err := icrypto.GenerateNovaKey(rand.Reader)
				if err != nil {
					return nil, nil, err
				}
				pubx, puby := pubKey.(*icrypto.NovaPublicKey).ToXY()

				salt, err := types.RandomSalt()
				if err != nil {
					return nil, nil, err
				}

				lockingScript := &types.LockingScript{
					ScriptCommitment: types.NewID(zk.BasicTransferScriptCommitment()),
					LockingParams:    [][]byte{pubx, puby},
				}
				scriptHash, err := lockingScript.Hash()
				if err != nil {
					return nil, nil, err
				}
				outputNote := &types.SpendNote{
					ScriptHash: scriptHash,
					Amount:     (sn.Note.Amount / types.Amount(outputsPerTx)) - types.Amount(fee),
					AssetID:    types.IlliumCoinID,
					Salt:       salt,
					State:      types.State{},
				}
				outputNotes = append(outputNotes, &SpendableNote{
					Note:             outputNote,
					PrivateKey:       privKey,
					LockingScript:    lockingScript,
					cachedScriptHash: scriptHash,
				})

				outputCommitment, err := outputNote.Commitment()
				if err != nil {
					return nil, nil, err
				}

				outNullifier, err := types.CalculateNullifier(nCommitments-1, outputNote.Salt, lockingScript.ScriptCommitment.Bytes(), lockingScript.LockingParams...)
				if err != nil {
					return nil, nil, err
				}

				remainingNotes[outNullifier] = &SpendableNote{
					Note:          outputNote,
					LockingScript: lockingScript,
					PrivateKey:    privKey,
				}

				outputs = append(outputs, &transactions.Output{
					Commitment: outputCommitment[:],
					Ciphertext: make([]byte, blockchain.CiphertextLen),
				})
			}
			standardTx := &transactions.StandardTransaction{
				Outputs:    outputs,
				Fee:        1,
				Nullifiers: [][]byte{inNullifier.Bytes()},
				TxoRoot:    acc.Root().Bytes(),
				Proof:      nil,
			}

			sigHash, err := standardTx.SigHash()
			if err != nil {
				return nil, nil, err
			}

			sig, err := sn.PrivateKey.Sign(sigHash)
			if err != nil {
				return nil, nil, err
			}

			sigRx, sigRy, sigS := icrypto.UnmarshalSignature(sig)

			privateParams := &circparams.PrivateParams{
				Inputs: []circparams.PrivateInput{
					{
						Amount:          sn.Note.Amount,
						Salt:            sn.Note.Salt,
						AssetID:         sn.Note.AssetID,
						State:           types.State{},
						CommitmentIndex: inclusionProof.Index,
						InclusionProof: circparams.InclusionProof{
							Hashes: inclusionProof.Hashes,
							Flags:  inclusionProof.Flags,
						},
						Script:          zk.BasicTransferScript(),
						LockingParams:   sn.LockingScript.LockingParams,
						UnlockingParams: fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS),
					},
				},
			}
			for _, outNote := range outputNotes {
				privateParams.Outputs = append(privateParams.Outputs, circparams.PrivateOutput{
					State:      types.State{},
					Amount:     outNote.Note.Amount,
					Salt:       outNote.Note.Salt,
					AssetID:    outNote.Note.AssetID,
					ScriptHash: outNote.cachedScriptHash,
				})
			}

			publicOutputs := make([]circparams.PublicOutput, len(outputNotes))
			for i, output := range outputs {
				publicOutputs[i] = circparams.PublicOutput{
					Commitment: types.NewID(output.Commitment),
					CipherText: output.Ciphertext,
				}
			}

			publicPrams := &circparams.PublicParams{
				TXORoot:    acc.Root(),
				SigHash:    types.NewID(sigHash),
				Outputs:    publicOutputs,
				Nullifiers: []types.Nullifier{inNullifier},
				Fee:        types.Amount(fee),
				Locktime:   time.Unix(0, 0),
			}

			proof, err := h.prover.Prove(zk.StandardValidationProgram(), privateParams, publicPrams)
			if err != nil {
				return nil, nil, err
			}
			standardTx.Proof = proof
			txs = append(txs, transactions.WrapTransaction(standardTx))
		}

		merkleRoot := blockchain.TransactionsMerkleRoot(txs)

		h.timeSource++

		var (
			networkKey crypto.PrivKey
			validator  peer.ID
		)
		for k, v := range h.validators {
			networkKey = v.networkKey
			validator = k
		}
		valBytes, err := validator.Marshal()
		if err != nil {
			return nil, nil, err
		}

		blk := &blocks.Block{
			Header: &blocks.BlockHeader{
				Version:     1,
				Height:      bestHeight + 1,
				Parent:      bestID.Bytes(),
				Timestamp:   h.timeSource,
				TxRoot:      merkleRoot[:],
				Producer_ID: valBytes,
				Signature:   nil,
			},
			Transactions: txs,
		}

		sigHash, err := blk.Header.SigHash()
		if err != nil {
			return nil, nil, err
		}
		sig, err := networkKey.Sign(sigHash)
		if err != nil {
			return nil, nil, err
		}
		blk.Header.Signature = sig

		newBlocks = append(newBlocks, blk)
		bestHeight++
		bestID = blk.ID()

		for _, out := range blk.Outputs() {
			acc.Insert(out.Commitment, true)
		}

		for _, del := range toDelete {
			delete(remainingNotes, del)
		}
		//fmt.Println(blk.Header.Height)
	}
	return newBlocks, remainingNotes, nil
}

func (h *TestHarness) generateBlockWithTransactions(txs []*transactions.Transaction) (*blocks.Block, error) {
	bestID, bestHeight, _ := h.chain.BestBlock()
	merkleRoot := blockchain.TransactionsMerkleRoot(txs)

	h.timeSource++

	var (
		networkKey crypto.PrivKey
		validator  peer.ID
	)
	for k, v := range h.validators {
		networkKey = v.networkKey
		validator = k
	}
	valBytes, err := validator.Marshal()
	if err != nil {
		return nil, err
	}

	blk := &blocks.Block{
		Header: &blocks.BlockHeader{
			Version:     1,
			Height:      bestHeight + 1,
			Parent:      bestID.Bytes(),
			Timestamp:   h.timeSource,
			TxRoot:      merkleRoot[:],
			Producer_ID: valBytes,
			Signature:   nil,
		},
		Transactions: txs,
	}

	sigHash, err := blk.Header.SigHash()
	if err != nil {
		return nil, err
	}
	sig, err := networkKey.Sign(sigHash)
	if err != nil {
		return nil, err
	}
	blk.Header.Signature = sig

	for _, n := range blk.Nullifiers() {
		delete(h.spendableNotes, n)
	}
	return blk, nil
}

func createGenesisBlock(params *params.NetworkParams, networkKey, spendKey crypto.PrivKey,
	initialCoins uint64, additionalOutputs []*transactions.Output, prover zk.Prover) (*blocks.Block, *SpendableNote, error) {

	// First we'll create the spend note for the coinbase transaction.
	// The initial coins will be generated to the spendKey.
	salt1, err := types.RandomSalt()
	if err != nil {
		return nil, nil, err
	}

	pubx, puby := spendKey.GetPublic().(*icrypto.NovaPublicKey).ToXY()

	note1LockingScript := &types.LockingScript{
		ScriptCommitment: types.NewID(zk.BasicTransferScriptCommitment()),
		LockingParams:    [][]byte{pubx, puby},
	}
	note1ScriptHash, err := note1LockingScript.Hash()
	if err != nil {
		return nil, nil, err
	}
	note1 := &types.SpendNote{
		ScriptHash: note1ScriptHash,
		Amount:     types.Amount(initialCoins) / 2,
		AssetID:    types.IlliumCoinID,
		Salt:       salt1,
		State:      types.State{},
	}

	salt2, err := types.RandomSalt()
	if err != nil {
		return nil, nil, err
	}

	note2LockingScript := &types.LockingScript{
		ScriptCommitment: types.NewID(zk.BasicTransferScriptCommitment()),
		LockingParams:    [][]byte{pubx, puby},
	}
	note2ScriptHash, err := note2LockingScript.Hash()
	if err != nil {
		return nil, nil, err
	}
	note2 := &types.SpendNote{
		ScriptHash: note2ScriptHash,
		Amount:     types.Amount(initialCoins) / 2,
		AssetID:    types.IlliumCoinID,
		Salt:       salt2,
		State:      types.State{},
	}

	// Next we're going to start building the coinbase transaction
	commitment1, err := note1.Commitment()
	if err != nil {
		return nil, nil, err
	}
	commitment2, err := note2.Commitment()
	if err != nil {
		return nil, nil, err
	}
	validatorID, err := peer.IDFromPublicKey(networkKey.GetPublic())
	if err != nil {
		return nil, nil, err
	}
	idBytes, err := validatorID.Marshal()
	if err != nil {
		return nil, nil, err
	}

	coinbaseTx := &transactions.CoinbaseTransaction{
		Validator_ID: idBytes,
		NewCoins:     initialCoins,
		Outputs: []*transactions.Output{
			{
				Commitment: commitment1[:],
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
			{
				Commitment: commitment2[:],
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
	}
	coinbaseTx.Outputs = append(coinbaseTx.Outputs, additionalOutputs...)

	// And now sign the coinbase transaction with the network key
	sigHash, err := coinbaseTx.SigHash()
	if err != nil {
		return nil, nil, err
	}

	sig, err := networkKey.Sign(sigHash)
	if err != nil {
		return nil, nil, err
	}
	coinbaseTx.Signature = sig

	// Finally we're going to create the zk-snark proof for the coinbase
	// transaction.

	nullifier1, err := types.CalculateNullifier(0, salt1, note1LockingScript.ScriptCommitment.Bytes(), note1LockingScript.LockingParams...)
	if err != nil {
		return nil, nil, err
	}

	publicParams := &circparams.PublicParams{
		Outputs: []circparams.PublicOutput{
			{
				Commitment: commitment1,
			},
			{
				Commitment: commitment2,
			},
		},
		Coinbase: types.Amount(initialCoins),
	}
	privateParams := &circparams.PrivateParams{
		Outputs: []circparams.PrivateOutput{
			{
				ScriptHash: note1ScriptHash,
				Amount:     types.Amount(initialCoins / 2),
				Salt:       note1.Salt,
				AssetID:    note1.AssetID,
				State:      note1.State,
			},
			{
				ScriptHash: note2ScriptHash,
				Amount:     types.Amount(initialCoins / 2),
				Salt:       note2.Salt,
				AssetID:    note2.AssetID,
				State:      note2.State,
			},
		},
	}

	proof, err := prover.Prove(zk.CoinbaseValidationProgram(), privateParams, publicParams)
	if err != nil {
		return nil, nil, err
	}
	coinbaseTx.Proof = proof

	// Next we have to build the transaction staking the coins generated
	// in the prior coinbase transaction. This is needed because if no
	// validators are set in the genesis block we can't move the chain
	// forward.
	//
	// Notice there is a special validation rule for the genesis block
	// that doesn't apply to any other block. Normally, transactions
	// must contain a txoRoot for a block already in the chain. However,
	// in the case of the genesis block there are no other blocks in the
	// chain yet. So the rules allow the genesis block to reference its
	// own txoRoot.
	acc := blockchain.NewAccumulator()
	for i, output := range coinbaseTx.Outputs {
		acc.Insert(output.Commitment, i == 0)
	}
	txoRoot := acc.Root()
	inclusionProof, err := acc.GetProof(commitment1[:])
	if err != nil {
		return nil, nil, err
	}

	stakeTx := &transactions.StakeTransaction{
		Validator_ID: idBytes,
		Amount:       initialCoins,
		Nullifier:    nullifier1.Bytes(),
		TxoRoot:      txoRoot.Bytes(), // See note above
	}

	// Sign the stake transaction
	sigHash2, err := stakeTx.SigHash()
	if err != nil {
		return nil, nil, err
	}

	sig2, err := networkKey.Sign(sigHash2)
	if err != nil {
		return nil, nil, err
	}
	stakeTx.Signature = sig2

	// And generate the zk-snark proof
	sig3, err := spendKey.Sign(sigHash2)
	if err != nil {
		return nil, nil, err
	}

	sigRx, sigRy, sigS := icrypto.UnmarshalSignature(sig3)

	publicParams2 := &circparams.StakePublicParams{
		StakeAmount: types.Amount(initialCoins / 2),
		PublicParams: circparams.PublicParams{
			SigHash:    types.NewID(sigHash2),
			Nullifiers: []types.Nullifier{nullifier1},
			TXORoot:    txoRoot,
			Locktime:   time.Unix(0, 0),
		},
	}
	privateParams2 := &circparams.PrivateInput{
		Amount:          types.Amount(initialCoins / 2),
		AssetID:         types.IlliumCoinID,
		Salt:            salt1,
		State:           types.State{},
		CommitmentIndex: 0,
		InclusionProof: circparams.InclusionProof{
			Hashes: inclusionProof.Hashes,
			Flags:  inclusionProof.Flags,
		},
		Script:          zk.BasicTransferScript(),
		LockingParams:   [][]byte{pubx, puby},
		UnlockingParams: fmt.Sprintf("(cons 0x%x (cons 0x%x (cons 0x%x nil)))", sigRx, sigRy, sigS),
	}

	proof, err = prover.Prove(zk.StakeValidationProgram(), privateParams2, publicParams2)
	if err != nil {
		return nil, nil, err
	}
	stakeTx.Proof = proof

	// Now we add the transactions to the genesis block
	genesis := params.GenesisBlock
	genesis.Transactions = []*transactions.Transaction{
		transactions.WrapTransaction(coinbaseTx),
		transactions.WrapTransaction(stakeTx),
	}

	// And create the genesis merkle root
	merkleRoot := blockchain.TransactionsMerkleRoot(genesis.Transactions)
	genesis.Header.TxRoot = merkleRoot[:]
	genesis.Header.Timestamp = time.Now().Add(-time.Hour * 24 * 365 * 10).Unix()

	spendableNote := &SpendableNote{
		Note:          note2,
		LockingScript: note2LockingScript,
		PrivateKey:    spendKey,
	}
	return genesis, spendableNote, nil
}
