// Package cosi implements a round of a Collective Signing protocol.
package protocol

import (
	"fmt"
	"sync"

	"github.com/dedis/cothority/lib/dbg"
	"github.com/dedis/cothority/lib/sda"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/cosi"
)

var Name = "CoSi"

func init() {
	sda.ProtocolRegisterName(Name, NewProtocolCosi)
}

// This Cosi protocol is the simplest version, the "vanilla" version with the
// four phases:
//  - Announcement
//  - Commitment
//  - Challenge
//  - Response

// ProtocolCosi is the main structure holding the round and the sda.Node.
type ProtocolCosi struct {
	// The node that represents us
	*sda.TreeNodeInstance
	// TreeNodeId cached
	treeNodeID sda.TreeNodeID
	// the cosi struct we use (since it is a cosi protocol)
	// Public because we will need it from other protocols.
	Cosi *cosi.CoSi
	// the message we want to sign typically given by the Root
	Message []byte
	// The channel waiting for Announcement message
	announce chan chanAnnouncement
	// the channel waiting for Commitment message
	commit chan chanCommitment
	// the channel waiting for Challenge message
	challenge chan chanChallenge
	// the channel waiting for Response message
	response chan chanResponse
	// the channel that indicates if we are finished or not
	done chan bool
	// temporary buffer of commitment messages
	tempCommitment []abstract.Point
	// lock associated
	tempCommitLock *sync.Mutex
	// temporary buffer of Response messages
	tempResponse []abstract.Secret
	// lock associated
	tempResponseLock *sync.Mutex
	DoneCallback     func(sig []byte)

	// hooks related to the various phase of the protocol.
	// XXX NOT DEPLOYED YET / NOT IN USE.
	// announcement hook
	announcementHook AnnouncementHook
	commitmentHook   CommitmentHook
	challengeHook    ChallengeHook
}

// AnnouncementHook allows for handling what should happen upon an
// announcement
type AnnouncementHook func() error

// CommitmentHook allows for handling what should happen when a
// commitment is received
type CommitmentHook func(in []abstract.Point) error

// ChallengeHook allows for handling what should happen when a
// challenge is received
type ChallengeHook func(ch abstract.Secret) error

// NewProtocolCosi returns a ProtocolCosi with the node set with the right channels.
// Use this function like this:
// ```
// round := NewRound****()
// fn := func(n *sda.Node) sda.ProtocolInstance {
//      pc := NewProtocolCosi(round,n)
//		return pc
// }
// sda.RegisterNewProtocolName("cothority",fn)
// ```
func NewProtocolCosi(node *sda.TreeNodeInstance) (sda.ProtocolInstance, error) {
	var err error
	// XXX just need to take care to take the global list of cosigners once we
	// do the exception stuff
	publics := make([]abstract.Point, len(node.EntityList().List))
	for i, e := range node.EntityList().List {
		publics[i] = e.Public
	}

	pc := &ProtocolCosi{
		Cosi:             cosi.NewCosi(node.Suite(), node.Private(), publics),
		TreeNodeInstance: node,
		done:             make(chan bool),
		tempCommitLock:   new(sync.Mutex),
		tempResponseLock: new(sync.Mutex),
	}
	// Register the channels we want to register and listens on

	if err := node.RegisterChannel(&pc.announce); err != nil {
		return pc, err
	}
	if err := node.RegisterChannel(&pc.commit); err != nil {
		return pc, err
	}
	if err := node.RegisterChannel(&pc.challenge); err != nil {
		return pc, err
	}
	if err := node.RegisterChannel(&pc.response); err != nil {
		return pc, err
	}

	return pc, err
}

// Start will call the announcement function of its inner Round structure. It
// will pass nil as *in* message.
func (pc *ProtocolCosi) Start() error {
	return pc.StartAnnouncement()
}

