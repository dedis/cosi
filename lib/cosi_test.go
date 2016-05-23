package cosi

import (
	"fmt"
	"testing"

	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/config"
	"github.com/dedis/crypto/edwards"
	// XXX In order to check the compatibility with the bford fork of
	// golang-x-crypto we do a comparison in the test. Uncomment if you want to
	// try.
	/* "github.com/bford/golang-x-crypto/ed25519"*/ //"github.com/bford/golang-x-crypto/ed25519/cosi"
)

var testSuite = edwards.NewAES128SHA256Ed25519(false)

// TestCosiCommitment test if the commitment generation is correct
func TestCosiCommitment(t *testing.T) {
	var length = 5
	cosis := genCosis(length)
	// gen commitments from children
	commitments := genCommitments(cosis[1:])
	root := cosis[0]
	root.Commit(nil, commitments)
	// compute the aggregate commitment ourself...
	aggCommit := testSuite.Point().Null()
	// add commitment of children
	for _, com := range commitments {
		aggCommit = aggCommit.Add(aggCommit, com.Commitment)
	}
	// add commitment of root
	aggCommit = aggCommit.Add(aggCommit, root.commitment)
	if !aggCommit.Equal(root.aggregateCommitment) {
		t.Fatal("Aggregate Commitment are not equal")
	}
}

func TestCosiChallenge(t *testing.T) {
	root, children := genPostCommitmentPhaseCosi(5)
	msg := []byte("Hello World Cosi\n")
	chal, err := root.CreateChallenge(msg)
	if err != nil {
		t.Fatal("Error during challenge generation")
	}
	for _, child := range children {
		child.Challenge(chal)
		if !child.challenge.Equal(chal.Challenge) {
			t.Fatal("Error during challenge on children")
		}
	}
}

// TestCosiResponse will test wether the response generation is correct or not
func TestCosiResponse(t *testing.T) {
	msg := []byte("Hello World Cosi")
	// go to the challenge phase
	root, children := genPostChallengePhaseCosi(5, msg)
	var responses []*Response

	// for verification later
	aggResponse := testSuite.Secret().Zero()
	for _, ch := range children {
		// generate the response of each children
		r, err := ch.CreateResponse()
		if err != nil {
			t.Fatal("Error creating response:", err)
		}
		responses = append(responses, r)
		aggResponse = aggResponse.Add(aggResponse, r.Response)
	}
	// pass them up to the root
	_, err := root.Response(responses)
	if err != nil {
		t.Fatal("Response phase failed:", err)
	}

	// verify it
	aggResponse = aggResponse.Add(aggResponse, root.response)
	if !aggResponse.Equal(root.aggregateResponse) {
		t.Fatal("Responses aggregated not equal")
	}
}

// TestCosiSignature test if the signature generation is correct,i.e. if we
// can verify the final signature.
func TestCosiSignature(t *testing.T) {
	msg := []byte("Hello World Cosi")
	nb := 2
	root, children, err := genFinalCosi(nb, msg)
	if err != nil {
		t.Fatal(err)
	}
	var publics []abstract.Point
	// add root public key
	rootPublic := Ed25519Public(testSuite, root.private)
	//fmt.Println("TestCosi genPublic root : ", own.Abstract2Hex(rootPublic))
	publics = append(publics, rootPublic)
	for _, ch := range children {
		// add children public key
		public := Ed25519Public(testSuite, ch.private)
		publics = append(publics, public)
		//fmt.Println("TestCosi genPublic : ", own.Abstract2Hex(public))
	}
	sig := root.Signature()

	if err := VerifySignature(testSuite, publics, msg, sig); err != nil {
		t.Fatal("Error veriying:", err)
	}
}

func genKeyPair(nb int) ([]*config.KeyPair, []abstract.Point) {
	var kps []*config.KeyPair
	var publics []abstract.Point
	for i := 0; i < nb; i++ {
		kp := config.NewKeyPair(testSuite)
		kp.Public = Ed25519Public(testSuite, kp.Secret)
		kps = append(kps, kp)
		publics = append(publics, kp.Public)
		//fmt.Println("genKeyPair: genPublic: ", own.Abstract2Hex(kp.Public))
	}
	return kps, publics
}

func genCosis(nb int) []*Cosi {
	kps, publics := genKeyPair(nb)
	var cosis []*Cosi
	for _, kp := range kps {
		cosis = append(cosis, NewCosi(testSuite, kp.Secret, publics))
	}
	return cosis
}

func genCommitments(cosis []*Cosi) []*Commitment {
	commitments := make([]*Commitment, len(cosis))
	for i := range cosis {
		commitments[i] = cosis[i].CreateCommitment(nil)
	}
	return commitments
}

