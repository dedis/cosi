package cosi

import (
	"testing"

	"sync"

	"github.com/dedis/cosi/lib"
	"github.com/dedis/cothority/lib/network"
	"github.com/stretchr/testify/assert"
	"gopkg.in/dedis/cothority.v0/lib/dbg"
	"gopkg.in/dedis/cothority.v0/lib/sda"
)

func TestMain(m *testing.M) {
	dbg.MainTest(m)
}

func TestServiceCosi(t *testing.T) {
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

func TestParallel(t *testing.T) {
	local := sda.NewLocalTest()
	nbrHosts := 2
	nbrParallel := 2
	// generate nbrHosts hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, el, _ := local.GenTree(nbrHosts, false, true, false)
	formatEntityList(local, hosts, el)
	defer local.CloseAll()

	// Send a request to the service
	client := NewClient()
	msg := []byte("hello cosi service")
	dbg.Lvl1("Sending request to service...")
	var wg sync.WaitGroup
	wg.Add(nbrParallel)
	for i := 0; i < nbrParallel; i++ {
		go func(i int) {
			dbg.Lvl1("Starting", i)
			res, err := client.SignMsg(el, msg)
			dbg.ErrFatal(err, "Couldn't send")

			// verify the response still
			assert.Nil(t, cosi.VerifySignature(hosts[0].Suite(), el.Publics(),
				msg, res.Signature))
			wg.Done()
			dbg.Lvl1("Done", i)
		}(i)
	}
	wg.Wait()
}

func formatEntityList(local *sda.LocalTest, h []*sda.Host, el *sda.EntityList) {
	for i := range el.List {
		priv := local.GetPrivate(h[i])
		el.List[i].Public = cosi.Ed25519Public(network.Suite, priv)
	}
}
