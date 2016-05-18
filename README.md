# CoSi

This repository implements a of the collective signing (CoSi) protocol
which itself builds on Schnorr multi-signatures over elliptic curves. 
CoSi enables authorities to have their statements collectively signed 
(co-signed) by a diverse, decentralized, and scalable group of 
(potentially thousands of) witnesses and, for example, could be employed 
to proactively harden critical Internet authorities. 
Among other things, one could imagine applications to Certificate 
Transparency, DNSSEC, software distribution, the Tor anonymity 
network, and cryptocurrencies.
Each run of the protocol yields a single digital signature with size and 
verification cost comparable to an individual signature, but compactly
attests that both the leader and perhaps many witnesses observed and 
agreed to sign the statement.

## Further Information

Primary information sources:
* Keeping Authorities "Honest or Bust" with Decentralized Witness 
Cosigning: [paper](http://dedis.cs.yale.edu/dissent/papers/witness-abs), 
[slides](http://dedis.cs.yale.edu/dissent/pres/151009-stanford-cothorities.pdf)
* For questions and discussions please join the
[mailing list](https://groups.google.com/forum/#!forum/cothority).

Other cothority-related research papers:
* Certificate Cothority - Towards Trustworthy Collective CAs: 
[paper](https://petsymposium.org/2015/papers/syta-cc-hotpets2015.pdf)
* Enhancing Bitcoin Security and Performance with Strong Consistency via Collective Signing: [paper](http://arxiv.org/abs/1602.06997)
 

## Warning

**The software provided in this repository is highly experimental and under
heavy development. Do not use it yet for anything security-critical.  or if you
use it, do so in a way that supplements (rather than replacing) existing, stable
signing mechanisms.

All usage is at your own risk!**

## Requirements

In order to build (and run) the simulations you need to install a recent 
[Golang](https://golang.org/dl/) version (1.5.2+).
See Golang's documentation on how-to 
[install and configure](https://golang.org/doc/install) Go,
including setting the GOPATH environment variable. 

## Installation

For convenience we provide x86-64 binaries for Linux and Mac OS X,
which are self-contained and don't require Go to be installed.
But of course you can also compile the tools from source.
 
### Installing binaries from .tar.gz

Download the latest package from 

	https://github.com/dedis/cosi/releases/latest

and untar into a directory that is in your `$PATH`:

```bash
tar xf cosi-*tar.gz -C ~/bin
```

### Installing from source

To install the command-line tools from source, make sure that
[Go is installed](https://golang.org/doc/install)
and that
[`$GOPATH` and `$GOBIN` are set](https://golang.org/doc/code.html#GOPATH).

```bash
go get -u github.com/dedis/cosi
```

The `cosi` binary will be installed in the directory indicated by `$GOBIN`.

# Command-line Interface

The `cosi` binary provides both the client application and the server
application.

## Client side

In order to initiate a Collective Signing round, you need to get a list of CoSi
servers with their public keys and address. We provide you with already with a
list of our servers running the CoSi server [dedis_group.toml]. However, CoSi
will by default search for a file "group.toml" in the default configuration folders
which are `$HOME/.config/cosi/` for Linux systems and `$HOME/Library/cosi/` for
mac systems. If CoSi did not find anything, the default is to search in the current
directory.

Once you have a valid group definition, you can sign a file using:

```bash
cosi sign -g dedis_group.toml my_file 
```

By default, the signature is written to STDOUT. In order to get the signature
written to a file, you can either redirect or use the the `-o output` flag:

```base
cosi sign -g dedis_group.toml -o my_file.sig my_file
```

To verify a signature, just type:
  
```bash
cosi verify -g dedis_group.toml -s my_file.sig my_file
```

You can pass the signature directly to STDOUT and omit the `-s sig` flag:

```bash
# will read the signature from STDIN
cat my_file.sig | cosi verify -g dedis_group.toml my_file
```

In the current implementation, the witnesses do not validate or check the 
messages you propose in any way; they merely serve to provide transparency
by publicly attesting the fact that they have observed and cosigned the message.

## Running your own CoSi server

First you need to create a configuration file for the server including a 
public/private key pair for the server. 
You can create a default server configuration with a fresh 
public/private key pair as follows:

```bash
cosi server setup
```

Follow the instructions on the screen. At the end, you should have two files:
* One local server configuration file which is used by your cothority server,
* One group definition file that you will share with other cothority members and
  clients that wants to contact you.

To run the server, simply type:
```bash
cosi server
```

The server will try to read the default configuration file; if you have put the
file in a custom location, provide the path using:
```base
cosi server -config path/file.toml
``` 

### Creating a Collective Signing Group
By running several `cothorityd` instances (and copying the appropriate lines 
of their output) you can create a `servers.toml` that looks like 
this:

```
Description = "My Test group"

[[servers]]
  Addresses = ["127.0.0.1:2000"]
  Public = "6T7FwlCuVixvu7XMI9gRPmyCuqxKk/WUaGbwVvhA+kc="
  Description = "Local Server 1"

[[servers]]
  Addresses = ["127.0.0.1:2001"]
  Public = "Aq0mVAeAvBZBxQPC9EbI8w6he2FHlz83D+Pz+zZTmJI="
  Description = "Local Server 2"
```

Your list will look different, as the public keys will not be the same. But
it is important that you run the servers on different ports. Here the ports
are 2000 and 2001.
 
### Checking server-list

The `cosi`-binary has a command to verify the availability for all
servers in a `servers.toml`-file:

```bash
cosi check
```

This will first contact each server individually, then make a small cothority-
group of all possible pairs of servers. If there is a problem with regard to
some firewalls or bad connections, you will see a "Timeout on signing" error
message and you can fix the problem.
