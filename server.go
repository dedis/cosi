package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"

	c "github.com/dedis/cothority/lib/config"
	"github.com/dedis/cothority/lib/crypto"
	"github.com/dedis/cothority/lib/dbg"
	"github.com/dedis/cothority/lib/network"
	"gopkg.in/codegangsta/cli.v1"
	// Empty imports to have the init-functions called which should
	// register the protocol

	_ "github.com/dedis/cosi/protocol/cosi"
	_ "github.com/dedis/cosi/service/cosi"
	"github.com/dedis/crypto/config"
)

func runServer(ctx *cli.Context) {
	// first check the options
	config := ctx.String("config")

	if _, err := os.Stat(config); os.IsNotExist(err) {
		dbg.Fatalf("[-] Configuration file does not exist. %s. "+
			"Use `cosi server setup` to create one.", config)
	}
	// Let's read the config
	_, host, err := c.ParseCothorityd(config)
	if err != nil {
		dbg.Fatal("Couldn't parse config:", err)
	}
	host.ListenAndBind()
	host.StartProcessMessages()
	host.WaitForClose()

}

// interactiveConfig will ask through the command line to create a Private / Public
// key, what is the listening address
func interactiveConfig() {
	fmt.Println("[+] Welcome ! Let's setup the configuration file for a CoSi server...")

	fmt.Println("[*] We need to know on which [address:]PORT you want your server to listen to.")
	fmt.Print("[*] Type <Enter> for default port " + strconv.Itoa(DefaultPort) + ": ")
	reader := bufio.NewReader(os.Stdin)
	var str = readString(reader)
	// let's dissect the port / IP
	var hostStr string
	var ipProvided = true
	var portStr string
	var serverBinding string
	splitted := strings.Split(str, ":")

	if str == "" {
		portStr = strconv.Itoa(DefaultPort)
		hostStr = "0.0.0.0"
		ipProvided = false
	} else if len(splitted) == 1 {
		// one element provided
		if _, err := strconv.Atoi(splitted[0]); err != nil {
			stderrExit("[-] You have to provide a port number at least!")
		}
		// ip
		ipProvided = false
		hostStr = "0.0.0.0"
		portStr = splitted[0]
	} else if len(splitted) == 2 {
		hostStr = splitted[0]
		portStr = splitted[1]
	}
	// let's check if they are correct
	serverBinding = hostStr + ":" + portStr
	hostStr, portStr, err := net.SplitHostPort(serverBinding)
	if err != nil {
		stderrExit("[-] Invalid connection information for %s: %v", serverBinding, err)
	}
	if net.ParseIP(hostStr) == nil {
		stderrExit("[-] Invalid connection  information for %s", serverBinding)
	}

	fmt.Println("[+] We now need to get a reachable address for other CoSi servers")
	fmt.Println("    and clients to contact you. This address will be put in a group definition")
	fmt.Println("    file that you can share and combine with others to form a Cothority roster.")

	var publicAddress string
	var failedPublic bool
	// if IP was not provided then let's get the public IP address
	if !ipProvided {
		resp, err := http.Get("http://myexternalip.com/raw")
		// cant get the public ip then ask the user for a reachable one
		if err != nil {
			stderr("[-] Could not get your public IP address")
			failedPublic = true
		} else {
			buff, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				stderr("[-] Could not parse your public IP address", err)
				failedPublic = true
			} else {
				publicAddress = strings.TrimSpace(string(buff)) + ":" + portStr
			}
		}
	} else {
		publicAddress = serverBinding
	}

	var reachableAddress string
	// Let's directly ask the user for a reachable address
	if failedPublic {
		reachableAddress = askReachableAddress(reader, portStr)
	} else {
		// try  to connect to ipfound:portgiven
		tryIP := publicAddress
		fmt.Println("[+] Check if the address", tryIP, "is reachable from Internet...")
		if err := tryConnect(tryIP, serverBinding); err != nil {
			stderr("[-] Could not connect to your public IP")
			reachableAddress = askReachableAddress(reader, portStr)
		} else {
			reachableAddress = tryIP
			fmt.Println("[+] Address", reachableAddress, " publicly available from Internet!")
		}
	}

	// create the keys
	fmt.Println("\n[+] Creation of the ed25519 private and public keys...")
	kp := config.NewKeyPair(network.Suite)
	privStr, err := crypto.SecretHex(network.Suite, kp.Secret)
	if err != nil {
		stderrExit("[-] Error formating private key to hexadecimal. Abort.")
	}

	pubStr, err := crypto.PubHex(network.Suite, kp.Public)
	if err != nil {
		stderrExit("[-] Could not parse public key. Abort.")
	}

	fmt.Println("[+] Public key: ", pubStr, "\n")

	conf := &c.CothoritydConfig{
		Public:    pubStr,
		Private:   privStr,
		Addresses: []string{serverBinding},
	}

	var configDone bool
	var configFolder string
	var defaultFolder = path.Dir(getDefaultConfigFile())
	var configFile string
	var groupFile string

	for !configDone {
		// get name of config file and write to config file
		fmt.Println("[*] We need a folder where to write the configuration files: " + DefaultServerConfig + "and " + DefaultGroupFile + ".")
		fmt.Print("[*] Type <Enter> to use the default folder [ " + defaultFolder + " ] :")
		configFolder = readString(reader)
		if configFolder == "" {
			configFolder = defaultFolder
		}
		configFile = path.Join(configFolder, DefaultServerConfig)
		groupFile = path.Join(configFolder, DefaultGroupFile)

		// check if the directory exists
		if _, err := os.Stat(configFolder); os.IsNotExist(err) {
			fmt.Println("[+] Creating inexistant directory configuration", configFolder)
			if err = os.MkdirAll(configFolder, 0744); err != nil {
				stderrExit("[-] Could not create directory configuration %s %v", configFolder, err)
			}
		}

		if checkOverwrite(configFile, reader) && checkOverwrite(groupFile, reader) {
			break
		}
	}

	server := c.NewServerToml(network.Suite, kp.Public, reachableAddress)
	group := c.NewGroupToml(server)

	saveFiles(conf, configFile, group, groupFile)
	fmt.Println("[+] We're done! Have good time using CoSi :)")
}

