package main

import (
	"lukechampine.com/frand"
	"orly.dev/pkg/interfaces/signer"
)

// testSigner is a simple signer implementation for benchmarking
type testSigner struct {
	pub []byte
	sec []byte
}

func newTestSigner() *testSigner {
	return &testSigner{
		pub: frand.Bytes(32),
		sec: frand.Bytes(32),
	}
}

func (s *testSigner) Pub() []byte {
	return s.pub
}

func (s *testSigner) Sec() []byte {
	return s.sec
}

func (s *testSigner) Sign(msg []byte) ([]byte, error) {
	return frand.Bytes(64), nil
}

func (s *testSigner) Verify(msg, sig []byte) (bool, error) {
	return true, nil
}

func (s *testSigner) InitSec(sec []byte) error {
	s.sec = sec
	s.pub = frand.Bytes(32)
	return nil
}

func (s *testSigner) InitPub(pub []byte) error {
	s.pub = pub
	return nil
}

func (s *testSigner) Zero() {
	for i := range s.sec {
		s.sec[i] = 0
	}
}


func (s *testSigner) ECDH(pubkey []byte) ([]byte, error) {
	return frand.Bytes(32), nil
}

func (s *testSigner) Generate() error {
	return nil
}

var _ signer.I = (*testSigner)(nil)