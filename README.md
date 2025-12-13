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