// Copyright (c) 2024 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger"
	datastore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain/pb"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
)

func serializeValidator(v *Validator) ([]byte, error) {
	vProto := &pb.DBValidator{
		PeerId:          v.PeerID.String(),
		TotalStake:      uint64(v.TotalStake),
		WeightedStake:   uint64(v.WeightedStake),
		Nullifiers:      make([]*pb.DBValidator_Nullifier, 0, len(v.Nullifiers)),
		UnclaimedCoins:  uint64(v.UnclaimedCoins),
		EpochBLocks:     v.EpochBlocks,
		Stikes:          v.Strikes,
		CoinbasePenalty: v.CoinbasePenalty,
		ExpectedBlocks:  float32(v.ExpectedBlocks),
		ValidatorSince:  timestamppb.New(v.ValidatorSince),
	}

	for v, stake := range v.Nullifiers {
		n := &pb.DBValidator_Nullifier{
			Amount:         uint64(stake.Amount),
			WeightedAmount: uint64(stake.WeightedAmount),
			Locktime:       timestamppb.New(stake.Locktime),
			Blockstamp:     timestamppb.New(stake.Blockstamp),
			Hash:           make([]byte, len(v)),
		}
		copy(n.Hash, v[:])
		vProto.Nullifiers = append(vProto.Nullifiers, n)
	}

	return proto.Marshal(vProto)
}

func deserializeValidator(ser []byte) (*Validator, error) {
	var vProto pb.DBValidator
	err := proto.Unmarshal(ser, &vProto)
	if err != nil {
		return nil, err
	}
	pid, err := peer.Decode(vProto.PeerId)
	if err != nil {
		return nil, err
	}
	val := &Validator{
		PeerID:          pid,
		TotalStake:      types.Amount(vProto.TotalStake),
		WeightedStake:   types.Amount(vProto.WeightedStake),
		Nullifiers:      make(map[types.Nullifier]Stake),
		UnclaimedCoins:  types.Amount(vProto.UnclaimedCoins),
		EpochBlocks:     vProto.EpochBLocks,
		Strikes:         vProto.Stikes,
		CoinbasePenalty: vProto.CoinbasePenalty,
		ExpectedBlocks:  float64(vProto.ExpectedBlocks),
		ValidatorSince:  vProto.ValidatorSince.AsTime(),
	}

	for _, n := range vProto.Nullifiers {
		var nullifier [32]byte
		copy(nullifier[:], n.Hash)
		val.Nullifiers[nullifier] = Stake{
			Amount:         types.Amount(n.Amount),
			WeightedAmount: types.Amount(n.WeightedAmount),
			Locktime:       n.Locktime.AsTime(),
			Blockstamp:     n.Blockstamp.AsTime(),
		}
	}
	return val, nil
}

func serializeBlockNode(node *blockNode) ([]byte, error) {
	return proto.Marshal(&pb.DBBlockNode{
		BlockID:   node.blockID[:],
		Height:    node.height,
		Timestamp: node.timestamp,
	})
}

func deserializeBlockNode(ser []byte) (*blockNode, error) {
	var dbBlockNode pb.DBBlockNode
	if err := proto.Unmarshal(ser, &dbBlockNode); err != nil {
		return nil, err
	}
	return &blockNode{
		blockID:   types.NewID(dbBlockNode.BlockID),
		height:    dbBlockNode.Height,
		timestamp: dbBlockNode.Timestamp,
	}, nil
}

