// Copyright (c) 2022 The illium developers
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package net

import (
	"errors"
	"strconv"
	"strings"
)

type Service uint64

const (
	ServiceBlockchain Service = 1
)

type PeerServices uint64

func (s PeerServices) HasService(service Service) bool {
	return uint64(s)&uint64(service) > 0
}

func (s PeerServices) String() string {
	return strconv.Itoa(int(s))
}

func ExtractPeerServices(userAgent string) (PeerServices, error) {
	if len(userAgent) == 0 {
		return PeerServices(0), errors.New("userAgent is empty")
	}
	s := strings.Split(userAgent, "/")
	servicesStr := s[len(s)-1]

	servicesInt, err := strconv.Atoi(servicesStr)
	if err != nil {
		return PeerServices(0), err
	}
	return PeerServices(servicesInt), nil
}
