// Cosi takes a file or a message and signs it collectively.
// For usage, see README.md
package main

import (
	"time"

	"gopkg.in/codegangsta/cli.v1"

	"os"

	"github.com/dedis/cothority/lib/dbg"
)

// Name of the binary
const BinaryName = "cosi"

// Version of the binary
const Version = "0.1.4-alpha"

// DefaultGroupFile is the name of the default file to lookup for group
// definition
const DefaultGroupFile = "group.toml"

// DefaultServerConfig is the name of the default file to lookup for server
// configuration file
const DefaultServerConfig = "config.toml"

// DefaultPort where to listen; At time of writing, this port is not listed in
// /etc/services
const DefaultPort = 6879

// DefaultAddress where to be contacted by other servers
const DefaultAddress = "127.0.0.1"

const optionGroup = "group"
const optionGroupShort = "g"

const optionConfig = "config"
const optionConfigShort = "c"

// RequestTimeOut defines when the client stops waiting for the CoSi group to
// reply
const RequestTimeOut = time.Second * 10

func init() {
	dbg.SetDebugVisible(1)
	dbg.SetUseColors(false)
}

func main() {
	app := cli.NewApp()
	app.Name = "CoSi app"
	app.Usage = "Collectively sign a file or verify its signature."
	app.Version = Version
	binaryFlags := []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
	}

	clientFlags := []cli.Flag{
		cli.StringFlag{
			Name:  optionGroup + ", " + optionGroupShort,
			Value: DefaultGroupFile,
			Usage: "CoSi group definition file",
		},
	}

	serverFlags := []cli.Flag{
		cli.StringFlag{
			Name:  optionConfig + ", " + optionConfigShort,
			Value: getDefaultConfigFile(),
			Usage: "Configuration file of the server",
		},
	}
	app.Commands = []cli.Command{
		// BEGIN CLIENT ----------
		{
			Name:    "sign",
			Aliases: []string{"s"},
			Usage:   "Collectively sign a `FILE`. The signature is written to STDOUT by default.",
			Action:  signFile,
			Flags: append(clientFlags, []cli.Flag{
				cli.StringFlag{
					Name:  "out, o",
					Usage: "Write signature to `outfile` instead of standard output",
				},
			}...),
		},
		{
			Name:    "verify",
			Aliases: []string{"v"},
			Usage:   "Verify collective signature of a `FILE`. Signature is read by default from STDIN.",
			Action:  verifyFile,
			Flags: append(clientFlags, []cli.Flag{
				cli.StringFlag{
					Name:  "signature, s",
					Usage: "Read signature from `FILE` instead of STDIN",
				},
			}...),
		},
		{
			Name:    "check",
			Aliases: []string{"c"},
			Usage:   "Check if the servers in the group definition are up and running",
			Action:  checkConfig,
			Flags:   clientFlags,
		},

		// CLIENT END ----------
		// BEGIN SERVER --------
		{
			Name:  "server",
			Usage: "act as Cothority server",
			Action: func(c *cli.Context) error {
				runServer(c)
				return nil
			},
			Flags: serverFlags,
			Subcommands: []cli.Command{
				{
					Name:    "setup",
					Aliases: []string{"s"},
					Usage:   "Setup the configuration for the server (interactive)",
					Action: func(c *cli.Context) error {
						if c.String(optionConfig) != "" {
							stderrExit("[-] Configuration file option can't be used for the 'setup' command")
						}
						if c.GlobalIsSet("debug") {
							stderrExit("[-] Debug option can't be used for the 'setup' command")
						}
						interactiveConfig()
						return nil
					},
				},
			},
		},
		// SERVER END ----------
	}

	app.Flags = binaryFlags
	app.Before = func(c *cli.Context) error {
		dbg.SetDebugVisible(c.GlobalInt("debug"))
		return nil
	}
	app.Run(os.Args)
}
