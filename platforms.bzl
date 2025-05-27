"""Platforms for the build."""

def platforms():
    """Define platform constraints."""

    native.platform(
        name = "linux_amd64_platform",
        constraint_values = ["@platforms//os:linux", "@platforms//cpu:x86_64"],
    )

    native.platform(
        name = "linux_arm64_platform",
        constraint_values = ["@platforms//os:linux", "@platforms//cpu:arm64"],
    )

    native.platform(
        name = "darwin_amd64_platform",
        constraint_values = ["@platforms//os:osx", "@platforms//cpu:x86_64"],
    )

    native.platform(
        name = "darwin_arm64_platform",
        constraint_values = ["@platforms//os:osx", "@platforms//cpu:arm64"],
    )

    native.platform(
        name = "windows_amd64_platform",
        constraint_values = ["@platforms//os:windows", "@platforms//cpu:x86_64"],
    )

    native.platform(
        name = "windows_arm64_platform",
        constraint_values = ["@platforms//os:windows", "@platforms//cpu:arm64"],
    )
