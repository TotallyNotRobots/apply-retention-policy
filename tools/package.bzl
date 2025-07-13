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

"""Tools for packaging the application."""

load("@rules_go//go:def.bzl", "go_cross_binary")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")
load("@rules_pkg//pkg:zip.bzl", "pkg_zip")

def go_release_package(name, cpu, os, target):
    """Package a binary for distribution.

    Args:
        name: The name of the package.
        cpu: The CPU architecture.
        os: The operating system.
        target: The target platform.
    """

    # Build the binary
    go_cross_binary(
        name = name + "_binary",
        target = target,
        platform = "//:{}_{}_platform".format(os, cpu),
    )

    # Create a genrule to create a directory with the renamed binary
    binary_name = "apply-retention-policy" + (".exe" if os == "windows" else "")
    native.genrule(
        name = name + "_renamed_binary",
        srcs = [name + "_binary"],
        outs = [name + "/" + binary_name],
        cmd = "mkdir -p " + name + " && cp $< $@",
        output_to_bindir = True,
    )

    if os == "windows":
        pkg_zip(
            name = name,
            srcs = [
                name + "_renamed_binary",
                "//:LICENSE",
                "//:README.md",
            ],
            package_dir = name,
        )
    else:
        pkg_tar(
            name = name,
            srcs = [
                name + "_renamed_binary",
                "//:LICENSE",
                "//:README.md",
            ],
            extension = "tar.gz",
            package_dir = name,
        )
