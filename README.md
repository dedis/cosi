[![Build Status](https://travis-ci.org/dedis/cothority.svg?branch=master)](https://travis-ci.org/dedis/cothority)
[![Coverage Status](https://coveralls.io/repos/github/dedis/cothority/badge.svg?branch=master)](https://coveralls.io/github/dedis/costhority?branch=master)


# CoSi

**Update: Development of CoSi moved to the 
[cothority](https://github.com/dedis/cothority/) repository. 
For an updated version of this document see: 
https://github.com/dedis/cothority/blob/master/app/cosi
This repository only contains the first unstable 
[`v0` branch](https://gopkg.in/dedis/cosi.v0) and some outdated 
[releases](https://github.com/dedis/cosi/releases).** 

**For future development and releases please have a look at the
[cothority repository](https://github.com/dedis/cothority/).**

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
and next-generation crypto-currencies.
For further background and technical details see this research paper:
* [Keeping Authorities "Honest or Bust" with Decentralized Witness Cosigning](http://dedis.cs.yale.edu/dissent/papers/witness-abs), 
[IEEE Security & Privacy 2016](http://www.ieee-security.org/TC/SP2016/).

For questions and discussions please join the
[mailing list](https://groups.google.com/forum/#!forum/cothority).

Other related papers:
* [Certificate Cothority - Towards Trustworthy Collective CAs](https://petsymposium.org/2015/papers/syta-cc-hotpets2015.pdf), 
[HotPETS 2015](https://petsymposium.org/2015/hotpets.php)
* [Enhancing Bitcoin Security and Performance with Strong Consistency via Collective Signing](http://arxiv.org/abs/1602.06997), 
[USENIX Security 2016](https://www.usenix.org/conference/usenixsecurity16) (to appear)
 

# Versions

For the moment we have two version: _v0_ (this repo) and _master_. 
CoSi development continues in the 
[cothority repository](https://github.com/dedis/cothority/).

## V0

This is a stable version that depends on the v0-versions of the other dedis-packages. 
It will only receive bugfixes, but no changes that will make the code incompatible. 
You can find this version at:

https://github.com/dedis/cosi/tree/v0

Find more information on installing and using the _v0_ release 
in the _v0_ [documentation](https://github.com/dedis/cosi/blob/v0/README.md).

# Standalone Language-specific Verification/Signing Modules

The CoSi client and server software implemented in this repository is intended
to provide a scalable, robust distributed protocol for generating collective
signatures - but you do not always need a full distributed protocol to work
with CoSi signatures.  In particular, applications that wish to accept and
rely on CoSi signatures as part of some other protocol - e.g., a software
update daemon or certificate checking library - will typically need only a
small signature verification module, preferably written in (or with bindings
for) the language the relying application is written in.

Following is a list of pointers to standalone language-specific CoSi signature
verification modules available for use in applications this way, typically
implemented as an extension of an existing ed25519 implementation for that
language.  Pointers to more such standalone modules will be added for other
languages as we or others create them.  Some of these standalone modules also
include (limited) CoSi signature creation support.  We hope that eventually
some or all of these CoSi signature handling extensions will be merged back
into the base crypto libraries they were derived from.  Note that the
repositories below are experimental, likely to change, and may disappear
if/when they get successfully upstreamed.

* C language, signature verification only: in [temporary fork of libsodium](https://github.com/bford/libsodium).
See the new `crypto_sign_ed25519_verify_cosi` function in the
[crypto_sign/ed25519/ref10](https://github.com/bford/libsodium/blob/master/src/libsodium/crypto_sign/ed25519/ref10/open.c)
module, and the test suites for CoSi signature verification in
[libsodium/test/default/sign.c](https://github.com/bford/libsodium/blob/master/test/default/sign.c).
Run `make check` as usual for libsodium to run all tests including these.
* Go language, verification and signing code: in
[temporary fork of golang.org/x/crypto](https://github.com/bford/golang-x-crypto).
See the new [ed25519/cosi] package, with
[extensive godoc API documentation here](https://godoc.org/github.com/bford/golang-x-crypto/ed25519/cosi).
Run `go test` to run the standard test suite, and `go test -bench=.` to run a
suite of performance benchmarks for this package.
