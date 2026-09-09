#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
#
# Runs serial_upload with -h and checks its usage text. Go's flag package
# exits with status 2 on -h, so the status is not the check; the text is.
set -o nounset -o pipefail

bin="${1:?path to serial_upload}"
out="$("${bin}" -h 2>&1)" || true
if ! grep -q "serial port device name" <<<"${out}"; then
  echo "usage text missing from serial_upload -h output:" >&2
  echo "${out}" >&2
  exit 1
fi
