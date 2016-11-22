package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"errors"
	"time"

	"fmt"

	"gopkg.in/codegangsta/cli.v1"
	"gopkg.in/dedis/cosi.v0/lib"
	s "gopkg.in/dedis/cosi.v0/service"
	"gopkg.in/dedis/cothority.v0/lib/config"
	"gopkg.in/dedis/cothority.v0/lib/crypto"
	"gopkg.in/dedis/cothority.v0/lib/dbg"
	"gopkg.in/dedis/cothority.v0/lib/network"
	"gopkg.in/dedis/cothority.v0/lib/sda"
	"gopkg.in/dedis/crypto.v0/abstract"
)

// checkConfig contacts all servers and verifies if it receives a valid
// signature from each.
func checkConfig(c *cli.Context) error {
	tomlFileName := c.String(optionGroup)
	f, err := os.Open(tomlFileName)
	printErrAndExit("Couldn't open group definition file: %v", err)
	el, descs, err := config.ReadGroupDescToml(f)
	printErrAndExit("Error while reading group definition file: %v", err)
	if len(el.List) == 0 {
		printErrAndExit("Empty entity or invalid group defintion in: %s",
			tomlFileName)
	}
	fmt.Println("[+] Checking the availability and responsiveness of the servers in the group...")
	// First check all servers individually
	for i := range el.List {
		checkList(sda.NewEntityList(el.List[i:i+1]), descs[i:i+1])
	}
	if len(el.List) > 1 {
		// Then check pairs of servers
		for i, first := range el.List {
			for j, second := range el.List[i+1:] {
				desc := []string{descs[i], descs[i+j+1]}
				es := []*network.Entity{first, second}
				checkList(sda.NewEntityList(es), desc)
				es[0], es[1] = es[1], es[0]
				desc[0], desc[1] = desc[1], desc[0]
				checkList(sda.NewEntityList(es), desc)
			}
		}
	}

	return nil
}

// checkList sends a message to the list and waits for the reply
func checkList(list *sda.EntityList, descs []string) {
	serverStr := ""
	for i, s := range list.List {
		name := strings.Split(descs[i], " ")[0]
		serverStr += fmt.Sprintf("%s_%s ", s.Addresses[0], name)
	}
	dbg.Lvl3("Sending message to: " + serverStr)
	msg := "verification"
	fmt.Print("[+] Checking server(s) ", serverStr, ": ")
	sig, err := signStatement(strings.NewReader(msg), list)
	if err != nil {
		fmt.Fprintln(os.Stderr,
			fmt.Sprintf("Error '%v'", err))
	} else {
		err := verifySignatureHash([]byte(msg), sig, list)
		if err != nil {
			fmt.Println(os.Stderr,
				fmt.Sprintf("Invalid signature: %v", err))
		} else {
			fmt.Println("Success")
		}
	}

}

// signFile will search for the file and sign it
// it always returns nil as an error
func signFile(c *cli.Context) error {
	if c.Args().First() == "" {
		printErrAndExit("Please give the file to sign", 1)
	}
	fileName := c.Args().First()
	groupToml := c.String(optionGroup)
	file, err := os.Open(fileName)
	if err != nil {
		printErrAndExit("Couldn't read file to be signed: %v", err)
	}
	sig, err := sign(file, groupToml)
	printErrAndExit("Couldn't create signature: %v", err)
	dbg.Lvl3(sig)
	var outFile *os.File
	outFileName := c.String("out")
	if outFileName != "" {
		outFile, err = os.Create(outFileName)
		printErrAndExit("Couldn't create signature file: %v", err)
	} else {
		outFile = os.Stdout
	}
	writeSigAsJSON(sig, outFile)
	if outFileName != "" {
		dbg.Lvl2("Signature written to: %s", outFile.Name())
	} // else keep the Stdout empty
	return nil
}

func verifyFile(c *cli.Context) error {
	if len(c.Args().First()) == 0 {
		printErrAndExit("Please give the 'msgFile'", 1)
	}
	dbg.SetDebugVisible(c.GlobalInt("debug"))
	sigOrEmpty := c.String("signature")
	err := verify(c.Args().First(), sigOrEmpty, c.String(optionGroup))
	verifyPrintResult(err)
	return nil
}

// verifyPrintResult prints out OK or what failed.
func verifyPrintResult(err error) {
	if err == nil {
		fmt.Println("[+] OK: Signature is valid.")
	} else {
		printErrAndExit("Invalid: Signature verification failed: %v", err)
	}
}

