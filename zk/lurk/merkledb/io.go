// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package merkledb

import (
	"context"
	"github.com/ipfs/go-datastore"
	"github.com/project-illium/ilxd/params/hash"
	"github.com/project-illium/ilxd/types"
)

const (
	merkleDBKeyPrefix = "merkledb/"
	merkleDBNodeKey   = "node/"
	merkleDBValueKey  = "value/"
	merkleDBKeyKey    = "key/"
	merkleDBRootKey   = "root"
)

func fetchRoot(dbtx datastore.Txn) (*Node, error) {
	val, err := dbtx.Get(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBRootKey))
	if err != nil {
		return nil, err
	}
	return &Node{
		left:  types.NewID(val[:hash.HashSize]),
		right: types.NewID(val[hash.HashSize:]),
	}, nil
}

func putRoot(dbtx datastore.Txn, n *Node) error {
	return dbtx.Put(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBRootKey), append(n.left[:], n.right[:]...))
}

func fetchNode(dbtx datastore.Txn, nodeID types.ID) (*Node, error) {
	val, err := dbtx.Get(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBNodeKey+nodeID.String()))
	if err != nil {
		return nil, err
	}
	return &Node{
		left:  types.NewID(val[:hash.HashSize]),
		right: types.NewID(val[hash.HashSize:]),
	}, nil
}

func putNode(dbtx datastore.Txn, n *Node) error {
	return dbtx.Put(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBNodeKey+n.Hash().String()), append(n.left.Bytes(), n.right.Bytes()...))
}

func deleteNode(dbtx datastore.Txn, nodeID types.ID) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBNodeKey+nodeID.String()))
}

func putValue(dbtx datastore.Txn, key types.ID, value []byte) error {
	return dbtx.Put(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBValueKey+key.String()), value)
}

func getValue(dbtx datastore.Txn, key types.ID) ([]byte, error) {
	return dbtx.Get(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBValueKey+key.String()))
}

func deleteValue(dbtx datastore.Txn, key types.ID) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBValueKey+key.String()))
}

func putKey(dbtx datastore.Txn, valueHash types.ID, key types.ID) error {
	return dbtx.Put(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBKeyKey+valueHash.String()), key.Bytes())
}

func getKey(dbtx datastore.Txn, valueHash types.ID) (types.ID, error) {
	val, err := dbtx.Get(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBKeyKey+valueHash.String()))
	if err != nil {
		return types.ID{}, err
	}
	return types.NewID(val), nil
}

func deleteKey(dbtx datastore.Txn, valueHash types.ID) error {
	return dbtx.Delete(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBKeyKey+valueHash.String()))
}
