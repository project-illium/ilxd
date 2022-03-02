// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package blockchain

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/project-illium/ilxd/blockchain/pb"
	"github.com/project-illium/ilxd/models"
	"github.com/project-illium/ilxd/models/blocks"
	"github.com/project-illium/ilxd/repo"
	"strconv"
)

func serializeValidatorCommitment(v *Validator) ([]byte, error) {
	vProto := &pb.ValidatorCommitment{
		PeerId:     v.PeerID.Pretty(),
		TotalStake: v.TotalStake,
		Nullifiers: make([]*pb.ValidatorCommitment_Nullifier, 0, len(v.Nullifiers)),
	}

	for v, amt := range v.Nullifiers {
		vProto.Nullifiers = append(vProto.Nullifiers, &pb.ValidatorCommitment_Nullifier{
			Hash:   v[:],
			Amount: amt,
		})
	}

	return proto.Marshal(vProto)
}

func serializeValidator(v *Validator) ([]byte, error) {
	vProto := &pb.DBValidator{
		PeerId:         v.PeerID.Pretty(),
		TotalStake:     v.TotalStake,
		Nullifiers:     make([]*pb.DBValidator_Nullifier, 0, len(v.Nullifiers)),
		UnclaimedCoins: v.unclaimedCoins,
		EpochBLocks:    v.epochBlocks,
	}

	for v, amt := range v.Nullifiers {
		vProto.Nullifiers = append(vProto.Nullifiers, &pb.DBValidator_Nullifier{
			Hash:   v[:],
			Amount: amt,
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
		Nullifiers:     make(map[[32]byte]uint64),
		unclaimedCoins: vProto.UnclaimedCoins,
		epochBlocks:    vProto.EpochBLocks,
		isModified:     false,
	}

	for _, n := range vProto.Nullifiers {
		var nullifier [32]byte
		copy(nullifier[:], n.Hash)
		val.Nullifiers[nullifier] = n.Amount
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
		blockID: models.NewID(dbBlockNode.BlockID),
		height:  dbBlockNode.Height,
	}, nil
}

func dsFetchHeader(ds repo.Datastore, blockID models.ID) (*blocks.BlockHeader, error) {
	serialized, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+blockID.String()))
	if err != nil {
		return nil, err
	}
	var blockHeader blocks.BlockHeader
	if err := proto.Unmarshal(serialized, &blockHeader); err != nil {
		return nil, err
	}
	return &blockHeader, nil
}

func dsPutHeader(dbtx datastore.Txn, header *blocks.BlockHeader) error {
	ser, err := proto.Marshal(header)
	if err != nil {
		return err
	}
	return dbtx.Put(context.Background(), datastore.NewKey(repo.BlockKeyPrefix+header.ID().String()), ser)
}

func dsFetchBlock(ds repo.Datastore, blockID models.ID) (*blocks.Block, error) {
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

func dsFetchBlockIDFromHeight(ds repo.Datastore, height uint32) (models.ID, error) {
	blockIDBytes, err := ds.Get(context.Background(), datastore.NewKey(repo.BlockByHeightKeyPrefix+strconv.Itoa(int(height))))
	if err != nil {
		return models.ID{}, err
	}
	return models.NewID(blockIDBytes), nil
}

func dsPutBlockIDFromHeight(dbtx datastore.Txn, blockID models.ID, height uint32) error {
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
