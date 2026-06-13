---
title: "Installation"
description: "Install bsky from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/bluesky-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `bsky` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/bluesky-cli/cmd/bsky@latest
```

That puts `bsky` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/bluesky-cli
cd bluesky-cli
make build        # produces ./bin/bsky
./bin/bsky version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/bsky:latest --help
```

## Checking the install

```bash
bsky version
```

prints the version and exits.
