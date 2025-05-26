"""Bazel rules for golangci-lint."""

load("@bazel_skylib//lib:selects.bzl", "selects")

def golangci_lint_config():
    """Configure golangci-lint for different platforms."""
    selects.config_setting_group(
        name = "linux_amd64",
        match_all = [
            "@platforms//os:linux",
            "@platforms//cpu:x86_64",
        ],
    )

    selects.config_setting_group(
        name = "linux_arm64",
        match_all = [
            "@platforms//os:linux",
            "@platforms//cpu:aarch64",
        ],
    )

    selects.config_setting_group(
        name = "darwin_amd64",
        match_all = [
            "@platforms//os:osx",
            "@platforms//cpu:x86_64",
        ],
    )

    selects.config_setting_group(
        name = "darwin_arm64",
        match_all = [
            "@platforms//os:osx",
            "@platforms//cpu:aarch64",
        ],
    )

    selects.config_setting_group(
        name = "windows_amd64",
        match_all = [
            "@platforms//os:windows",
            "@platforms//cpu:x86_64",
        ],
    )

    native.alias(
        name = "golangci-lint",
        actual = select({
            ":linux_amd64": "@golangci_lint_linux_amd64//:golangci-lint-linux-amd64",
            ":linux_arm64": "@golangci_lint_linux_arm64//:golangci-lint-linux-arm64",
            ":darwin_amd64": "@golangci_lint_darwin_amd64//:golangci-lint-darwin-amd64",
            ":darwin_arm64": "@golangci_lint_darwin_arm64//:golangci-lint-darwin-arm64",
            ":windows_amd64": "@golangci_lint_windows_amd64//:golangci-lint-windows-amd64",
            "//conditions:default": "@golangci_lint_linux_amd64//:golangci-lint-linux-amd64",
        }),
        visibility = ["//visibility:public"],
    )
