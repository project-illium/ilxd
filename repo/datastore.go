// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package repo

import (
	"github.com/ipfs/go-datastore"
)

const Libp2pDatastoreKey = "/obx/libp2pkey/"

type Datastore interface {
	datastore.Datastore
	datastore.Batching
	datastore.PersistentDatastore
	datastore.TxnDatastore
}