// genPostCommitmentPhaseCosi returns the Root and its Children Cosi. They have
// already made the Commitment phase.
func genPostCommitmentPhaseCosi(nb int) (*Cosi, []*Cosi) {
	cosis := genCosis(nb)
	commitments := genCommitments(cosis[1:])
	root := cosis[0]
	root.Commit(nil, commitments)
	return root, cosis[1:]
}

func genPostChallengePhaseCosi(nb int, msg []byte) (*Cosi, []*Cosi) {
	r, children := genPostCommitmentPhaseCosi(nb)
	chal, _ := r.CreateChallenge(msg)
	for _, ch := range children {
		ch.Challenge(chal)
	}
	return r, children
}

func genFinalCosi(nb int, msg []byte) (*Cosi, []*Cosi, error) {
	// go to the challenge phase
	root, children := genPostChallengePhaseCosi(nb, msg)
	var responses []*Response
	for _, ch := range children {
		resp, err := ch.CreateResponse()
		if err != nil {
			panic("Aie")
		}
		responses = append(responses, resp)
	}
	// pass them up to the root
	_, err := root.Response(responses)
	if err != nil {
		return nil, nil, fmt.Errorf("Response phase failed:%v", err)
	}
	return root, children, nil
}

/*func TestCosiEd25519(t *testing.T) {*/
//suite := testSuite
//msg := []byte("Hello World")
//// Create keypairs for the two cosigners.
//_, priKey1, _ := ed25519.GenerateKey(nil)
//_, priKey2, _ := ed25519.GenerateKey(nil)
//// XXX NOTE XXX : Modified version where we take the module version of the key
//// AS IS for the private key
//// Reason: abstract.Secret is already modulo, can't expand it again.
//privKey1Modulo := own.Modulo(suite, priKey1)
//privKey2Modulo := own.Modulo(suite, priKey2)
//pubKey1 := own.Ed25519ScalarToPublic(privKey1Modulo)
//pubKey2 := own.Ed25519ScalarToPublic(privKey2Modulo)
//// Extend the privKey for giving it to ed25519
//var privKey1ModuloExtended = own.ReducedScalarToExtended(privKey1Modulo, pubKey1)
//var privKey2ModuloExtended = own.ReducedScalarToExtended(privKey2Modulo, pubKey2)

//pubKeys := []ed25519.PublicKey{pubKey1, pubKey2}
//// get the equivalent to abstract.Secret
//priKey1Int := own.SliceToInt(suite, privKey1Modulo)
//priKey2Int := own.SliceToInt(suite, privKey2Modulo)
//priKey1IntPruned := own.Ed25519ScalarToSecret(suite, privKey1Modulo)
//priKey2IntPruned := own.Ed25519ScalarToSecret(suite, privKey2Modulo)
//// get the key into abstract.Secret/Point form
//abPubKey1 := suite.Point().Mul(nil, priKey1IntPruned)
//abPubKey2 := suite.Point().Mul(nil, priKey2IntPruned)
//aggPublic := suite.Point().Add(abPubKey1, abPubKey2)
//abPubKeys := []abstract.Point{abPubKey1, abPubKey2}

//fmt.Println("----------------- Public Keys -----------------\n")
//fmt.Println("Abstract Pub 1 = ", own.Abstract2Hex(abPubKey1))
//fmt.Println("Abstract Pub 2 = ", own.Abstract2Hex(abPubKey2))
//fmt.Println("Abstract Pub Agg = ", own.Abstract2Hex(aggPublic))
//fmt.Println("Ed25519 Pub 1  = ", hex.EncodeToString(pubKey1))
//fmt.Println("Ed25519 Pub 2  = ", hex.EncodeToString(pubKey2))
////fmt.Println("Ed25519 Pub  Agg = ",

//fmt.Println("\n---------------- Sign Ed25519 -----------------\n")
//sigEd25519 := SignEd25519(msg, pubKeys, privKey1ModuloExtended, privKey2ModuloExtended)

//fmt.Println("\n---------------- Sign Abstract ----------------\n")
//sigAbstract := SignAbstract(suite, msg, abPubKeys, priKey1Int, priKey2Int)

//fmt.Println("\n\n------------------- 1- Ed25519.Verify(Ed25519 Sig) -----------\n")
//b := cosi.Verify(pubKeys, nil, msg, sigEd25519)
//fmt.Println(" => valid ? ", b)
//fmt.Println("\n------------------- 1- Ed25519.Verify(Abstract Sig) -----------\n")
//b = cosi.Verify(pubKeys, nil, msg, sigAbstract)
//fmt.Println(" => valid ? ", b)
//fmt.Println("\n------------------- 2- Abstract.Verify --------------\n")
//err := VerifySignature(suite, abPubKeys, msg, sigAbstract)
//fmt.Println(" => valid ? ", err == nil)

//}

//// XXX TESTING XXX
//var SEED1 []byte
//var SEED2 []byte