// writeSigAsJSON - writes the JSON out to a file
func writeSigAsJSON(res *s.SignatureResponse, outW io.Writer) {
	b, err := json.Marshal(res)
	if err != nil {
		printErrAndExit("Couldn't encode signature: %v", err)
	}
	var out bytes.Buffer
	json.Indent(&out, b, "", "\t")
	outW.Write([]byte("\n"))
	if _, err := out.WriteTo(outW); err != nil {
		printErrAndExit("Couldn't write signature: %v", err)
	}
	outW.Write([]byte("\n"))
}

func printErrAndExit(format string, a ...interface{}) {
	if len(a) > 0 && a[0] != nil {
		fmt.Fprintln(os.Stderr, "[-] "+fmt.Sprintf(format, a...))
		os.Exit(1)
	}
}

// sign takes a stream and a toml file defining the servers
func sign(r io.Reader, tomlFileName string) (*s.SignatureResponse, error) {
	dbg.Lvl2("Starting signature")
	f, err := os.Open(tomlFileName)
	if err != nil {
		return nil, err
	}
	el, err := config.ReadGroupToml(f)
	if err != nil {
		return nil, err
	}
	if len(el.List) <= 0 {
		return nil, errors.New("Empty or invalid cosi group file:" +
			tomlFileName)
	}
	dbg.Lvl2("Sending signature to", el)
	res, err := signStatement(r, el)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// signStatement can be used to sign the contents passed in the io.Reader
// (pass an io.File or use an strings.NewReader for strings)
func signStatement(read io.Reader, el *sda.EntityList) (*s.SignatureResponse,
	error) {
	publics := entityListToPublics(el)
	client := s.NewClient()
	msg, _ := crypto.HashStream(network.Suite.Hash(), read)

	pchan := make(chan *s.SignatureResponse)
	var err error
	go func() {
		dbg.Lvl3("Waiting for the response on SignRequest")
		response, e := client.SignMsg(el, msg)
		if e != nil {
			err = e
			close(pchan)
			return
		}
		pchan <- response
	}()

	select {
	case response, ok := <-pchan:
		dbg.Lvl5("Response:", response)
		if !ok || err != nil {
			return nil, errors.New("Received an invalid repsonse.")
		}

		err = cosi.VerifySignature(network.Suite, publics, msg, response.Signature)
		if err != nil {
			return nil, err
		}
		return response, nil
	case <-time.After(RequestTimeOut):
		return nil, errors.New("timeout on signing request")
	}
}

// verify takes a file and a group-definition, calls the signature
// verification and prints the result. If sigFileName is empty it
// assumes to find the standard signature in fileName.sig
func verify(fileName, sigFileName, groupToml string) error {
	// if the file hash matches the one in the signature
	dbg.Lvl4("Reading file " + fileName)
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return errors.New("Couldn't open msgFile: " + err.Error())
	}
	// Read the JSON signature file
	dbg.Lvl4("Reading signature")
	var sigBytes []byte
	if sigFileName == "" {
		fmt.Println("[+] Reading signature from standard input ...")
		sigBytes, err = ioutil.ReadAll(os.Stdin)
	} else {
		sigBytes, err = ioutil.ReadFile(sigFileName)
	}
	if err != nil {
		return err
	}
	sig := &s.SignatureResponse{}
	dbg.Lvl4("Unmarshalling signature ")
	if err := json.Unmarshal(sigBytes, sig); err != nil {
		return err
	}
	fGroup, err := os.Open(groupToml)
	if err != nil {
		return err
	}
	dbg.Lvl4("Reading group definition")
	el, err := config.ReadGroupToml(fGroup)
	if err != nil {
		return err
	}
	dbg.Lvl4("Verfifying signature")
	err = verifySignatureHash(b, sig, el)
	return err
}

func verifySignatureHash(b []byte, sig *s.SignatureResponse, el *sda.EntityList) error {
	// We have to hash twice, as the hash in the signature is the hash of the
	// message sent to be signed
	publics := entityListToPublics(el)
	fHash, _ := crypto.HashBytes(network.Suite.Hash(), b)
	hashHash, _ := crypto.HashBytes(network.Suite.Hash(), fHash)
	if !bytes.Equal(hashHash, sig.Sum) {
		return errors.New("You are trying to verify a signature " +
			"belonging to another file. (The hash provided by the signature " +
			"doesn't match with the hash of the file.)")
	}
	if err := cosi.VerifySignature(network.Suite, publics, fHash, sig.Signature); err != nil {
		return errors.New("Invalid sig:" + err.Error())
	}
	return nil
}
func entityListToPublics(el *sda.EntityList) []abstract.Point {
	publics := make([]abstract.Point, len(el.List))
	for i, e := range el.List {
		publics[i] = e.Public
	}
	return publics
}
