package main

import (
	"orly.dev/pkg/crypto/p256k"
	"orly.dev/pkg/interfaces/signer"
	"orly.dev/pkg/utils/chk"
)

type testSigner struct {
	*p256k.Signer
}

func newTestSigner() *testSigner {
	s := &p256k.Signer{}
	if err := s.Generate(); chk.E(err) {
		panic(err)
	}
	return &testSigner{Signer: s}
}

var _ signer.I = (*testSigner)(nil)