// Returns true if file exists and user is OK to overwrite, or file dont exists
// Return false if file exists and user is NOT OK to overwrite.
// stderrExit if stg is wrong
func checkOverwrite(file string, reader *bufio.Reader) bool {
	// check if the file exists and ask for override
	if _, err := os.Stat(file); err == nil {
		fmt.Print("[*] Configuration file " + file + " already exists. Override ? (y/n) : ")
		var answer = readString(reader)
		answer = strings.ToLower(answer)
		if answer == "y" {
			return true
		} else if answer == "n" {
			return false
		} else {
			stderrExit("[-] Could not interpret your response. Abort.")
		}
	}
	return false
}

func saveFiles(conf *c.CothoritydConfig, fileConf string, group *c.GroupToml, fileGroup string) {
	if err := conf.Save(fileConf); err != nil {
		stderrExit("[-] Unable to write the config to file:", err)
	}
	fmt.Println("[+] Sucess! You can now use the CoSi server with the config file", fileConf)
	// group definition part
	if err := group.Save(fileGroup); err != nil {
		stderrExit("[-] Could not write your group file snippet: %v", err)
	}

	fmt.Println("[+] Saved a group definition snippet for your server at", fileGroup)
	fmt.Println(group.String() + "\n")

}

func stderr(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf(format, a...)+"\n")
}

func stderrExit(format string, a ...interface{}) {
	stderr(format, a...)
	os.Exit(1)
}

func getDefaultConfigFile() string {
	u, err := user.Current()
	// can't get the user dir, so fallback to current working dir
	if err != nil {
		fmt.Print("[-] Could not get your home's directory. Switching back to current dir.")
		if curr, err := os.Getwd(); err != nil {
			stderrExit("[-] Impossible to get the current directory. %v", err)
		} else {
			return path.Join(curr, DefaultServerConfig)
		}
	}
	// let's try to stick to usual OS folders
	switch runtime.GOOS {
	case "darwin":
		return path.Join(u.HomeDir, "Library", BinaryName, DefaultServerConfig)
	default:
		return path.Join(u.HomeDir, ".config", BinaryName, DefaultServerConfig)
		// TODO WIndows ? FreeBSD ?
	}
}

func readString(reader *bufio.Reader) string {
	str, err := reader.ReadString('\n')
	if err != nil {
		stderrExit("[-] Could not read input.")
	}
	return strings.TrimSpace(str)
}

func askReachableAddress(reader *bufio.Reader, port string) string {
	fmt.Println("[*] Enter the IP address you would like others cothority servers and client to contact you.")
	fmt.Print("[*] Type <Enter> to use the default address [ " + DefaultAddress + " ] if you plan to do local experiments:")
	ipStr := readString(reader)
	if ipStr == "" {
		return DefaultAddress + ":" + port
	}

	splitted := strings.Split(ipStr, ":")
	if len(splitted) == 2 && splitted[1] != port {
		// if the client gave a port number, it must be the same
		stderrExit("[-] The port you gave is not the same as the one your server will be listening. Abort.")
	} else if len(splitted) == 2 && net.ParseIP(splitted[0]) == nil {
		// of if the IP address is wrong
		stderrExit("[-] Invalid IP:port address given (%s)", ipStr)
	} else if len(splitted) == 1 {
		// check if the ip is valid
		if net.ParseIP(ipStr) == nil {
			stderrExit("[-] Invalid IP address given (%s)", ipStr)
		}
		// add the port
		ipStr = ipStr + ":" + port
	}
	return ipStr
}

// Service used to get the port connection service
const whatsMyIP = "http://www.whatsmyip.org/"

// tryConnect will bind to the ip address and ask a internet service to try to
// connect to it. binding is the address where we must listen (needed because
// the reachable address might not be the same as the binding address => NAT, ip
// rules etc).
func tryConnect(ip string, binding string) error {

	stopCh := make(chan bool, 1)
	// let's bind
	go func() {
		ln, err := net.Listen("tcp", binding)
		if err != nil {
			fmt.Println("[-] Trouble with binding to the address:", err)
			return
		}
		con, _ := ln.Accept()
		<-stopCh
		con.Close()
	}()
	defer func() { stopCh <- true }()

	_, port, err := net.SplitHostPort(ip)
	if err != nil {
		return err
	}
	values := url.Values{}
	values.Set("port", port)
	values.Set("timeout", "default")

	// ask the check
	url := whatsMyIP + "port-scanner/scan.php"
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Host", "www.whatsmyip.org")
	req.Header.Set("Referer", "http://www.whatsmyip.org/port-scanner/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:46.0) Gecko/20100101 Firefox/46.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buffer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !bytes.Contains(buffer, []byte("1")) {
		return errors.New("Address unrechable")
	}
	return nil
}