func SerializeAccumulator(accumulator *Accumulator) ([]byte, error) {
	proofs := make([]*pb.DBAccumulator_InclusionProof, 0, len(accumulator.proofs))
	for id, p := range accumulator.proofs {
		proof := &pb.DBAccumulator_InclusionProof{
			Key:    make([]byte, len(id.Bytes())),
			Id:     make([]byte, len(p.ID.Bytes())),
			Index:  p.Index,
			Hashes: make([][]byte, len(p.Hashes)),
			Flags:  p.Flags,
			Last:   p.last,
		}
		copy(proof.Key, id.Bytes())
		copy(proof.Id, p.ID.Bytes())
		for i := range p.Hashes {
			proof.Hashes[i] = make([]byte, len(p.Hashes[i]))
			copy(proof.Hashes[i], p.Hashes[i])
		}
		proofs = append(proofs, proof)
	}
	lookUpMap := make([]*pb.DBAccumulator_LookupMap, 0, len(accumulator.lookupMap))
	for id, idx := range accumulator.lookupMap {
		m := &pb.DBAccumulator_LookupMap{
			Key:   make([]byte, len(id.Bytes())),
			Index: idx,
		}
		copy(m.Key, id.Bytes())
		lookUpMap = append(lookUpMap, m)
	}
	dbAcc := &pb.DBAccumulator{
		Accumulator: make([][]byte, len(accumulator.acc)),
		NElements:   accumulator.nElements,
		Proofs:      proofs,
		LookupMap:   lookUpMap,
	}
	for i := range accumulator.acc {
		dbAcc.Accumulator[i] = make([]byte, len(accumulator.acc[i]))
		copy(dbAcc.Accumulator[i], accumulator.acc[i])
	}

	return proto.Marshal(dbAcc)
}

func DeserializeAccumulator(ser []byte) (*Accumulator, error) {
	var dbAcc pb.DBAccumulator
	if err := proto.Unmarshal(ser, &dbAcc); err != nil {
		return nil, err
	}
	acc := &Accumulator{
		acc:       make([][]byte, len(dbAcc.Accumulator)),
		nElements: dbAcc.NElements,
		proofs:    make(map[types.ID]*InclusionProof),
		lookupMap: make(map[types.ID]uint64),
	}
	for i := range dbAcc.Accumulator {
		if len(dbAcc.Accumulator[i]) == 0 {
			acc.acc[i] = nil
		} else {
			acc.acc[i] = make([]byte, len(dbAcc.Accumulator[i]))
			copy(acc.acc[i], dbAcc.Accumulator[i])
		}
	}
	for _, entry := range dbAcc.Proofs {
		acc.proofs[types.NewID(entry.Key)] = &InclusionProof{
			ID:     types.NewID(entry.Id),
			Hashes: entry.Hashes,
			Flags:  entry.Flags,
			Index:  entry.Index,
			last:   entry.Last,
		}
	}
	for _, entry := range dbAcc.LookupMap {
		acc.lookupMap[types.NewID(entry.Key)] = entry.Index
	}
	return acc, nil
}

func dsPutHeader(dbtx datastore.Txn, header *blocks.BlockHeader) error {
	ser, err := header.Serialize()
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+header.ID().String()), ser)
}

func dsFetchHeader(ds repo.Datastore, blockID types.ID) (*blocks.BlockHeader, error) {
	serialized, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+blockID.String()))
	if err != nil {
		return nil, err
	}
	blockHeader := &blocks.BlockHeader{}
	if err := blockHeader.Deserialize(serialized); err != nil {
		return nil, err
	}
	return blockHeader, nil
}

func dsBlockExists(ds repo.Datastore, blockID types.ID) (bool, error) {
	return ds.Has(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+blockID.String()))
}

func dsPutBlock(dbtx datastore.Txn, blk *blocks.Block) error {
	serializedHeader, err := blk.Header.Serialize()
	if err != nil {
		return err
	}
	if err := dbtx.Put(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+blk.ID().String()), serializedHeader); err != nil {
		return err
	}
	txns := &pb.DBTxs{
		Transactions: blk.Transactions,
	}
	serializedTxs, err := proto.Marshal(txns)
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.BlockTxsKeyPrefix+blk.ID().String()), serializedTxs)
}

