package cosi

import (
	"testing"

	"gopkg.in/dedis/cosi.v0/lib"
	"github.com/dedis/cothority/lib/network"
	"github.com/stretchr/testify/assert"
	"gopkg.in/dedis/cothority.v0/lib/dbg"
	"gopkg.in/dedis/cothority.v0/lib/sda"
)

func TestServiceCosi(t *testing.T) {
	defer dbg.AfterTest(t)
	dbg.TestOutput(testing.Verbose(), 4)
	local := sda.NewLocalTest()
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, el, _ := local.GenTree(5, false, true, false)
	formatEntityList(local, hosts, el)
	defer local.CloseAll()

	// Send a request to the service
	client := NewClient()
	msg := []byte("hello cosi service")
	dbg.Lvl1("Sending request to service...")
	res, err := client.SignMsg(el, msg)
	dbg.ErrFatal(err, "Couldn't send")

	// verify the response still
	assert.Nil(t, cosi.VerifySignature(hosts[0].Suite(), el.Publics(),
		msg, res.Signature))
}

func formatEntityList(local *sda.LocalTest, h []*sda.Host, el *sda.EntityList) {
	for i := range el.List {
		priv := local.GetPrivate(h[i])
		el.List[i].Public = cosi.Ed25519Public(network.Suite, priv)
	}
}
