"""Bazel rules for golangci-lint."""

load("@bazel_skylib//lib:selects.bzl", "selects")

def golangci_lint_config(name = "golangci-lint"):
    """Configure golangci-lint for different platforms.

    Args:
      name: The name of the golangci-lint target to create. Defaults to "golangci-lint".
    """
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

    selects.config_setting_group(
        name = "windows_arm64",
        match_all = [
            "@platforms//os:windows",
            "@platforms//cpu:aarch64",
        ],
    )

    native.alias(
        name = name,
        actual = select({
            ":linux_amd64": "@golangci_lint_linux_amd64//:golangci-lint",
            ":linux_arm64": "@golangci_lint_linux_arm64//:golangci-lint",
            ":darwin_amd64": "@golangci_lint_darwin_amd64//:golangci-lint",
            ":darwin_arm64": "@golangci_lint_darwin_arm64//:golangci-lint",
            ":windows_amd64": "@golangci_lint_windows_amd64//:golangci-lint.exe",
            ":windows_arm64": "@golangci_lint_windows_arm64//:golangci-lint.exe",
            "//conditions:default": "@golangci_lint_linux_amd64//:golangci-lint",
        }),
        visibility = ["//visibility:public"],
    )
