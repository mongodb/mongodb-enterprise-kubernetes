#!/usr/bin/env bash

# Copyright 2022 MongoDB Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -Eeou pipefail

# Notarize generated binaries with Apple and replace the original binary with the notarized one
# This depends on binaries being generated in a goreleaser manner and gon being set up.
# goreleaser should already take care of calling this script as a hook.

if [[ -f "./dist/kubectl-mongodb_darwin_amd64_v1/kubectl-mongodb" && -f "./dist/kubectl-mongodb_darwin_arm64/kubectl-mongodb" && ! -f "./dist/kubectl-mongodb_macos_signed.zip" ]]; then
	echo "notarizing macOs binaries"
	zip -r ./dist/kubectl-mongodb_amd64_arm64_bin.zip ./dist/kubectl-mongodb_darwin_amd64_v1/kubectl-mongodb ./dist/kubectl-mongodb_darwin_arm64/kubectl-mongodb # The Notarization Service takes an archive as input
	"${workdir:-.}"/linux_amd64/macnotary \
		-f ./dist/kubectl-mongodb_amd64_arm64_bin.zip \
		-m notarizeAndSign -u https://dev.macos-notary.build.10gen.cc/api \
		-b com.mongodb.mongodb-kubectl-mongodb \
		-o ./dist/kubectl-mongodb_macos_signed.zip

	echo "replacing original files"
	unzip -oj ./dist/kubectl-mongodb_macos_signed.zip dist/kubectl-mongodb_darwin_amd64_v1/kubectl-mongodb -d ./dist/kubectl-mongodb_darwin_amd64_v1/
	unzip -oj ./dist/kubectl-mongodb_macos_signed.zip dist/kubectl-mongodb_darwin_arm64/kubectl-mongodb -d ./dist/kubectl-mongodb_darwin_arm64/
fi
