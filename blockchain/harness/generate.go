// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package harness

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/params"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/project-illium/ilxd/zk/circuits/stake"
	"github.com/project-illium/ilxd/zk/circuits/standard"
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
		for _, note := range remainingNotes {
			notes = append(notes, note)
		}

		toDelete := make([]types.Nullifier, 0, len(remainingNotes))
		txs := make([]*transactions.Transaction, 0, len(remainingNotes))
		for i := 0; i < numTxs; i++ {
			sn := notes[i]

			commitment := sn.Note.Commitment()
			inclusionProof, err := acc.GetProof(commitment[:])
			if err != nil {
				return nil, nil, err
			}

			nullifier := types.CalculateNullifier(inclusionProof.Index, sn.Note.Salt, sn.UnlockingScript.ScriptCommitment, sn.UnlockingScript.ScriptParams...)
			toDelete = append(toDelete, nullifier)

			var (
				outputs     = make([]*transactions.Output, 0, outputsPerTx)
				outputNotes = make([]*SpendableNote, 0, outputsPerTx)
			)

			for x := 0; x < outputsPerTx; x++ {
				nCommitments++
				privKey, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
				if err != nil {
					return nil, nil, err
				}
				pubKeyBytes, err := crypto.MarshalPublicKey(pubKey)
				if err != nil {
					return nil, nil, err
				}

				mockStandardScriptCommitment := make([]byte, 32)
				rand.Read(mockStandardScriptCommitment)

				var salt [types.SaltLen]byte
				rand.Read(salt[:])

				unlockingScript := &types.UnlockingScript{
					ScriptCommitment: mockStandardScriptCommitment,
					ScriptParams:     [][]byte{pubKeyBytes},
				}
				scriptHash := unlockingScript.Hash()
				outputNote := &types.SpendNote{
					ScriptHash: scriptHash[:],
					Amount:     (sn.Note.Amount / types.Amount(outputsPerTx)) - types.Amount(fee),
					AssetID:    types.IlliumCoinID,
					Salt:       salt,
					State:      [types.StateLen]byte{},
				}
				outputNotes = append(outputNotes, &SpendableNote{
					Note:            outputNote,
					PrivateKey:      privKey,
					UnlockingScript: unlockingScript,
				})

				outputCommitment := outputNote.Commitment()

				outNullifier := types.CalculateNullifier(nCommitments-1, outputNote.Salt, unlockingScript.ScriptCommitment, unlockingScript.ScriptParams...)

				remainingNotes[outNullifier] = &SpendableNote{
					Note:            outputNote,
					UnlockingScript: unlockingScript,
					PrivateKey:      privKey,
				}

				outputs = append(outputs, &transactions.Output{
					Commitment: outputCommitment[:],
					Ciphertext: make([]byte, blockchain.CiphertextLen),
				})
			}
			standardTx := &transactions.StandardTransaction{
				Outputs:    outputs,
				Fee:        1,
				Nullifiers: [][]byte{nullifier.Bytes()},
				TxoRoot:    acc.Root().Bytes(),
				Proof:      nil,
			}

			sigHash, err := standardTx.SigHash()
			if err != nil {
				return nil, nil, err
			}

			mockUnlockingSig := make([]byte, 32)
			rand.Read(mockUnlockingSig)

			privateParams := &standard.PrivateParams{
				Inputs: []standard.PrivateInput{
					{
						Amount:          uint64(sn.Note.Amount),
						Salt:            sn.Note.Salt,
						AssetID:         sn.Note.AssetID,
						State:           [types.StateLen]byte{},
						CommitmentIndex: inclusionProof.Index,
						InclusionProof: standard.InclusionProof{
							Hashes:      inclusionProof.Hashes,
							Flags:       inclusionProof.Flags,
							Accumulator: inclusionProof.Accumulator,
						},
						ScriptCommitment: sn.UnlockingScript.ScriptCommitment,
						ScriptParams:     sn.UnlockingScript.ScriptParams,
						UnlockingParams:  [][]byte{mockUnlockingSig},
					},
				},
			}
			for _, outNote := range outputNotes {
				scriptHash := outNote.UnlockingScript.Hash()
				privateParams.Outputs = append(privateParams.Outputs, standard.PrivateOutput{
					State:      [types.StateLen]byte{},
					Amount:     uint64(outNote.Note.Amount),
					Salt:       outNote.Note.Salt,
					AssetID:    outNote.Note.AssetID,
					ScriptHash: scriptHash[:],
				})
			}

			publicOutputs := make([]standard.PublicOutput, len(outputNotes))
			for i, output := range outputs {
				publicOutputs[i] = standard.PublicOutput{
					Commitment: output.Commitment,
					CipherText: output.Ciphertext,
				}
			}

			publicPrams := &standard.PublicParams{
				TXORoot:    acc.Root().Bytes(),
				SigHash:    sigHash,
				Outputs:    publicOutputs,
				Nullifiers: [][]byte{nullifier.Bytes()},
				Fee:        fee,
				Coinbase:   0,
				MintID:     nil,
				MintAmount: 0,
				Locktime:   time.Time{},
			}

			proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicPrams)
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
	initialCoins uint64, additionalOutputs []*transactions.Output) (*blocks.Block, *SpendableNote, error) {

	// First we'll create the spend note for the coinbase transaction.
	// The initial coins will be generated to the spendKey.
	var salt1 [32]byte
	rand.Read(salt1[:])

	mockStandardScriptCommitment := make([]byte, 32)
	rand.Read(mockStandardScriptCommitment)

	spendPubkeyBytes, err := crypto.MarshalPublicKey(spendKey.GetPublic())
	if err != nil {
		return nil, nil, err
	}

	note1UnlockingScript := &types.UnlockingScript{
		ScriptCommitment: mockStandardScriptCommitment,
		ScriptParams:     [][]byte{spendPubkeyBytes},
	}
	note1ScriptHash := note1UnlockingScript.Hash()
	note1 := &types.SpendNote{
		ScriptHash: note1ScriptHash[:],
		Amount:     types.Amount(initialCoins) / 2,
		AssetID:    types.IlliumCoinID,
		Salt:       salt1,
		State:      [types.StateLen]byte{},
	}

	var salt2 [32]byte
	rand.Read(salt2[:])

	note2UnlockingScript := &types.UnlockingScript{
		ScriptCommitment: mockStandardScriptCommitment,
		ScriptParams:     [][]byte{spendPubkeyBytes},
	}
	note2ScriptHash := note2UnlockingScript.Hash()
	note2 := &types.SpendNote{
		ScriptHash: note2ScriptHash[:],
		Amount:     types.Amount(initialCoins) / 2,
		AssetID:    types.IlliumCoinID,
		Salt:       salt2,
		State:      [types.StateLen]byte{},
	}

	// Next we're going to start building the coinbase transaction
	commitment1 := note1.Commitment()
	commitment2 := note2.Commitment()
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

	nullifier1 := types.CalculateNullifier(0, salt1, note1UnlockingScript.ScriptCommitment, note1UnlockingScript.ScriptParams...)
	nullifier2 := types.CalculateNullifier(1, salt2, note2UnlockingScript.ScriptCommitment, note2UnlockingScript.ScriptParams...)

	publicParams := &standard.PublicParams{
		Outputs: []standard.PublicOutput{
			{
				Commitment: commitment1[:],
			},
			{
				Commitment: commitment2[:],
			},
		},
		Nullifiers: [][]byte{nullifier1.Bytes(), nullifier2.Bytes()},
		Fee:        0,
		Coinbase:   initialCoins,
	}
	privateParams := &standard.PrivateParams{
		Outputs: []standard.PrivateOutput{
			{
				ScriptHash: note1ScriptHash[:],
				Amount:     initialCoins / 2,
				Salt:       note1.Salt,
				AssetID:    note1.AssetID,
				State:      note1.State,
			},
			{
				ScriptHash: note2ScriptHash[:],
				Amount:     initialCoins / 2,
				Salt:       note2.Salt,
				AssetID:    note2.AssetID,
				State:      note2.State,
			},
		},
	}

	proof, err := zk.CreateSnark(standard.StandardCircuit, privateParams, publicParams)
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

	publicParams2 := &stake.PublicParams{
		TXORoot:   txoRoot.Bytes(),
		SigHash:   sigHash2,
		Amount:    initialCoins / 2,
		Nullifier: nullifier1.Bytes(),
	}
	privateParams2 := &stake.PrivateParams{
		AssetID:         types.IlliumCoinID,
		Salt:            salt1,
		State:           [types.StateLen]byte{},
		CommitmentIndex: 0,
		InclusionProof: standard.InclusionProof{
			Hashes:      inclusionProof.Hashes,
			Flags:       inclusionProof.Flags,
			Accumulator: inclusionProof.Accumulator,
		},
		ScriptCommitment: mockStandardScriptCommitment,
		ScriptParams:     [][]byte{spendPubkeyBytes},
		UnlockingParams:  [][]byte{sig3},
	}

	proof2, err := zk.CreateSnark(stake.StakeCircuit, privateParams2, publicParams2)
	if err != nil {
		return nil, nil, err
	}
	stakeTx.Proof = proof2

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
		Note:            note2,
		UnlockingScript: note2UnlockingScript,
		PrivateKey:      spendKey,
	}
	return genesis, spendableNote, nil
}