func dsDeleteBlock(dbtx datastore.Txn, blockID types.ID) error {
	if err := dbtx.Delete(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+blockID.String())); err != nil {
		return err
	}
	return dbtx.Delete(context.Background(), datastore.NewKey(repo.BlockTxsKeyPrefix+blockID.String()))
}

func dsFetchBlock(ds repo.Datastore, blockID types.ID) (*blocks.Block, error) {
	serializedHeader, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+blockID.String()))
	if err != nil {
		return nil, err
	}
	serializedTxs, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockTxsKeyPrefix+blockID.String()))
	if err != nil {
		return nil, err
	}
	var blockHeader blocks.BlockHeader
	if err := proto.Unmarshal(serializedHeader, &blockHeader); err != nil {
		return nil, err
	}

	var dsTxs pb.DBTxs
	if err := proto.Unmarshal(serializedTxs, &dsTxs); err != nil {
		return nil, err
	}
	return &blocks.Block{
		Header:       &blockHeader,
		Transactions: dsTxs.Transactions,
	}, nil
}

func dsPutBlockIDFromHeight(dbtx datastore.Txn, blockID types.ID, height uint32) error {
	return dbtx.Put(context.Background(), datastore.NewKey(repo.BlockByHeightKeyPrefix+fmt.Sprintf("%010d", int(height))), blockID[:])
}

func dsFetchBlockIDFromHeight(ds repo.Datastore, height uint32) (types.ID, error) {
	blockIDBytes, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockByHeightKeyPrefix+fmt.Sprintf("%010d", int(height))))
	if err != nil {
		return types.ID{}, err
	}
	return types.NewID(blockIDBytes), nil
}

func dsFetchBlockIDFromHeightWithTx(dbtx datastore.Txn, height uint32) (types.ID, error) {
	blockIDBytes, err := dbtx.Get(context.Background(), datastore.NewKey(repo.BlockByHeightKeyPrefix+fmt.Sprintf("%010d", int(height))))
	if err != nil {
		return types.ID{}, err
	}
	return types.NewID(blockIDBytes), nil
}

func dsDeleteBlockIDFromHeight(dbtx datastore.Txn, height uint32) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(repo.BlockByHeightKeyPrefix+fmt.Sprintf("%010d", int(height))))
}

func dsPutBlockIndexState(dbtx datastore.Txn, node *blockNode) error {
	ser, err := serializeBlockNode(node)
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.BlockIndexStateKey), ser)
}

func dsFetchBlockIndexState(ds repo.Datastore) (*blockNode, error) {
	ser, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockIndexStateKey))
	if err != nil {
		return nil, err
	}
	node, err := deserializeBlockNode(ser)
	if err != nil {
		return nil, err
	}
	node.ds = ds
	return node, nil
}

func dsDeleteBlockIndexState(dbtx datastore.Txn) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(repo.BlockIndexStateKey))
}

func dsPutValidatorSetConsistencyStatus(ds repo.Datastore, status setConsistencyStatus) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(status))
	return ds.Put(context.Background(), datastore.NewKey(repo.ValidatorSetConsistencyStatusKey), b)
}

func dsFetchValidatorSetConsistencyStatus(ds repo.Datastore) (setConsistencyStatus, error) {
	b, err := ds.Get(context.Background(), datastore.NewKey(repo.ValidatorSetConsistencyStatusKey))
	if err == datastore.ErrNotFound {
		return scsEmpty, nil
	}
	if err != nil {
		return 0, err
	}
	return setConsistencyStatus(binary.BigEndian.Uint16(b)), nil
}

func dsPutValidatorLastFlushHeight(dbtx datastore.Txn, height uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, height)
	return dbtx.Put(context.Background(), datastore.NewKey(repo.ValidatorSetLastFlushHeight), b)
}

func dsFetchValidatorLastFlushHeight(ds repo.Datastore) (uint32, error) {
	b, err := ds.Get(context.Background(), datastore.NewKey(repo.ValidatorSetLastFlushHeight))
	if err == datastore.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

func dsPutValidator(dbtx datastore.Txn, v *Validator) error {
	ser, err := serializeValidator(v)
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.ValidatorDatastoreKeyPrefix+v.PeerID.String()), ser)
}

