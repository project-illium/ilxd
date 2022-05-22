// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/blockchain/pb"
	"github.com/project-illium/ilxd/repo"
	"github.com/project-illium/ilxd/types"
	"github.com/project-illium/ilxd/types/blocks"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strconv"
)

func serializeValidator(v *Validator) ([]byte, error) {
	vProto := &pb.DBValidator{
		PeerId:         v.PeerID.Pretty(),
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
	pid, err := peer.IDFromString(vProto.PeerId)
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

func dsFetchHeader(ds repo.Datastore, blockID types.ID) (*blocks.BlockHeader, error) {
	serialized, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+blockID.String()))
	if err != nil {
		return nil, err
	}
	var blockHeader blocks.BlockHeader
	if err := blockHeader.Deserialize(serialized); err != nil {
		return nil, err
	}
	return &blockHeader, nil
}

func dsPutHeader(dbtx datastore.Txn, header *blocks.BlockHeader) error {
	ser, err := header.Serialize()
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+header.ID().String()), ser)
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

func dsFetchBlockIDFromHeight(ds repo.Datastore, height uint32) (types.ID, error) {
	blockIDBytes, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockByHeightKeyPrefix+strconv.Itoa(int(height))))
	if err != nil {
		return types.ID{}, err
	}
	return types.NewID(blockIDBytes), nil
}

func dsPutBlockIDFromHeight(dbtx datastore.Txn, blockID types.ID, height uint32) error {
	return dbtx.Put(context.Background(), datastore.NewKey(repo.BlockByHeightKeyPrefix+strconv.Itoa(int(height))), blockID[:])
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
	return dbtx.Put(context.Background(), datastore.NewKey(repo.ValidatorDatastoreKeyPrefix+v.PeerID.Pretty()), ser)
}

func dsDeleteValidator(dbtx datastore.Txn, id peer.ID) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(repo.ValidatorDatastoreKeyPrefix+id.Pretty()))
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

func dsPutNullifierSetConsistencyStatus(ds repo.Datastore, status setConsistencyStatus) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(status))
	return ds.Put(context.Background(), datastore.NewKey(repo.NullifierSetConsistencyStatusKey), b)
}

func dsFetchNullifierSetConsistencyStatus(ds repo.Datastore) (setConsistencyStatus, error) {
	b, err := ds.Get(context.Background(), datastore.NewKey(repo.NullifierSetConsistencyStatusKey))
	if err == datastore.ErrNotFound {
		return scsEmpty, nil
	}
	if err != nil {
		return 0, err
	}
	return setConsistencyStatus(binary.BigEndian.Uint16(b)), nil
}

func dsPutNullifierLastFlushHeight(dbtx datastore.Txn, height uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, height)
	return dbtx.Put(context.Background(), datastore.NewKey(repo.NullifierSetLastFlushHeight), b)
}

func dsFetchNullifierLastFlushHeight(ds repo.Datastore) (uint32, error) {
	b, err := ds.Get(context.Background(), datastore.NewKey(repo.NullifierSetLastFlushHeight))
	if err == datastore.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

func dsNullifierExists(ds repo.Datastore, nullifier types.Nullifier) (bool, error) {
	_, err := ds.Get(context.Background(), datastore.NewKey(repo.NullifierKeyPrefix+nullifier.String()))
	if err == datastore.ErrNotFound {
		return false, nil
	} else if err == nil {
		return true, nil
	}
	return false, err
}

func dsPutNullifiers(dbtx datastore.Txn, nullifiers []types.Nullifier) error {
	for _, n := range nullifiers {
		if err := dbtx.Put(context.Background(), datastore.NewKey(repo.NullifierKeyPrefix+n.String()), []byte{}); err != nil {
			return err
		}
	}
	return nil
}

func dsDeleteNullifiers(dbtx datastore.Txn, nullifiers []types.Nullifier) error {
	for _, n := range nullifiers {
		if err := dbtx.Delete(context.Background(), datastore.NewKey(repo.NullifierKeyPrefix+n.String())); err != nil {
			return err
		}
	}
	return nil
}
