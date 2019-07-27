load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_docker//container:container.bzl", "container_image")
load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_docker//container:container.bzl", "container_push")

# gazelle:proto disable_global
# gazelle:prefix github.com/alexeldeib/incendiary-iguana
gazelle(name = "gazelle")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/alexeldeib/incendiary-iguana",
    visibility = ["//visibility:private"],
    deps = [
        "//api/v1alpha1:go_default_library",
        "//controllers:go_default_library",
        "//pkg/config:go_default_library",
        "@io_k8s_apimachinery//pkg/runtime:go_default_library",
        "@io_k8s_client_go//kubernetes/scheme:go_default_library",
        "@io_k8s_client_go//plugin/pkg/client/auth/gcp:go_default_library",
        "@io_k8s_sigs_controller_runtime//:go_default_library",
        "@io_k8s_sigs_controller_runtime//pkg/log/zap:go_default_library",
    ],
)

go_binary(
    name = "incendiary-iguana",
    embed = [":go_default_library"],
    pure = "on",
    static = "on",
    visibility = ["//visibility:public"],
)

container_image(
    name = "preimage",
    repository = "alexeldeib",
    base = "@nonroot_base//image",
    files = [":incendiary-iguana"],
    cmd = ["/incendiary-iguana"],
)

go_image(
    name = "image",
    srcs = ["main.go"],
    importpath = "github.com/alexeldeib/incendiary-iguana",
    base = "@nonroot_base//image",
    visibility = ["//visibility:public"],
    deps = [
        "//api/v1alpha1:go_default_library",
        "//controllers:go_default_library",
        "//pkg/config:go_default_library",
        "@io_k8s_apimachinery//pkg/runtime:go_default_library",
        "@io_k8s_client_go//kubernetes/scheme:go_default_library",
        "@io_k8s_client_go//plugin/pkg/client/auth/gcp:go_default_library",
        "@io_k8s_sigs_controller_runtime//:go_default_library",
        "@io_k8s_sigs_controller_runtime//pkg/log/zap:go_default_library",
    ],
)

container_push(
    name = "publish",
    image = ":image",
    format = "Docker",
    repository = "alexeldeib/incendiary-iguana",
    tag = "latest",
    registry = "index.docker.io",
)
