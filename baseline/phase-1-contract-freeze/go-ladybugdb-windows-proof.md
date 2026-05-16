# Go LadybugDB Windows Proof

Date: 2026-05-08

Status: PASS

Purpose: prove that the Go rewrite can load and read LadybugDB on Windows before broad analyzer work starts.

## Versions

- Go: `go version go1.26.3 windows/amd64`
- Go binding: `github.com/LadybugDB/go-ladybug v0.13.1`
- Native runtime: LadybugDB `v0.16.1`
- Native asset: `liblbug-windows-x86_64.zip`
- Asset SHA256: `3825B9B8ECCA5DE85EA9ECD308F14608545D5E3C551E62F7E112146A50BDCDD4`
- DLL SHA256: `EB976DEBD08D8C9602E89B71DD6798E89A7802997598F11A832A21B91AF6FB20`

## Proof

The proof program in `.tmp/phase1-go-proofs/ladybugdb` used Go 1.26.3, the Go LadybugDB binding,
and the Windows native `lbug_shared` release asset.

Operations verified:

- Opened an in-memory LadybugDB database.
- Opened a connection.
- Created a `Person` node table.
- Inserted one node.
- Read the node back with `MATCH`.

Observed output:

```text
ladybugdb proof ok: created Person and read p1/Alice
```

## Windows Runtime Notes

The Go module expects a native library named `lbug_shared` on Windows. The native release asset
provides `lbug_shared.dll`, `lbug_shared.lib`, `lbug.h`, and `lbug.hpp`.

The proof passed with:

```text
CGO_ENABLED=1
CGO_LDFLAGS=-L.tmp/phase1-go-proofs/ladybugdb/native/liblbug-windows-x86_64 -llbug_shared
PATH=<native-lib-dir>;%PATH%
```

Acceptance: Go can load/read LadybugDB on Windows for the conversion path.
