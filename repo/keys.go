// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package repo

import (
	"context"
	"crypto/rand"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func HasNetworkKey(ds datastore.Datastore) (bool, error) {
	return ds.Has(context.Background(), datastore.NewKey(NetworkKeyDatastoreKey))
}

func LoadNetworkKey(ds datastore.Datastore) (crypto.PrivKey, error) {
	keyBytes, err := ds.Get(context.Background(), datastore.NewKey(NetworkKeyDatastoreKey))
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPrivateKey(keyBytes)
}

func PutNetworkKey(ds datastore.Datastore, key crypto.PrivKey) error {
	keyBytes, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return err
	}
	return ds.Put(context.Background(), datastore.NewKey(NetworkKeyDatastoreKey), keyBytes)
}

func GenerateNetworkKeypair() (crypto.PrivKey, crypto.PubKey, error) {
	privkey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privkey, nil, nil
}
