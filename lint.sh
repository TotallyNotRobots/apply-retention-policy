#!/bin/bash
# The MIT License (MIT)
#
# Copyright © 2025 linuxdaemon <linuxdaemon.irc@gmail.com>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

set -euo pipefail

# Find the runfiles directory
if [[ -n "${RUNFILES_DIR:-}" ]]; then
    # Running under Bazel
    GOLANGCI_LINT="${RUNFILES_DIR}/+golangci_lint+golangci_lint/bin/golangci-lint"
    # Set GOLANGCI_LINT_CACHE to a writable directory in the test environment
    export GOLANGCI_LINT_CACHE="${TEST_TMPDIR:-/tmp}/golangci-lint-cache"
    mkdir -p "$GOLANGCI_LINT_CACHE"

    # Change to the Bazel workspace root if available
    if [[ -n "${BUILD_WORKSPACE_DIRECTORY:-}" ]]; then
        cd "$BUILD_WORKSPACE_DIRECTORY"
    else
        # Fallback: Traverse up from runfiles to find the workspace root by searching for go.mod
        SEARCH_DIR="${RUNFILES_DIR}/_main"
        while [[ "$SEARCH_DIR" != "/" && ! -f "$SEARCH_DIR/go.mod" ]]; do
            SEARCH_DIR="$(dirname "$SEARCH_DIR")"
        done
        if [[ -f "$SEARCH_DIR/go.mod" ]]; then
            cd "$SEARCH_DIR"
        else
            echo "Error: Could not find workspace root containing go.mod"
            exit 1
        fi
    fi

    # Ensure go.mod and go.sum are in the current directory
    if [[ ! -f go.mod || ! -f go.sum ]]; then
        echo "Error: go.mod and/or go.sum not found in $(pwd)"
        exit 1
    fi

    # Run golangci-lint
    "$GOLANGCI_LINT" run --config .golangci.yml ./...
else
    # Running directly
    GOLANGCI_LINT="$(dirname "$0")/external/golangci_lint/bin/golangci-lint"
    "$GOLANGCI_LINT" run --config .golangci.yml ./...
fi
