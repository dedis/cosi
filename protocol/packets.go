package protocol

import (
	"github.com/dedis/cothority/lib/sda"
	"github.com/dedis/crypto/abstract"
)

// This files defines the structure we use for registering to the channels by
// SDA.

// The main messages used by CoSi

// VerifyResponse sets how the checks are done,
// see https://github.com/dedis/cothority/issues/260
// 0 - no check at all
// 1 - check only at root
// 2 - check at each level of the tree
var VerifyResponse = 1

// Announcement is broadcasted message initiated and signed by proposer.
type Announcement struct {
}

// Commitment of all nodes together with the data they want
// to have signed
type Commitment struct {
	Comm abstract.Point
}

// Challenge is the challenge computed by the root-node.
type Challenge struct {
	Chall abstract.Secret
}

// Response with which every node replies with.
type Response struct {
	Resp abstract.Secret
}

//Theses are pairs of TreeNode + the actual message we want to listen on.
type chanAnnouncement struct {
	*sda.TreeNode
	Announcement
}

type chanCommitment struct {
	*sda.TreeNode
	Commitment
}

type chanChallenge struct {
	*sda.TreeNode
	Challenge
}

type chanResponse struct {
	*sda.TreeNode
	Response
}