//func init() {
//SEED1, _ = hex.DecodeString("3aed8a2f6ca4c385ad90dbebcfef29ceaea9e2df09530399dc82245c96d643945da80212409bad9c4af7511fdc5caf8fe196ff669cbb51334c4070d8e798df0a")
//SEED2, _ = hex.DecodeString("4afcd0cc48d60d94db58fbc5de2261513750b10e3a5f0c8cec2978f6d2c008b6d182674965dbff66725f472cd10d9ba82d13228af96e4636ff0faf5882eb8504")
//}

//func SignAbstract(suite abstract.Suite, msg []byte, keys []abstract.Point, pri1, pri2 abstract.Secret) []byte {
//// create the two cosi structs
//cosi1 := NewCosi(suite, pri1, keys)
//cosi2 := NewCosi(suite, pri2, keys)
//commit1 := cosi1.CreateCommitment(bytes.NewReader(SEED1))
//commit2 := cosi2.Commit(bytes.NewReader(SEED2), []*Commitment{commit1})

//fmt.Println("Abstract Sign Secret 1 = ", own.Abstract2Hex(cosi1.random))
//fmt.Println("Abstract Sign Commit 1 = ", own.Abstract2Hex(commit1.Commitment))
//fmt.Println("Abstract Sign Secret 2 = ", own.Abstract2Hex(cosi2.random))
//fmt.Println("Abstract Sign Commit 2 = ", own.Abstract2Hex(commit2.Commitment))

//challenge2, _ := cosi2.CreateChallenge(msg)
//challenge1 := cosi1.Challenge(challenge2)
//fmt.Println("Abstract Sign Challenge 1 = ", own.Abstract2Hex(challenge1.Challenge))
//fmt.Println("Abstract Sign Challenge 2 = ", own.Abstract2Hex(challenge2.Challenge))

//resp1, _ := cosi1.CreateResponse()
//resp2, _ := cosi2.Response([]*Response{resp1})
//fmt.Println("Abstract Sign Response 1 = ", own.Abstract2Hex(resp1.Response))
//fmt.Println("Abstract Sign Response 2 = ", own.Abstract2Hex(resp2.Response))
//fmt.Println("Abstract Sign AggResponse  = ", own.Abstract2Hex(cosi2.aggregateResponse))

//fmt.Println("Abstract Sign signature = ", hex.EncodeToString(cosi2.Signature()))
//// create the challenge
//return cosi2.Signature()
//}

//// Helper function to implement a bare-bones cosigning process.
//// In practice the two cosigners would be on different machines
//// ideally managed by independent badministrators or key-holders.
//func SignEd25519(message []byte, pubKeys []ed25519.PublicKey,
//priKey1, priKey2 ed25519.PrivateKey) []byte {

//// Each cosigner first needs to produce a per-message commit.
//commit1, secret1, _ := cosi.Commit(bytes.NewReader(SEED1))
//commit2, secret2, _ := cosi.Commit(bytes.NewReader(SEED2))
//commits := []cosi.Commitment{commit1, commit2}
//fmt.Println("Ed25519 Sign Secret1 = ", hex.EncodeToString(secret1.Reduced()))
//fmt.Println("Ed25519 Sign Commit1 = ", hex.EncodeToString(commit1))
//fmt.Println("Ed25519 Sign Secret2 = ", hex.EncodeToString(secret2.Reduced()))
//fmt.Println("Ed25519 Sign Commit2 = ", hex.EncodeToString(commit2))

//// The leader then combines these into msg an aggregate commit.
//cosigners := cosi.NewCosigners(pubKeys, nil)
//aggregatePublicKey := cosigners.AggregatePublicKey()
//aggregateCommit := cosigners.AggregateCommit(commits)
//// The cosigners now produce their parts of the collective signature.
//fmt.Println("------------------ Cosign Ed25519 1 ------------")
//sigPart1 := cosi.Cosign(priKey1, secret1, message, aggregatePublicKey, aggregateCommit)
//fmt.Println("------------------ Cosign Ed25519 2 ------------")
//sigPart2 := cosi.Cosign(priKey2, secret2, message, aggregatePublicKey, aggregateCommit)
//sigParts := []cosi.SignaturePart{sigPart1, sigPart2}
//fmt.Println("------------------ Aggregate Ed25519 -------------")
//fmt.Println("Ed25519 Sign Aggregate = ", hex.EncodeToString(aggregatePublicKey))
//fmt.Println("Ed25519 Sign AggCommit = ", hex.EncodeToString(aggregateCommit))

//// Finally, the leader combines the two signature parts
//// into a final collective signature.
//sig := cosigners.AggregateSignature(aggregateCommit, sigParts)
//fmt.Println("Ed25519 Sign signature = ", hex.EncodeToString(sig))
//return sig
//}
