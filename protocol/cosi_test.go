package cosi

import (
	"testing"
	"time"

	"gopkg.in/dedis/cosi.v0/lib"
	"gopkg.in/dedis/cothority.v0/lib/dbg"
	"gopkg.in/dedis/cothority.v0/lib/network"
	"gopkg.in/dedis/cothority.v0/lib/sda"
)

func TestCosi(t *testing.T) {
	//defer dbg.AfterTest(t)
	dbg.TestOutput(testing.Verbose(), 4)
	for _, nbrHosts := range []int{1, 3, 13} {
		dbg.Lvl2("Running cosi with", nbrHosts, "hosts")
		local := sda.NewLocalTest()
		hosts, el, tree := local.GenBigTree(nbrHosts, nbrHosts, 3, true, true)
		formatEntityList(local, hosts, el)
		aggPublic := network.Suite.Point().Null()
		for _, e := range el.List {
			aggPublic = aggPublic.Add(aggPublic, e.Public)
		}

		done := make(chan bool)
		// create the message we want to sign for this round
		msg := []byte("Hello World Cosi")

		// Register the function generating the protocol instance
		var root *ProtocolCosi
		// function that will be called when protocol is finished by the root
		doneFunc := func(sig []byte) {
			suite := hosts[0].Suite()
			publics := el.Publics()
			if err := root.Cosi.VerifyResponses(aggPublic); err != nil {
				t.Fatal("Error verifying responses", err)
			}
			if err := cosi.VerifySignature(suite, publics, msg, sig); err != nil {
				t.Fatal("Error verifying signature:", err)
			}
			done <- true
		}

		// Start the protocol
		p, err := local.CreateProtocol("CoSi", tree)
		if err != nil {
			t.Fatal("Couldn't create new node:", err)
		}
		root = p.(*ProtocolCosi)
		root.Message = msg
		root.RegisterDoneCallback(doneFunc)
		go root.StartProtocol()
		select {
		case <-done:
		case <-time.After(time.Second * 2):
			t.Fatal("Could not get signature verification done in time")
		}
		local.CloseAll()
	}
}

func formatEntityList(local *sda.LocalTest, h []*sda.Host, el *sda.EntityList) {
	for i := range el.List {
		priv := local.GetPrivate(h[i])
		el.List[i].Public = cosi.Ed25519Public(network.Suite, priv)
	}
}