// Dispatch will listen on the four channels we use (i.e. four steps)
func (pc *ProtocolCosi) Dispatch() error {
	for {
		var err error
		select {
		case packet := <-pc.announce:
			err = pc.handleAnnouncement(&packet.Announcement)
		case packet := <-pc.commit:
			err = pc.handleCommitment(&packet.Commitment)
		case packet := <-pc.challenge:
			err = pc.handleChallenge(&packet.Challenge)
		case packet := <-pc.response:
			err = pc.handleResponse(&packet.Response)
		case <-pc.done:
			return nil
		}
		if err != nil {
			dbg.Error("ProtocolCosi -> err treating incoming:", err)
		}
	}
}

// StartAnnouncement will start a new announcement.
func (pc *ProtocolCosi) StartAnnouncement() error {
	dbg.Lvl3(pc.Name(), "Message:", pc.Message)
	out := &Announcement{
		From: pc.treeNodeID,
	}

	return pc.handleAnnouncement(out)
}

// handleAnnouncement will pass the message to the round and send back the
// output. If in == nil, we are root and we start the round.
func (pc *ProtocolCosi) handleAnnouncement(in *Announcement) error {
	dbg.Lvl3("Message:", pc.Message)
	// If we have a hook on announcement call the hook
	if pc.announcementHook != nil {
		return pc.announcementHook()
	}

	// If we are leaf, we should go to commitment
	if pc.IsLeaf() {
		return pc.handleCommitment(nil)
	}
	out := &Announcement{
		From: pc.treeNodeID,
	}

	// send the output to children
	return pc.sendAnnouncement(out)
}

// sendAnnouncement simply send the announcement to every children
func (pc *ProtocolCosi) sendAnnouncement(ann *Announcement) error {
	var err error
	for _, tn := range pc.Children() {
		// still try to send to everyone
		err = pc.SendTo(tn, ann)
	}
	return err
}

// handleAllCommitment takes the full set of messages from the children and passes
// it to the parent
func (pc *ProtocolCosi) handleCommitment(in *Commitment) error {
	if !pc.IsLeaf() {
		// add to temporary
		pc.tempCommitLock.Lock()
		pc.tempCommitment = append(pc.tempCommitment, in.Comm)
		pc.tempCommitLock.Unlock()
		// do we have enough ?
		// TODO: exception mechanism will be put into another protocol
		if len(pc.tempCommitment) < len(pc.Children()) {
			return nil
		}
	}
	dbg.Lvl3(pc.Name(), "aggregated")
	// pass it to the hook
	if pc.commitmentHook != nil {
		return pc.commitmentHook(pc.tempCommitment)
	}

	// go to Commit()
	out := pc.Cosi.Commit(nil, pc.tempCommitment)

	// if we are the root, we need to start the Challenge
	if pc.IsRoot() {
		return pc.StartChallenge()
	}

	// otherwise send it to parent
	outMsg := &Commitment{
		Comm: out,
	}
	return pc.SendTo(pc.Parent(), outMsg)
}

// StartChallenge start the challenge phase. Typically called by the Root ;)
func (pc *ProtocolCosi) StartChallenge() error {
	challenge, err := pc.Cosi.CreateChallenge(pc.Message)
	if err != nil {
		return err
	}
	out := &Challenge{
		Chall: challenge,
	}
	dbg.Lvl3(pc.Name(), "Starting Chal=", fmt.Sprintf("%+v", challenge), " (message =", string(pc.Message))
	return pc.handleChallenge(out)

}

// VerifySignature verifies if the challenge and the secret (from the response phase) form a
// correct signature for this message using the aggregated public key.
// This is copied from lib/cosi, so that you don't need to include both lib/cosi
// and protocols/cosi
func VerifySignature(suite abstract.Suite, publics []abstract.Point, msg, sig []byte) error {
	return cosi.VerifySignature(suite, publics, msg, sig)
}

