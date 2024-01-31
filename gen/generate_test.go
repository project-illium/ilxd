// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package gen

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain"
	"github.com/project-illium/ilxd/blockchain/harness"
	"github.com/project-illium/ilxd/mempool"
	"github.com/project-illium/ilxd/policy"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"github.com/project-illium/ilxd/types/transactions"
	"github.com/project-illium/ilxd/zk"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGenerator(t *testing.T) {
	testHarness, err := harness.NewTestHarness(harness.DefaultOptions())
	assert.NoError(t, err)

	verifier := &zk.MockVerifier{}
	verifier.SetValid(true)
	mpool, err := mempool.NewMempool([]mempool.Option{mempool.DefaultOptions(), mempool.Verifier(verifier), mempool.BlockchainView(testHarness.Blockchain())}...)
	assert.NoError(t, err)

	respChan := make(chan *blocks.XThinnerBlock)

	broadcast := func(blk *blocks.XThinnerBlock) error {
		respChan <- blk
		return nil
	}

	sk := testHarness.ValidatorKey()

	pid, err := peer.IDFromPrivateKey(sk)
	assert.NoError(t, err)
	pidBytes, err := pid.Marshal()
	assert.NoError(t, err)

	generator, err := NewBlockGenerator(
		Blockchain(testHarness.Blockchain()),
		Mempool(mpool),
		BroadcastFunc(broadcast),
		PrivateKey(sk),
		Policy(policy.NewPolicy(repo.DefaultFeePerKilobyte, repo.DefaultMinimumStake, repo.DefaultSoftLimit)),
		tickInterval(time.Millisecond),
	)
	assert.NoError(t, err)

	err = testHarness.GenerateBlocks(10)
	assert.NoError(t, err)

	notes := testHarness.SpendableNotes()
	assert.NotZero(t, len(notes))

	commitment, err := notes[0].Note.Commitment()
	assert.NoError(t, err)

	proof, err := testHarness.Accumulator().GetProof(commitment[:])
	assert.NoError(t, err)

	root := testHarness.Accumulator().Root()

	nullifier, err := types.CalculateNullifier(proof.Index, notes[0].Note.Salt, notes[0].LockingScript.ScriptCommitment.Bytes(), notes[0].LockingScript.LockingParams...)
	assert.NoError(t, err)

	stakeTx := &transactions.StakeTransaction{
		Validator_ID: pidBytes,
		Amount:       uint64(notes[0].Note.Amount),
		Nullifier:    nullifier[:],
		TxoRoot:      root[:],
		Signature:    nil,
		Proof:        make([]byte, 11000),
	}

	sigHash, err := stakeTx.SigHash()
	assert.NoError(t, err)
	sig, err := sk.Sign(sigHash)
	assert.NoError(t, err)
	stakeTx.Signature = sig

	err = testHarness.GenerateBlockWithTransactions([]*transactions.Transaction{transactions.WrapTransaction(stakeTx)}, nil)
	assert.NoError(t, err)

	transferTx := &transactions.StandardTransaction{
		Outputs: []*transactions.Output{
			{
				Commitment: make([]byte, types.CommitmentLen),
				Ciphertext: make([]byte, blockchain.CiphertextLen),
			},
		},
		Nullifiers: [][]byte{nullifier[:]},
		TxoRoot:    root[:],
		Fee:        90000,
		Proof:      make([]byte, 11000),
	}

	err = mpool.ProcessTransaction(transactions.WrapTransaction(transferTx))
	assert.NoError(t, err)

	generator.Start()

	select {
	case blk := <-respChan:
		assert.Equal(t, uint32(1), blk.TxCount)
	case <-time.After(time.Second):
		t.Error("Failed to receive block from broadcast")
	}
}
