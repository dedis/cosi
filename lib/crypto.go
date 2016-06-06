package cosi

import (
	"crypto/sha512"
	"math/big"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/nist"
	"gopkg.in/dedis/crypto.v0/util"
)

func SecretToSlice(secret abstract.Secret) []byte {
	i := secret.(*nist.Int)
	min := 32
	max := 32
	act := i.MarshalSize()
	vSize := len(i.V.Bytes())
	if vSize < act {
		act = vSize
	}
	pad := act
	if pad < min {
		pad = min
	}
	if max != 0 && pad > max {
		panic("Int not representable in max bytes")
	}
	buf := make([]byte, pad)
	util.Reverse(buf[:act], i.V.Bytes())
	return buf
	/*secBuff := make([]byte, 32)*/
	//vBuff := secret.(*nist.Int).V.Bytes()
	//util.Reverse(secBuff[32-len(vBuff):], vBuff)
	/*return secBuff*/
	//return secret.(*nist.Int).LittleEndian(32, 32)
}

func sliceToSecret(suite abstract.Suite, buffer []byte) abstract.Secret {
	s := suite.Secret().(*nist.Int)
	s.SetLittleEndian(buffer)
	return s
}

func sliceToPoint(suite abstract.Suite, buffer []byte) abstract.Point {
	point := suite.Point()
	if err := point.UnmarshalBinary(buffer); err != nil {
		panic(err)
	}
	return point
}

// Ed25519ToPublic will transform a ed25519 scalar to a ed25519 public key using
// the digest + prune transofrmation
func Ed25519Public(suite abstract.Suite, s abstract.Secret) abstract.Point {
	// secret modulo-d
	//secMarshal := s.(*nist.Int).LittleEndian(32, 32)
	secMarshal := SecretToSlice(s)
	pruned := sha512.Sum512(secMarshal)
	pruned[0] &= 248
	pruned[31] &= 127
	pruned[31] |= 64

	// go back to secret, now formatted as ed25519
	//secPruned := SliceToInt(suite, pruned)
	base := big.NewInt(2)
	exp := big.NewInt(256)
	modulo := big.NewInt(0).Exp(base, exp, nil)
	modulo.Sub(modulo, big.NewInt(1))
	secPruned := nist.NewInt(0, modulo)
	secPruned.SetLittleEndian(pruned[:32])
	return suite.Point().Mul(nil, secPruned)
}

func SumPublics(suite abstract.Suite, publics []abstract.Point) abstract.Point {
	agg := suite.Point().Null()
	for _, p := range publics {
		agg = agg.Add(agg, p)
	}
	return agg
}