func dsDeleteValidatorSet(dbtx datastore.Txn) error {
	q := query.Query{
		Prefix: repo.ValidatorDatastoreKeyPrefix,
	}

	results, err := dbtx.Query(context.Background(), q)
	if err != nil {
		return err
	}

	for result, ok := results.NextSync(); ok; result, ok = results.NextSync() {
		if err := dbtx.Delete(context.Background(), datastore.NewKey(result.Key)); err != nil {
			return err
		}
	}
	return nil
}

func dsDeleteValidator(dbtx datastore.Txn, id peer.ID) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(repo.ValidatorDatastoreKeyPrefix+id.String()))
}

func dsFetchValidators(ds repo.Datastore) ([]*Validator, error) {
	q := query.Query{
		Prefix: repo.ValidatorDatastoreKeyPrefix,
	}

	results, err := ds.Query(context.Background(), q)
	if err != nil {
		return nil, err
	}

	var validators []*Validator
	for result, ok := results.NextSync(); ok; result, ok = results.NextSync() {
		validator, err := deserializeValidator(result.Value)
		if err != nil {
			return nil, err
		}
		validators = append(validators, validator)
	}
	return validators, nil
}

func dsNullifierExists(ds repo.Datastore, nullifier types.Nullifier) (bool, error) {
	return ds.Has(context.Background(), datastore.NewKey(repo.NullifierKeyPrefix+nullifier.String()))
}

func dsPutNullifiers(dbtx datastore.Txn, nullifiers []types.Nullifier) error {
	for _, n := range nullifiers {
		if err := dbtx.Put(context.Background(), datastore.NewKey(repo.NullifierKeyPrefix+n.String()), []byte{}); err != nil {
			return err
		}
	}
	return nil
}

func dsDeleteNullifierSet(dbtx datastore.Txn) error {
	q := query.Query{
		Prefix: repo.NullifierKeyPrefix,
	}

	results, err := dbtx.Query(context.Background(), q)
	if err != nil {
		return err
	}

	for result, ok := results.NextSync(); ok; result, ok = results.NextSync() {
		if err := dbtx.Delete(context.Background(), datastore.NewKey(result.Key)); err != nil {
			return err
		}
	}
	return nil
}

func dsPutTxoSetRoot(dbtx datastore.Txn, txoRoot types.ID) error {
	return dbtx.Put(context.Background(), datastore.NewKey(repo.TxoRootKeyPrefix+txoRoot.String()), []byte{})
}

func dsTxoSetRootExists(ds repo.Datastore, txoRoot types.ID) (bool, error) {
	return ds.Has(context.Background(), datastore.NewKey(repo.TxoRootKeyPrefix+txoRoot.String()))
}

func dsDeleteTxoRootSet(dbtx datastore.Txn) error {
	q := query.Query{
		Prefix: repo.TxoRootKeyPrefix,
	}

	results, err := dbtx.Query(context.Background(), q)
	if err != nil {
		return err
	}

	for result, ok := results.NextSync(); ok; result, ok = results.NextSync() {
		if err := dbtx.Delete(context.Background(), datastore.NewKey(result.Key)); err != nil {
			return err
		}
	}
	return nil
}

func dsDebitTreasury(dbtx datastore.Txn, amount types.Amount) error {
	balanceBytes, err := dbtx.Get(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey))
	if err != nil {
		return err
	}
	balance := types.Amount(binary.BigEndian.Uint64(balanceBytes))
	balance -= amount

	newBalance := make([]byte, 8)
	binary.BigEndian.PutUint64(newBalance, uint64(balance))
	return dbtx.Put(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey), newBalance)
}

func dsInitTreasury(ds datastore.Datastore) error {
	zero := make([]byte, 8)
	return ds.Put(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey), zero)
}

