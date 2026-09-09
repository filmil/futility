# Project Overview

[![Test](https://github.com/filmil/futility/actions/workflows/test.yml/badge.svg)](https://github.com/filmil/futility/actions/workflows/test.yml)
[![Tag and Release](https://github.com/filmil/futility/actions/workflows/tag-and-release.yml/badge.svg)](https://github.com/filmil/futility/actions/workflows/tag-and-release.yml)
[![Publish to my Bazel registry](https://github.com/filmil/futility/actions/workflows/publish.yml/badge.svg)](https://github.com/filmil/futility/actions/workflows/publish.yml)
[![Publish on Bazel Central Registry](https://github.com/filmil/futility/actions/workflows/publish-bcr.yml/badge.svg)](https://github.com/filmil/futility/actions/workflows/publish-bcr.yml)

This repository contains various utilities and tools. Below is a summary of the main components:

## `cmd/serial_upload`

The `serial_upload` utility is a Go program that facilitates uploading text files over a serial connection. It is designed to wait for a specific prompt on the serial port before sending the file's contents, making it suitable for interacting with devices that require a handshake or specific command sequence.

For more in-depth information, including detailed specifications and usage instructions, please refer to the [serial_upload README](cmd/serial_upload/README.md).

This module was partially written using an automated coding assistant, with
human supervision.

## Releasing and publishing

`Tag and Release` in `.github/workflows/tag-and-release.yml` runs monthly and
on `workflow_dispatch`.
It releases only when a commit landed since the last tag, computes the
version from the conventional-commit titles since that tag, and pushes it.
The release itself goes through bazel-contrib's `release_ruleset.yaml`:
it runs `bazel test //...` at the tag, has
`.github/workflows/release_prep.sh` build `futility-<tag>.zip` and the
release notes, and attests the archive's provenance.
A release then lists the archive, `futility-<tag>.zip.intoto.jsonl`, and the
`serial_upload` binaries for linux 386, amd64 and arm64, which a later job
builds at the tag and adds.

The same run publishes the release to [my Bazel registry][reg] as a pull
request, through `.github/workflows/publish.yml`, and then opens a pull
request against the Bazel Central Registry through
`.github/workflows/publish-bcr.yml`, with attested `MODULE.bazel` and
`source.json`, which is what the BCR presubmit verifies with `slsa-verifier`.
Only that publish attests: two attesting publishes would overwrite each
other's attestation files on the release.
The presubmit runs the tests in `integration/`, a module that depends on
futility the way a registry user does; CI runs them on every change.

To check a release the way the BCR does, with the archive downloaded from
the release:

```
slsa-verifier verify-github-attestation \
  --attestation-path futility-<tag>.zip.intoto.jsonl \
  --source-uri github.com/filmil/futility \
  --builder-id https://github.com/bazel-contrib/.github/.github/workflows/release_ruleset.yaml \
  futility-<tag>.zip
```

[reg]: https://github.com/filmil/bazel-registry
