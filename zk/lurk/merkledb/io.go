// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package merkledb

import (
	"context"
	"github.com/ipfs/go-datastore"
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
	size := 32
	if val[0] == 0x01 {
		size = 64
	}
	return &Node{
		left:  val[1 : size+1],
		right: val[size+1:],
	}, nil
}

func putRoot(dbtx datastore.Txn, n *Node) error {
	size := []byte{0x00}
	if len(n.left) == 64 {
		size = []byte{0x01}
	}
	return dbtx.Put(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBRootKey), append(size, append(n.left[:], n.right[:]...)...))
}

func fetchNode(dbtx datastore.Txn, nodeID HashVal) (*Node, error) {
	val, err := dbtx.Get(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBNodeKey+nodeID.String()))
	if err != nil {
		return nil, err
	}
	size := 32
	if val[0] == 0x01 {
		size = 64
	}
	return &Node{
		left:  val[1 : size+1],
		right: val[size+1:],
	}, nil
}

func putNode(dbtx datastore.Txn, n *Node) error {
	size := []byte{0x00}
	if len(n.left) == 64 {
		size = []byte{0x01}
	}
	return dbtx.Put(context.Background(), datastore.NewKey(merkleDBKeyPrefix+merkleDBNodeKey+n.Hash().String()), append(size, append(n.left[:], n.right[:]...)...))
}

func deleteNode(dbtx datastore.Txn, nodeID HashVal) error {
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
