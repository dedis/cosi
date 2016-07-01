package cosi

import (
	"errors"

	"fmt"
	"time"

	libcosi "gopkg.in/dedis/cosi.v0/lib"
	"gopkg.in/dedis/cosi.v0/protocol"
	"gopkg.in/dedis/cothority.v0/lib/crypto"
	"gopkg.in/dedis/cothority.v0/lib/dbg"
	"gopkg.in/dedis/cothority.v0/lib/network"
	"gopkg.in/dedis/cothority.v0/lib/sda"
)

// This file contains all the code to run a CoSi service. It is used to reply to
// client request for signing something using CoSi.
// As a prototype, it just signs and returns. It would be very easy to write an
// updated version that chains all signatures for example.

// ServiceName is the name to refer to the CoSi service
const ServiceName = "CoSi"

func init() {
	sda.RegisterNewService(ServiceName, newCosiService)
	network.RegisterMessageType(&SignatureRequest{})
	network.RegisterMessageType(&SignatureResponse{})
}

// Cosi is the service that handles collective signing operations
type Cosi struct {
	*sda.ServiceProcessor
	path string
}

// SignatureRequest is what the Cosi service is expected to receive from clients.
type SignatureRequest struct {
	Message    []byte
	EntityList *sda.EntityList
}

// CosiRequestType is the type that is embedded in the Request object for a
// CosiRequest
var CosiRequestType = network.RegisterMessageType(SignatureRequest{})

// SignatureResponse is what the Cosi service will reply to clients.
type SignatureResponse struct {
	Sum       []byte
	Signature []byte
}

// CosiResponseType is the type that is embedded in the Request object for a
// CosiResponse
var CosiResponseType = network.RegisterMessageType(SignatureResponse{})

// SignatureRequest treats external request to this service.
func (cs *Cosi) SignatureRequest(e *network.Entity, req *SignatureRequest) (network.ProtocolMessage, error) {
	tree := req.EntityList.GenerateBinaryTree()
	tni := cs.NewTreeNodeInstance(tree, tree.Root)
	pi, err := cosi.NewProtocolCosi(tni)
	if err != nil {
		return nil, errors.New("Couldn't make new protocol: " + err.Error())
	}
	cs.RegisterProtocolInstance(pi)
	pcosi := pi.(*cosi.ProtocolCosi)
	pcosi.SigningMessage(req.Message)
	h, err := crypto.HashBytes(network.Suite.Hash(), req.Message)
	if err != nil {
		return nil, errors.New("Couldn't hash message: " + err.Error())
	}
	response := make(chan *libcosi.Signature)
	pcosi.RegisterDoneCallback(func(sig []byte) {
		response <- &libcosi.Signature{
			Sig: sig,
		}
	})
	dbg.Lvl3("Cosi Service starting up root protocol")
	go pi.Dispatch()
	go pi.Start()
	sig := <-response
	if dbg.DebugVisible() > 0 {
		fmt.Printf("%s: Signed a message.\n", time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
	}
	return &SignatureResponse{
		Sum:       h,
		Signature: sig.Sig,
	}, nil
}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
func (cs *Cosi) NewProtocol(tn *sda.TreeNodeInstance, conf *sda.GenericConfig) (sda.ProtocolInstance, error) {
	dbg.Lvl3("Cosi Service received New Protocol event")
	pi, err := cosi.NewProtocolCosi(tn)
	go pi.Dispatch()
	return pi, err
}

func newCosiService(c sda.Context, path string) sda.Service {
	s := &Cosi{
		ServiceProcessor: sda.NewServiceProcessor(c),
		path:             path,
	}
	err := s.RegisterMessage(s.SignatureRequest)
	if err != nil {
		dbg.ErrFatal(err, "Couldn't register message:")
	}
	return s
}
