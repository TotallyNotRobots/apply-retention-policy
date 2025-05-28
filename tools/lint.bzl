"""Bazel rules for golangci-lint."""

load("@buildifier_prebuilt//:rules.bzl", "buildifier")

def golangci_lint_config(name = "golangci-lint"):
    """Configure golangci-lint for different platforms.

    Args:
      name: The name of the golangci-lint target to create. Defaults to "golangci-lint".
    """

    native.alias(
        name = name + "_osx",
        actual = select({
            "@platforms//cpu:x86_64": "@golangci_lint_darwin_amd64//:golangci-lint",
            "@platforms//cpu:arm64": "@golangci_lint_darwin_arm64//:golangci-lint",
            "//conditions:default": "@golangci_lint_darwin_amd64//:golangci-lint",
        }),
    )

    native.alias(
        name = name + "_linux",
        actual = select({
            "@platforms//cpu:x86_64": "@golangci_lint_linux_amd64//:golangci-lint",
            "@platforms//cpu:arm64": "@golangci_lint_linux_arm64//:golangci-lint",
            "//conditions:default": "@golangci_lint_linux_amd64//:golangci-lint",
        }),
    )

    native.alias(
        name = name + "_windows",
        actual = select({
            "@platforms//cpu:x86_64": "@golangci_lint_windows_amd64//:golangci-lint.exe",
            "@platforms//cpu:arm64": "@golangci_lint_windows_arm64//:golangci-lint.exe",
            "//conditions:default": "@golangci_lint_windows_amd64//:golangci-lint.exe",
        }),
    )

    native.alias(
        name = name,
        actual = select({
            "@platforms//os:osx": name + "_osx",
            "@platforms//os:linux": name + "_linux",
            "@platforms//os:windows": name + "_windows",
            "//conditions:default": name + "_linux",
        }),
        visibility = ["//visibility:public"],
    )

def buildifier_config(name = "buildifier"):
    buildifier(
        name = name,
        lint_mode = "fix",
        mode = "fix",
    )

    buildifier(
        name = name + ".check",
        lint_mode = "warn",
        mode = "diff",
    )