func dsCreditTreasury(dbtx datastore.Txn, amount types.Amount) error {
	balanceBytes, err := dbtx.Get(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey))
	if err != nil {
		return err
	}
	balance := types.Amount(binary.BigEndian.Uint64(balanceBytes))
	balance += amount

	newBalance := make([]byte, 8)
	binary.BigEndian.PutUint64(newBalance, uint64(balance))
	return dbtx.Put(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey), newBalance)
}

func dsFetchTreasuryBalance(ds repo.Datastore) (types.Amount, error) {
	balance, err := ds.Get(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey))
	if err == datastore.ErrNotFound {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return types.Amount(binary.BigEndian.Uint64(balance)), nil
}

func dsPutAccumulator(dbtx datastore.Txn, accumulator *Accumulator) error {
	ser, err := SerializeAccumulator(accumulator)
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.AccumulatorStateKey), ser)
}

func dsFetchAccumulator(ds repo.Datastore) (*Accumulator, error) {
	ser, err := ds.Get(context.Background(), datastore.NewKey(repo.AccumulatorStateKey))
	if err != nil {
		return nil, err
	}
	return DeserializeAccumulator(ser)
}

func dsDeleteAccumulator(dbtx datastore.Txn) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(repo.AccumulatorStateKey))
}

func dsPutAccumulatorCheckpoint(dbtx datastore.Txn, height uint32, accumulator *Accumulator) error {
	ser, err := SerializeAccumulator(accumulator)
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.AccumulatorCheckpointKey+fmt.Sprintf("%010d", int(height))), ser)
}

func dsFetchAccumulatorCheckpoint(ds repo.Datastore, height uint32) (*Accumulator, error) {
	ser, err := ds.Get(context.Background(), datastore.NewKey(repo.AccumulatorCheckpointKey+fmt.Sprintf("%010d", int(height))))
	if err != nil {
		return nil, err
	}
	return DeserializeAccumulator(ser)
}

func dsDeleteAccumulatorCheckpoints(dbtx datastore.Txn) error {
	q := query.Query{
		Prefix: repo.AccumulatorCheckpointKey,
	}

	results, err := dbtx.Query(context.Background(), q)
	if err != nil {
		return err
	}

	for result, ok := results.NextSync(); ok; result, ok = results.NextSync() {
		if err := dbtx.Delete(context.Background(), datastore.NewKey(result.Key)); err != nil {
			return err
		}
	}
	return nil
}

func dsPutAccumulatorConsistencyStatus(ds repo.Datastore, status setConsistencyStatus) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(status))
	return ds.Put(context.Background(), datastore.NewKey(repo.AccumulatorConsistencyStatusKey), b)
}

func dsFetchAccumulatorConsistencyStatus(ds repo.Datastore) (setConsistencyStatus, error) {
	b, err := ds.Get(context.Background(), datastore.NewKey(repo.AccumulatorConsistencyStatusKey))
	if err == datastore.ErrNotFound {
		return scsEmpty, nil
	}
	if err != nil {
		return 0, err
	}
	return setConsistencyStatus(binary.BigEndian.Uint16(b)), nil
}

func dsPutAccumulatorLastFlushHeight(dbtx datastore.Txn, height uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, height)
	return dbtx.Put(context.Background(), datastore.NewKey(repo.AccumulatorLastFlushHeight), b)
}