// handleChallenge dispatch the challenge to the round and then dispatch the
// results down the tree.
func (pc *ProtocolCosi) handleChallenge(in *Challenge) error {
	// TODO check hook

	dbg.Lvl3(pc.Name(), "chal=", fmt.Sprintf("%+v", in.Chall))
	// else dispatch it to cosi
	pc.Cosi.Challenge(in.Chall)

	// if we are leaf, then go to response
	if pc.IsLeaf() {
		return pc.handleResponse(nil)
	}

	// otherwise send it to children
	return pc.sendChallenge(in)
}

// sendChallenge sends the challenge down the tree.
func (pc *ProtocolCosi) sendChallenge(out *Challenge) error {
	var err error
	for _, tn := range pc.Children() {
		err = pc.SendTo(tn, out)
	}
	return err

}

// handleResponse brings up the response of each node in the tree to the root.
func (pc *ProtocolCosi) handleResponse(in *Response) error {
	if !pc.IsLeaf() {
		// add to temporary
		pc.tempResponseLock.Lock()
		pc.tempResponse = append(pc.tempResponse, in.Resp)
		pc.tempResponseLock.Unlock()
		// do we have enough ?
		dbg.Lvl3(pc.Name(), "has", len(pc.tempResponse), "responses")
		if len(pc.tempResponse) < len(pc.Children()) {
			return nil
		}
	}
	defer pc.Cleanup()

	dbg.Lvl3(pc.Name(), "aggregated")
	outResponse, err := pc.Cosi.Response(pc.tempResponse)
	if err != nil {
		return err
	}

	// Simulation feature => time the verification process.
	if (VerifyResponse == 1 && pc.IsRoot()) || VerifyResponse == 2 {
		dbg.Lvl3(pc.Name(), "(root=", pc.IsRoot(), ") Doing Response verification", VerifyResponse)
		// verify the responses at each level with the aggregate
		// public key of this subtree.
		/*if err := pc.Cosi.VerifyResponses(pc); err != nil {*/
		//dbg.Error("Verification error", err)
		//return fmt.Errorf("%s Verifcation of responses failed:%s", pc.Name(), err)
		/*}*/
	} else {
		dbg.Lvl3(pc.Name(), "(root=", pc.IsRoot(), ") Skipping Response verification", VerifyResponse)
	}

	out := &Response{
		Resp: outResponse,
	}
	// send it back to parent
	if !pc.IsRoot() {
		return pc.SendTo(pc.Parent(), out)
	}
	return nil
}

// Cleanup closes the protocol and calls DoneCallback, if defined
func (pc *ProtocolCosi) Cleanup() {
	dbg.Lvl3(pc.Entity().First(), "Cleaning up")
	// if callback when finished
	if pc.DoneCallback != nil {
		dbg.Lvl3("Calling doneCallback")
		pc.DoneCallback(pc.Cosi.Signature())
	}
	close(pc.done)
	pc.Done()

}

// SigningMessage simply set the message to sign for this round
func (pc *ProtocolCosi) SigningMessage(msg []byte) {
	pc.Message = msg
	dbg.Lvl2(pc.Name(), "Root will sign message=", pc.Message)
}

// RegisterAnnouncementHook allows for handling what should happen upon an
// announcement
func (pc *ProtocolCosi) RegisterAnnouncementHook(fn AnnouncementHook) {
	pc.announcementHook = fn
}

// RegisterCommitmentHook allows for handling what should happen when a
// commitment is received
func (pc *ProtocolCosi) RegisterCommitmentHook(fn CommitmentHook) {
	pc.commitmentHook = fn
}

// RegisterChallengeHook allows for handling what should happen when a
// challenge is received
func (pc *ProtocolCosi) RegisterChallengeHook(fn ChallengeHook) {
	pc.challengeHook = fn
}

// RegisterDoneCallback allows for handling what should happen when a
// the protocol is done
func (pc *ProtocolCosi) RegisterDoneCallback(fn func(sig []byte)) {
	pc.DoneCallback = fn
}
