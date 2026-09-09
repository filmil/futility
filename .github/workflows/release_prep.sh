#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Builds the release source archive and prints the release notes to stdout.
#
# bazel-contrib's release_ruleset.yaml runs this as
# `release_prep.sh TAG > release_notes.txt`, requires this exact path, and
# uploads and attests the archive it produces. The archive keeps the name and
# the layout the previous release action gave it: .bcr/source.template.json
# names it, and the Bazel Central Registry compares the checked-in
# MODULE.bazel against the one in the archive.
set -o errexit -o nounset -o pipefail

TAG="$1"
VERSION="${TAG#v}"
ARCHIVE="futility-${TAG}.zip"

# The exclusions thedoctor0/zip-release applied before this script existed,
# plus the two files this workflow itself leaves in the tree: release_notes.txt,
# which the shell creates before this script runs, and the archive. `bazel-*`
# matters: zip follows symlinks and would store the whole output tree, and the
# reusable workflow runs `bazel test //...` before this script.
zip --quiet --recurse-paths "${ARCHIVE}" . \
  -x '*.git*' '/*node_modules/*' '.editorconfig' 'bazel-*' 'integration/bazel-*' \
     'release_notes.txt' "${ARCHIVE}"

cat <<NOTES
## Using Bzlmod

\`\`\`starlark
bazel_dep(name = "futility", version = "${VERSION}")
\`\`\`

With filmil/bazel-registry ahead of the Bazel Central Registry in \`.bazelrc\`:

\`\`\`
common --registry=https://raw.githubusercontent.com/filmil/bazel-registry/main
common --registry=https://bcr.bazel.build
\`\`\`

The \`serial_upload\` binaries for linux 386, amd64 and arm64 are attached to
this release as well.
NOTES