func dsFetchAccumulatorLastFlushHeight(ds repo.Datastore) (uint32, error) {
	b, err := ds.Get(context.Background(), datastore.NewKey(repo.AccumulatorLastFlushHeight))
	if err == datastore.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

func dsIncrementCurrentSupply(dbtx datastore.Txn, newCoins types.Amount) error {
	currentSupply, err := dsFetchCurrentSupply(dbtx)
	if err != nil {
		return err
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(currentSupply+newCoins))
	return dbtx.Put(context.Background(), datastore.NewKey(repo.CoinSupplyKey), b)
}

func dsInitCurrentSupply(ds datastore.Datastore) error {
	zero := make([]byte, 8)
	return ds.Put(context.Background(), datastore.NewKey(repo.CoinSupplyKey), zero)
}

func dsFetchCurrentSupply(dbtx datastore.Txn) (types.Amount, error) {
	b, err := dbtx.Get(context.Background(), datastore.NewKey(repo.CoinSupplyKey))
	if err != nil {
		return 0, err
	}
	return types.Amount(binary.BigEndian.Uint64(b)), nil
}

func dsPutPrunedFlag(ds repo.Datastore) error {
	return ds.Put(context.Background(), datastore.NewKey(repo.PrunedBlockchainDatastoreKey), []byte{})
}

func dsFetchPrunedFlag(ds repo.Datastore) (bool, error) {
	_, err := ds.Get(context.Background(), datastore.NewKey(repo.PrunedBlockchainDatastoreKey))
	if err != nil && !errors.Is(err, datastore.ErrNotFound) {
		return false, err
	}
	return !errors.Is(err, datastore.ErrNotFound), nil
}

func dsPutEpoch(dbtx datastore.Txn, blockID types.ID, height uint32) error {
	val := make([]byte, 36)
	blockBytes := blockID.Bytes()
	for i := 0; i < 32; i++ {
		val[i] = blockBytes[i]
	}
	binary.BigEndian.PutUint32(val[32:], height)

	return dbtx.Put(context.Background(), datastore.NewKey(repo.EpochDatastoreKey), val)
}

func dsFetchEpoch(ds datastore.Datastore) (types.ID, uint32, error) {
	val, err := ds.Get(context.Background(), datastore.NewKey(repo.EpochDatastoreKey))
	if err != nil {
		return types.ID{}, 0, err
	}
	id := types.NewID(val[:32])
	height := binary.BigEndian.Uint32(val[32:])
	return id, height, nil
}

// The database implementation limits the number of put operations in
// a transaction as well as the size of the transaction (in bytes).
//
// This function computes an estimate for those limits for a block,
// so we can make sure we can actually connect the block to the chain.
// It computes the ops and size assuming all execution paths are used
// even if not all are.
//
// These limits are high enough that we aren't likely to hit them any
// time in the near future, and the soft limit would need to be raised
// to hit them, but they do serve as a hard limit on the number of
// transactions that can fit in a block.
//
// If we every get to the point where we are threatening to hit these
// limits we should revisit the database implementation and find
// a solution that doesn't impose these limits.
func datastoreTxnLimits(blk *blocks.Block, bannedNullifiers int) (int, int, error) {
	txn := dbTxLimitCounter{}
	serializedHeader, err := blk.Header.Serialize()
	if err != nil {
		return 0, 0, err
	}
	txns := &pb.DBTxs{
		Transactions: blk.Transactions,
	}
	serializedTxs, err := proto.Marshal(txns)
	if err != nil {
		return 0, 0, err
	}
	serializedNode, err := serializeBlockNode(&blockNode{
		blockID:   types.ID{},
		height:    0,
		timestamp: 0,
	})
	if err != nil {
		return 0, 0, err
	}
	accumulator := NewAccumulator()
	accumulator.acc = make([][]byte, 64)
	for i := range accumulator.acc {
		accumulator.acc[i] = make([]byte, 32)
	}
	serializedAccumulator, err := SerializeAccumulator(accumulator)
	if err != nil {
		return 0, 0, err
	}

	id := types.ID{}
	maxU32 := math.MaxUint32
	fourBytes := make([]byte, 4)
	eightBytes := make([]byte, 8)
	serializedViewKey := hex.EncodeToString(make([]byte, 72))

	txn.Put(datastore.NewKey(repo.BlockKeyPrefix+blk.ID().String()), serializedHeader)
	txn.Put(datastore.NewKey(repo.BlockTxsKeyPrefix+blk.ID().String()), serializedTxs)
	txn.Put(datastore.NewKey(repo.BlockByHeightKeyPrefix+fmt.Sprintf("%010d", maxU32)), id[:])
	txn.Put(datastore.NewKey(repo.BlockIndexStateKey), serializedNode)
	txn.Put(datastore.NewKey(repo.TreasuryBalanceKey), fourBytes)
	txn.Put(datastore.NewKey(repo.TxoRootKeyPrefix+id.String()), []byte{})
	txn.Put(datastore.NewKey(repo.CoinSupplyKey), eightBytes)
	txn.Put(datastore.NewKey(repo.TreasuryBalanceKey), eightBytes)
	txn.Put(datastore.NewKey(repo.AccumulatorCheckpointKey+fmt.Sprintf("%010d", maxU32)), serializedAccumulator)
	txn.Delete(datastore.NewKey(repo.BlockByHeightKeyPrefix + fmt.Sprintf("%010d", maxU32)))
	txn.Delete(datastore.NewKey(repo.BlockKeyPrefix + id.String()))
	txn.Delete(datastore.NewKey(repo.BlockTxsKeyPrefix + id.String()))
	txn.Put(datastore.NewKey(repo.IndexerHeightKeyPrefix+repo.TxIndexKey), fourBytes)
	txn.Put(datastore.NewKey(repo.IndexerHeightKeyPrefix+repo.WalletServerIndexKey), fourBytes)
	for range blk.Outputs() {
		dsKey := repo.WalletServerTxKeyPrefix + serializedViewKey + "/" + id.String()
		txn.Put(datastore.NewKey(repo.IndexKeyPrefix+repo.WalletServerIndexKey+"/"+dsKey), nil)
		dsKey = repo.WalletServerNullifierKeyPrefix + serializedViewKey + "/" + id.String()
		txn.Put(datastore.NewKey(repo.IndexKeyPrefix+repo.WalletServerIndexKey+"/"+dsKey), id.Bytes())
	}
	for range blk.Nullifiers() {
		txn.Put(datastore.NewKey(repo.NullifierKeyPrefix+id.String()), id.Bytes())
		dsKey := repo.WalletServerTxKeyPrefix + serializedViewKey + "/" + id.String()
		txn.Put(datastore.NewKey(repo.IndexKeyPrefix+repo.WalletServerIndexKey+"/"+dsKey), nil)
		dsKey = repo.WalletServerNullifierKeyPrefix + serializedViewKey + "/" + id.String()
		txn.Delete(datastore.NewKey(repo.IndexKeyPrefix + repo.WalletServerIndexKey + "/" + dsKey))
	}
	for i := 0; i < bannedNullifiers; i++ {
		txn.Put(datastore.NewKey(repo.NullifierKeyPrefix+id.String()), id.Bytes())
	}
	for range blk.Transactions {
		txn.Put(datastore.NewKey(repo.IndexKeyPrefix+repo.TxIndexKey+"/"+id.String()), make([]byte, 36))
	}
	// Add a buffer just in case
	return txn.count + 10, txn.size + 500, nil
}

type dbTxLimitCounter struct {
	count int
	size  int
}

func (txn *dbTxLimitCounter) Put(key datastore.Key, value []byte) {
	e := badger.Entry{
		Key:   key.Bytes(),
		Value: value,
	}
	txn.count++
	txn.size += estimateSize(e, dsValueThreshold) + 10
}

func (txn *dbTxLimitCounter) Delete(key datastore.Key) {
	e := badger.Entry{
		Key: key.Bytes(),
	}
	txn.count++
	txn.size += estimateSize(e, dsValueThreshold) + 10
}

func estimateSize(e badger.Entry, threshold int) int {
	if len(e.Value) < threshold {
		return len(e.Key) + len(e.Value) + 2 // Meta, UserMeta
	}
	return len(e.Key) + 12 + 2 // 12 for ValuePointer, 2 for metas.
}
