### Meteora ☄️ DammV2GOSDK

The work done so far aims to maintain correctness & closeness w/ the JS/TS SDK. In that fact, directories, files, and variable names shares closeness with the JS/TS SDK... making comparison very easy. Task forwards from here is to introduce the project to the Meteora dev team, maintain & optimises where possible.

## Running test...

Since there’s no Bankrun or LiteSVM for Go, the next best option (if not better) is — [Surfpool, specifically Surfnet](https://docs.surfpool.run/rpc/surfnet). It runs a local Solana validator but with real on-chain data. The program binary can be [found here](https://github.com/txtx/surfpool/releases).

At the time of writing this README.md, I’m still using `v0.9.1`. Versions up to `v0.9.5` were crashing (I’ve been in touch with the devs on Discord about this 🤣). The current version is `v0.10.2`, which I’ll be testing soon and will update this README if all goes well.

- to run all test:

> go test .

- to run a specific test func :

> go test -run "TestSplitPosition"

- run a specific sub-test:

> go test -run "TestSplitPosition/xxxxx"

(Also, at the point of writing, three (of the 14) test does not pass yet and have been skipped... would be fixed soon.)

The idl was generated w/ [solana-anchor-go](https://github.com/fragmetric-labs/solana-anchor-go) from the guys are Fragmetric. The dependency is also inlcuded in the go.mod file w/ [`go tool`](https://www.bytesizego.com/blog/go-124-tool-directive).
