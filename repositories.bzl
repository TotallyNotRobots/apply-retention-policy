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

def _golangci_lint_impl(ctx):
    """Implementation of the golangci-lint repository rule."""
    os = ctx.os.name
    arch = ctx.os.arch
    if os == "linux" and (arch == "x86_64" or arch == "amd64"):
        url = "https://github.com/golangci/golangci-lint/releases/download/v2.1.6/golangci-lint-2.1.6-linux-amd64.tar.gz"
        sha256 = "e55e0eb515936c0fbd178bce504798a9bd2f0b191e5e357768b18fd5415ee541"  # Verified
        inner_path = "golangci-lint-2.1.6-linux-amd64/golangci-lint"
    elif os == "linux" and arch == "aarch64":
        url = "https://github.com/golangci/golangci-lint/releases/download/v2.1.6/golangci-lint-2.1.6-linux-arm64.tar.gz"
        sha256 = "582eb73880f4408d7fb89f12b502d577bd7b0b63d8c681da92bb6b9d934d4363"  # Verified
        inner_path = "golangci-lint-2.1.6-linux-arm64/golangci-lint"
    elif os == "darwin" and arch == "x86_64":
        url = "https://github.com/golangci/golangci-lint/releases/download/v2.1.6/golangci-lint-2.1.6-darwin-amd64.tar.gz"
        sha256 = "e091107c4ca7e283902343ba3a09d14fb56b86e071effd461ce9d67193ef580e"  # Verified
        inner_path = "golangci-lint-2.1.6-darwin-amd64/golangci-lint"
    elif os == "darwin" and arch == "arm64":
        url = "https://github.com/golangci/golangci-lint/releases/download/v2.1.6/golangci-lint-2.1.6-darwin-arm64.tar.gz"
        sha256 = "90783fa092a0f64a4f7b7d419f3da1f53207e300261773babe962957240e9ea6"  # Verified
        inner_path = "golangci-lint-2.1.6-darwin-arm64/golangci-lint"
    else:
        fail("Unsupported platform: {} {}".format(os, arch))

    # Download and extract to a temporary directory
    ctx.download_and_extract(
        url = url,
        sha256 = sha256,
        output = "temp",
    )

    # Create the directory structure
    ctx.execute(["mkdir", "-p", "bin"])

    # Move the binary to the bin directory
    ctx.execute(["mv", "temp/" + inner_path, "bin/golangci-lint"])
    ctx.execute(["chmod", "+x", "bin/golangci-lint"])

    # Create BUILD file at the root
    ctx.file(
        "BUILD.bazel",
        content = """
exports_files(["bin/golangci-lint"], visibility = ["//visibility:public"])
""",
    )

golangci_lint_repository = repository_rule(
    implementation = _golangci_lint_impl,
    attrs = {},
)

def _golangci_lint_extension_impl(ctx):
    golangci_lint_repository(name = "golangci_lint")

golangci_lint = module_extension(
    implementation = _golangci_lint_extension_impl,
    tag_classes = {},
)
