// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	datastore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/project-illium/ilxd/blockchain/pb"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeValidator(v *Validator) ([]byte, error) {
	vProto := &pb.DBValidator{
		PeerId:         v.PeerID.String(),
		TotalStake:     v.TotalStake,
		Nullifiers:     make([]*pb.DBValidator_Nullifier, 0, len(v.Nullifiers)),
		UnclaimedCoins: v.unclaimedCoins,
		EpochBLocks:    v.epochBlocks,
	}

	for v, stake := range v.Nullifiers {
		vProto.Nullifiers = append(vProto.Nullifiers, &pb.DBValidator_Nullifier{
			Hash:       v[:],
			Amount:     stake.Amount,
			Blockstamp: timestamppb.New(stake.Blockstamp),
		})
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
		PeerID:         pid,
		TotalStake:     vProto.TotalStake,
		Nullifiers:     make(map[types.Nullifier]Stake),
		unclaimedCoins: vProto.UnclaimedCoins,
		epochBlocks:    vProto.EpochBLocks,
		dirty:          false,
	}

	for _, n := range vProto.Nullifiers {
		var nullifier [32]byte
		copy(nullifier[:], n.Hash)
		val.Nullifiers[nullifier] = Stake{
			Amount:     n.Amount,
			Blockstamp: n.Blockstamp.AsTime(),
		}
	}
	return val, nil
}

func serializeBlockNode(node *blockNode) ([]byte, error) {
	return proto.Marshal(&pb.DBBlockNode{
		BlockID: node.blockID[:],
		Height:  node.height,
	})
}

func deserializeBlockNode(ser []byte) (*blockNode, error) {
	var dbBlockNode pb.DBBlockNode
	if err := proto.Unmarshal(ser, &dbBlockNode); err != nil {
		return nil, err
	}
	return &blockNode{
		blockID: types.NewID(dbBlockNode.BlockID),
		height:  dbBlockNode.Height,
	}, nil
}

func serializeAccumulator(accumulator *Accumulator) ([]byte, error) {
	proofs := make([]*pb.DBAccumulator_InclusionProof, 0, len(accumulator.proofs))
	for id, p := range accumulator.proofs {
		proofs = append(proofs, &pb.DBAccumulator_InclusionProof{
			Key:    id.Bytes(),
			Id:     p.ID.Bytes(),
			Index:  p.Index,
			Hashes: p.Hashes,
			Flags:  p.Flags,
			Last:   p.last,
		})
	}
	lookUpMap := make([]*pb.DBAccumulator_InclusionProof, 0, len(accumulator.lookupMap))
	for id, p := range accumulator.lookupMap {
		lookUpMap = append(lookUpMap, &pb.DBAccumulator_InclusionProof{
			Key:    id.Bytes(),
			Id:     p.ID.Bytes(),
			Index:  p.Index,
			Hashes: p.Hashes,
			Flags:  p.Flags,
			Last:   p.last,
		})
	}
	dbAcc := &pb.DBAccumulator{
		Accumulator: accumulator.acc,
		NElements:   accumulator.nElements,
		Proofs:      proofs,
		LookupMap:   lookUpMap,
	}

	return proto.Marshal(dbAcc)
}

func deserializeAccumulator(ser []byte) (*Accumulator, error) {
	var dbAcc pb.DBAccumulator
	if err := proto.Unmarshal(ser, &dbAcc); err != nil {
		return nil, err
	}
	acc := &Accumulator{
		acc:       dbAcc.Accumulator,
		nElements: dbAcc.NElements,
		proofs:    make(map[types.ID]*InclusionProof),
		lookupMap: make(map[types.ID]*InclusionProof),
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
		acc.lookupMap[types.NewID(entry.Key)] = &InclusionProof{
			ID:     types.NewID(entry.Id),
			Hashes: entry.Hashes,
			Flags:  entry.Flags,
			Index:  entry.Index,
			last:   entry.Last,
		}
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

func dsPutTxoSetRoot(dbtx datastore.Txn, txoRoot types.ID) error {
	return dbtx.Put(context.Background(), datastore.NewKey(repo.TxoRootKeyPrefix+txoRoot.String()), []byte{})
}

func dsTxoSetRootExists(ds repo.Datastore, txoRoot types.ID) (bool, error) {
	return ds.Has(context.Background(), datastore.NewKey(repo.TxoRootKeyPrefix+txoRoot.String()))
}

func dsDebitTreasury(dbtx datastore.Txn, amount uint64) error {
	balanceBytes, err := dbtx.Get(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey))
	if err != nil {
		return err
	}
	balance := binary.BigEndian.Uint64(balanceBytes)
	balance -= amount

	newBalance := make([]byte, 8)
	binary.BigEndian.PutUint64(newBalance, balance)
	return dbtx.Put(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey), newBalance)
}

func dsInitTreasury(ds datastore.Datastore) error {
	zero := make([]byte, 8)
	return ds.Put(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey), zero)
}

func dsCreditTreasury(dbtx datastore.Txn, amount uint64) error {
	balanceBytes, err := dbtx.Get(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey))
	if err != nil {
		return err
	}
	balance := binary.BigEndian.Uint64(balanceBytes)
	balance += amount

	newBalance := make([]byte, 8)
	binary.BigEndian.PutUint64(newBalance, balance)
	return dbtx.Put(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey), newBalance)
}

func dsFetchTreasuryBalance(ds repo.Datastore) (uint64, error) {
	balance, err := ds.Get(context.Background(), datastore.NewKey(repo.TreasuryBalanceKey))
	if err == datastore.ErrNotFound {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(balance), nil
}

func dsPutAccumulator(dbtx datastore.Txn, accumulator *Accumulator) error {
	ser, err := serializeAccumulator(accumulator)
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
	return deserializeAccumulator(ser)
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

func dsIncrementCurrentSupply(dbtx datastore.Txn, newCoins uint64) error {
	currentSupply, err := dsFetchCurrentSupply(dbtx)
	if err != nil {
		return err
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, currentSupply+newCoins)
	return dbtx.Put(context.Background(), datastore.NewKey(repo.CoinSupplyKey), b)
}

func dsInitCurrentSupply(ds datastore.Datastore) error {
	zero := make([]byte, 8)
	return ds.Put(context.Background(), datastore.NewKey(repo.CoinSupplyKey), zero)
}

func dsFetchCurrentSupply(dbtx datastore.Txn) (uint64, error) {
	b, err := dbtx.Get(context.Background(), datastore.NewKey(repo.CoinSupplyKey))
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}
