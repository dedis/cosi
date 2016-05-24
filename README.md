# CoSi

This repository implements the CoSi protocol for scalable collective signing.
CoSi enables authorities to have their statements collectively signed,
or *co-signed*, by a scalable group of independent parties or *witnesses*.
The signatures that CoSi produces convey the same information
as a list of conventional signatures from all participants,
but CoSi signatures are far more compact and efficient for clients to verify.
In practice, a CoSi collective signature is close to the same size as &mdash;
and comparable in verification costs to &mdash;
a *single* individual signature.

CoSi is intended to facilitate increased transparency and security-hardening
for critical Internet authorities such as certificate authorities,
[time services](http://www.nist.gov/pml/div688/grp40/its.cfm),
naming authorities such as [DNSSEC](http://www.dnssec.net),
software distribution and update services,
directory services used by tools such as [Tor](https://www.torproject.org),
and next-generation cryptocurrencies.
For further background and technical details see this research paper:
* [Keeping Authorities "Honest or Bust" with Decentralized Witness Cosigning](http://dedis.cs.yale.edu/dissent/papers/witness-abs), [IEEE Security & Privacy 2016](http://www.ieee-security.org/TC/SP2016/).

For questions and discussions please join the
[mailing list](https://groups.google.com/forum/#!forum/cothority).

Other related papers:
* [Certificate Cothority - Towards Trustworthy Collective CAs](https://petsymposium.org/2015/papers/syta-cc-hotpets2015.pdf), [HotPETS 2015](https://petsymposium.org/2015/hotpets.php)
* [Enhancing Bitcoin Security and Performance with Strong Consistency via Collective Signing](http://arxiv.org/abs/1602.06997), [USENIX Security 2016](https://www.usenix.org/conference/usenixsecurity16) (to appear)
 

**Warning: This software is experimental and still under development.
Do not use it yet for security-critical purposes.  If you use it,
do so to supplement, rather than replace, existing signing mechanisms.
Use at your own risk!**

# Installation

You may install CoSi from either pre-built binaries
or from [Go](https://golang.org/) source code,
as described below.

## Installing from binaries

For convenience we provide self-contained x86-64 binaries
for Linux and Mac OS X.
[Download the latest release](https://github.com/dedis/cosi/releases),
untar it, and move the appropriate binary for your platform
into a directory that is in your `$PATH`.
The 'cosi'-script choses the correct binary for you.
If your `~/bin` is in the `$PATH`, you can do:

```bash
tar xf cosi-*tar.gz -C ~/bin
```

Now you can go on directly to *Command-line Interface*

## Installing from source

To build and run CoSi from source code you will need to install
[Go](https://golang.org/) version 1.5.2 or later.
See
[the Go documentation](https://golang.org/doc/install)
on how to install and configure Go,
and make sure that
[`$GOPATH` and `$GOBIN` are set](https://golang.org/doc/code.html#GOPATH).
Then you can fetch, update, compile and install the cosi-binary using:

```bash
go get -u github.com/dedis/cosi
```

The `cosi` binary will be installed in the directory indicated by `$GOBIN`.

# Command-line Interface

The `cosi` application provides both a client for signing messages,
and a server implementing the cosigner or witness-server role
in the CoSi protocol.

## Collectively signing messages with the CoSi client

In order to sign messages collectively, you first need to define the set of
cosigners that will participate.  To do this, you need to prepare a *group definition* 
file which lists the cosigners to use with their public keys and Internet addresses.
You may use [our default list of public CoSi
servers](https://github.com/dedis/cosi/blob/master/dedis_group.toml) if you wish, or define your own.

CoSi will by default search for a file "group.toml" in the default configuration folders
which are `$HOME/.config/cosi/` for Linux systems and `$HOME/Library/cosi/` for
mac systems. If CoSi did not find anything, the default mechanism is to search in the current
directory.

Once you have a valid group definition, you can sign a file using:

```bash
cosi sign -g dedis_group.toml my_file 
```

When collective signing completes,
the resulting signature will be written to standard output by default.
To write the signature written to a file,
you may redirect output or use the the `-o` option:

```base
cosi sign -g dedis_group.toml -o my_file.sig my_file
```

To verify a collective signature, use the `cosi verify` command:
  
```bash
cosi verify -g dedis_group.toml -s my_file.sig my_file
```

Verification can also take the signature from standard input:

```bash
cat my_file.sig | cosi verify -g dedis_group.toml my_file
```

In the current prototype, CoSi witness servers do not validate or check the 
messages you propose in any way; they merely serve to provide transparency
by publicly attesting the fact that they have observed and cosigned the message.
A future CoSi release will add support for message validation plugins,
by which the servers can apply application-specific checks to messages
before signing off on them,
e.g., to validate a [collectively signed blockchain](http://arxiv.org/abs/1602.06997).

## Running your own CoSi witness server

First you need to create a configuration file for the server including a 
public/private key pair.
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
```bash
cosi server -config path/file.toml
```

You can also ask the server to print out some debugging messages by indicating
a level. Using level 1 shows when a message gets signed:

```bash
cosi -d 1 server
```

### Creating a collective signing group

If you run several CoSi servers,
you can concatenate their individual `group.toml` outputs
to define your own cosigning group.
You may optionally use any or all of our experimental
[default CoSi servers](https://github.com/dedis/cosi/blob/master/dedis_group.toml)
if you wish.
Your resulting `group.toml' file should look something like this:

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

Your specific list will be different, of course,
especially in the specific IP addresses and public keys.
If you run multiple servers on the same machine for experimentation,
they must of course be assigned different ports,
e.g., 2000 and 2001 in the example above.
 
### Checking the status of a cosigning group

You may use the `cosi check` command to
verify the availability and operation
of the servers listed in a group definition file:

```bash
cosi check -g group.toml
```

This will first contact each server individually, then make a small cothority-
group of all possible pairs of servers.
If there are connectivity problems,
due to firewalls or bad connections for example,
you will see a "Timeout on signing" or similar error message.
