load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

#
# Go rules
#
http_archive(
    name = "io_bazel_rules_go",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/0.19.0/rules_go-0.19.0.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/0.19.0/rules_go-0.19.0.tar.gz",
    ],
    sha256 = "9fb16af4d4836c8222142e54c9efa0bb5fc562ffc893ce2abeac3e25daead144",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

#
# Gazelle setup for Go rule generation
#
http_archive(
    name = "bazel_gazelle",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/0.18.1/bazel-gazelle-0.18.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.18.1/bazel-gazelle-0.18.1.tar.gz",
    ],
    sha256 = "be9296bfd64882e3c08e3283c58fcb461fa6dd3c171764fcc4cf322f60615a9b",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

#
# Protobuf setup
#
git_repository(
    name = "com_google_protobuf",
    commit = "09745575a923640154bcf307fba8aedff47f240a",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1558721209 -0700",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

gazelle_dependencies()

#
# Docker setup
#
# Download the rules_docker repository at release v0.8.1
http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "87fc6a2b128147a0a3039a2fd0b53cc1f2ed5adb8716f50756544a572999ae9a",
    strip_prefix = "rules_docker-0.8.1",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.8.1.tar.gz"],
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

load("@io_bazel_rules_docker//container:container.bzl", "container_pull")

container_pull(
    name = "nonroot_base",
    registry = "gcr.io",
    repository = "distroless/base",
    digest = "sha256:559425d074a4632c650930338074b4ec6129dec84942ac449d3777451737d680",
)

container_pull(
    name = "go_debug_image",
    registry = "gcr.io",
    repository = "distroless/base",
    digest = "sha256:1a7001e2ffde479e0bfb1c6f490659bbe4c62d385289d197f807e70dd8918ff1",
)

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",
    importpath = "google.golang.org/grpc",
    sum = "h1:TrBcJ1yqAl1G++wO39nD/qtgpsW9/1+QGrluyMGEYgM=",
    version = "v1.19.1",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    sum = "h1:R/3boaszxrf1GEUWTVDzSKVwLmSJpwZ1yqXm8j0v2QI=",
    version = "v0.0.0-20190620200207-3b0461eec859",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    sum = "h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=",
    version = "v0.3.2",
)

go_repository(
    name = "co_honnef_go_tools",
    importpath = "honnef.co/go/tools",
    sum = "h1:XJP7lxbSxWLOMNdBE4B/STaqVy6L73o0knwj2vIlxnw=",
    version = "v0.0.0-20190102054323-c2f93a96b099",
)

go_repository(
    name = "com_github_alecthomas_template",
    importpath = "github.com/alecthomas/template",
    sum = "h1:cAKDfWh5VpdgMhJosfJnn5/FoN2SRZ4p7fJNX58YPaU=",
    version = "v0.0.0-20160405071501-a0175ee3bccc",
)

go_repository(
    name = "com_github_alecthomas_units",
    importpath = "github.com/alecthomas/units",
    sum = "h1:qet1QNfXsQxTZqLG4oE62mJzwPIB8+Tee4RNCL9ulrY=",
    version = "v0.0.0-20151022065526-2efee857e7cf",
)

go_repository(
    name = "com_github_apache_thrift",
    importpath = "github.com/apache/thrift",
    sum = "h1:pODnxUFNcjP9UTLZGTdeh+j16A8lJbRvD3rOtrk/7bs=",
    version = "v0.12.0",
)

go_repository(
    name = "com_github_azure_azure_sdk_for_go",
    importpath = "github.com/Azure/azure-sdk-for-go",
    sum = "h1:5SzgnfAvUBdBwNTN23WLfZoCt/rGhLvd7QdCAaFXgY4=",
    version = "v31.1.0+incompatible",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest",
    importpath = "github.com/Azure/go-autorest/autorest",
    sum = "h1:Mlm9qy2fpQ9MvfyI41G2Zf5B4CsgjjNbLOWszfK6KrY=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_adal",
    importpath = "github.com/Azure/go-autorest/autorest/adal",
    sum = "h1:7IBDu1jgh+ADHXnEYExkV9RE/ztOOlxdACkkPRthGKw=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_azure_auth",
    importpath = "github.com/Azure/go-autorest/autorest/azure/auth",
    sum = "h1:YgO/vSnJEc76NLw2ecIXvXa8bDWiqf1pOJzARAoZsYU=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_azure_cli",
    importpath = "github.com/Azure/go-autorest/autorest/azure/cli",
    sum = "h1:YTtBrcb6mhA+PoSW8WxFDoIIyjp13XqJeX80ssQtri4=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_date",
    importpath = "github.com/Azure/go-autorest/autorest/date",
    sum = "h1:YGrhWfrgtFs84+h0o46rJrlmsZtyZRg470CqAXTZaGM=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_mocks",
    importpath = "github.com/Azure/go-autorest/autorest/mocks",
    sum = "h1:Kx+AUU2Te+A3JIyYn6Dfs+cFgx5XorQKuIXrZGoq/SI=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_to",
    importpath = "github.com/Azure/go-autorest/autorest/to",
    sum = "h1:nQOZzFCudTh+TvquAtCRjM01VEYx85e9qbwt5ncW4L8=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_validation",
    importpath = "github.com/Azure/go-autorest/autorest/validation",
    sum = "h1:ISSNzGUh+ZSzizJWOWzs8bwpXIePbGLW4z/AmUFGH5A=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_azure_go_autorest_logger",
    importpath = "github.com/Azure/go-autorest/logger",
    sum = "h1:ruG4BSDXONFRrZZJ2GUXDiUyVpayPmb1GnWeHDdaNKY=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_azure_go_autorest_tracing",
    importpath = "github.com/Azure/go-autorest/tracing",
    sum = "h1:TRBxC5Pj/fIuh4Qob0ZpkggbfT8RC0SubHbpV3p4/Vc=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_beorn7_perks",
    importpath = "github.com/beorn7/perks",
    sum = "h1:xJ4a3vCFaGF/jqvzLMYoU8P317H5OQ+Via4RmuPwCS0=",
    version = "v0.0.0-20180321164747-3a771d992973",
)

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_census_instrumentation_opencensus_proto",
    build_extra_args = ["-exclude=src"],
    importpath = "github.com/census-instrumentation/opencensus-proto",
    sum = "h1:LzQXZOgg4CQfE6bFvXGM30YZL1WW/M337pXml+GrcZ4=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_client9_misspell",
    importpath = "github.com/client9/misspell",
    sum = "h1:ta993UF76GwbvJcIo3Y68y/M3WxlpEHPWIGDkJYwzJI=",
    version = "v0.3.4",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_dgrijalva_jwt_go",
    importpath = "github.com/dgrijalva/jwt-go",
    sum = "h1:7qlOGliEKZXTDg6OTjfoBKDXWrumCAMpl/TFQ4/5kLM=",
    version = "v3.2.0+incompatible",
)

go_repository(
    name = "com_github_dimchansky_utfbom",
    importpath = "github.com/dimchansky/utfbom",
    sum = "h1:FcM3g+nofKgUteL8dm/UpdRXNC9KmADgTpLKsu0TRo4=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_eapache_go_resiliency",
    importpath = "github.com/eapache/go-resiliency",
    sum = "h1:1NtRmCAqadE2FN4ZcN6g90TP3uk8cg9rn9eNK2197aU=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_eapache_go_xerial_snappy",
    importpath = "github.com/eapache/go-xerial-snappy",
    sum = "h1:YEetp8/yCZMuEPMUDHG0CW/brkkEp8mzqk2+ODEitlw=",
    version = "v0.0.0-20180814174437-776d5712da21",
)

go_repository(
    name = "com_github_eapache_queue",
    importpath = "github.com/eapache/queue",
    sum = "h1:YOEu7KNc61ntiQlcEeUIoDTJ2o8mQznoNvUhiigpIqc=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_evanphx_json_patch",
    importpath = "github.com/evanphx/json-patch",
    sum = "h1:ouOWdg56aJriqS0huScTkVXPC5IcNrDCXZ6OoTAWu7M=",
    version = "v4.5.0+incompatible",
)

go_repository(
    name = "com_github_fsnotify_fsnotify",
    importpath = "github.com/fsnotify/fsnotify",
    sum = "h1:IXs+QLmnXW2CcXuY+8Mzv/fWEsPGWxqefPtCP5CnV9I=",
    version = "v1.4.7",
)

go_repository(
    name = "com_github_ghodss_yaml",
    importpath = "github.com/ghodss/yaml",
    sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_kit_kit",
    importpath = "github.com/go-kit/kit",
    sum = "h1:Wz+5lgoB0kkuqLEc6NVmwRknTKP6dTGbSqvhZtBI/j0=",
    version = "v0.8.0",
)

go_repository(
    name = "com_github_go_logfmt_logfmt",
    importpath = "github.com/go-logfmt/logfmt",
    sum = "h1:MP4Eh7ZCb31lleYCFuwm0oe4/YGak+5l1vA2NOE80nA=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_go_logr_logr",
    importpath = "github.com/go-logr/logr",
    sum = "h1:M1Tv3VzNlEHg6uyACnRdtrploV2P7wZqH8BoQMtz0cg=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_go_logr_zapr",
    importpath = "github.com/go-logr/zapr",
    sum = "h1:h+WVe9j6HAA01niTJPA/kKH0i7e0rLZBCwauQFcRE54=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_go_stack_stack",
    importpath = "github.com/go-stack/stack",
    sum = "h1:5SgMzNM5HxrEjV0ww2lTmX6E2Izsfxas4+YHWRs3Lsk=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_gogo_protobuf",
    importpath = "github.com/gogo/protobuf",
    sum = "h1:/s5zKNz0uPFCZ5hddgPdo2TK2TVrUNMn0OOX8/aZMTE=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_golang_glog",
    importpath = "github.com/golang/glog",
    sum = "h1:VKtxabqXZkF25pY9ekfRL6a582T4P37/31XEstQ5p58=",
    version = "v0.0.0-20160126235308-23def4e6c14b",
)

go_repository(
    name = "com_github_golang_groupcache",
    importpath = "github.com/golang/groupcache",
    sum = "h1:u4bArs140e9+AfE52mFHOXVFnOSBJBRlzTHrOPLOIhE=",
    version = "v0.0.0-20180513044358-24b0969c4cb7",
)

go_repository(
    name = "com_github_golang_mock",
    importpath = "github.com/golang/mock",
    sum = "h1:qGJ6qTW+x6xX/my+8YUVl4WNpX9B7+/l2tRsHGZ7f2s=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_golang_protobuf",
    importpath = "github.com/golang/protobuf",
    sum = "h1:YF8+flBXS5eO826T4nzqPrxfhQThhXl0YzfuUPu4SBg=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_golang_snappy",
    importpath = "github.com/golang/snappy",
    sum = "h1:woRePGFeVFfLKN/pOkfl+p/TAqKOfFu+7KPlMVpok/w=",
    version = "v0.0.0-20180518054509-2e65f85255db",
)

go_repository(
    name = "com_github_google_go_cmp",
    importpath = "github.com/google/go-cmp",
    sum = "h1:crn/baboCvb5fXaQ0IJ1SGTsTVrWpDsCWC8EGETZijY=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_google_gofuzz",
    importpath = "github.com/google/gofuzz",
    sum = "h1:+RRA9JqSOZFfKrOeqr2z77+8R2RKyh8PG66dcu1V0ck=",
    version = "v0.0.0-20170612174753-24818f796faf",
)

go_repository(
    name = "com_github_googleapis_gnostic",
    importpath = "github.com/googleapis/gnostic",
    sum = "h1:l6N3VoaVzTncYYW+9yOz2LJJammFZGBO13sqgEhpy9g=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_gorilla_context",
    importpath = "github.com/gorilla/context",
    sum = "h1:AWwleXJkX/nhcU9bZSnZoi3h/qGYqQAGhq6zZe/aQW8=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_gorilla_mux",
    importpath = "github.com/gorilla/mux",
    sum = "h1:Pgr17XVTNXAk3q/r4CpKzC5xBM/qW1uVLV+IhRZpIIk=",
    version = "v1.6.2",
)

go_repository(
    name = "com_github_grpc_ecosystem_grpc_gateway",
    importpath = "github.com/grpc-ecosystem/grpc-gateway",
    sum = "h1:2+KSC78XiO6Qy0hIjfc1OD9H+hsaJdJlb8Kqsd41CTE=",
    version = "v1.8.5",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    importpath = "github.com/hashicorp/golang-lru",
    sum = "h1:CL2msUPvZTLb5O648aiLNJw3hnBxN2+1Jq8rCOH9wdo=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_hpcloud_tail",
    importpath = "github.com/hpcloud/tail",
    sum = "h1:nfCOvKYfkgYP8hkirhJocXT2+zOD8yUNjXaWfTlyFKI=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_imdario_mergo",
    importpath = "github.com/imdario/mergo",
    sum = "h1:xTNEAn+kxVO7dTZGu0CegyqKZmoWFI0rF8UxjlB2d28=",
    version = "v0.3.6",
)

go_repository(
    name = "com_github_json_iterator_go",
    importpath = "github.com/json-iterator/go",
    sum = "h1:MrUvLMLTMxbqFJ9kzlvat/rYZqZnW3u4wkLzWTaFwKs=",
    version = "v1.1.6",
)

go_repository(
    name = "com_github_julienschmidt_httprouter",
    importpath = "github.com/julienschmidt/httprouter",
    sum = "h1:TDTW5Yz1mjftljbcKqRcrYhd4XeOoI98t+9HbQbYf7g=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_kisielk_errcheck",
    importpath = "github.com/kisielk/errcheck",
    sum = "h1:ZqfnKyx9KGpRcW04j5nnPDgRgoXUeLh2YFBeFzphcA0=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_kisielk_gotool",
    importpath = "github.com/kisielk/gotool",
    sum = "h1:AV2c/EiW3KqPNT9ZKl07ehoAGi4C5/01Cfbblndcapg=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_konsorten_go_windows_terminal_sequences",
    importpath = "github.com/konsorten/go-windows-terminal-sequences",
    sum = "h1:mweAR1A6xJ3oS2pRaGiHgQ4OO8tzTaLawm8vnODuwDk=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_kr_logfmt",
    importpath = "github.com/kr/logfmt",
    sum = "h1:T+h1c/A9Gawja4Y9mFVWj2vyii2bbUNDw3kt9VxK2EY=",
    version = "v0.0.0-20140226030751-b84e30acd515",
)

go_repository(
    name = "com_github_kr_pretty",
    importpath = "github.com/kr/pretty",
    sum = "h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_kr_pty",
    importpath = "github.com/kr/pty",
    sum = "h1:VkoXIwSboBpnk99O/KFauAEILuNHv5DVFKZMBN/gUgw=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_kr_text",
    importpath = "github.com/kr/text",
    sum = "h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
    sum = "h1:4hp9jkHxhMHkqkrB3Ix0jegS5sx/RkqARlsWZ6pIwiU=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_mitchellh_go_homedir",
    importpath = "github.com/mitchellh/go-homedir",
    sum = "h1:lukF9ziXFxDFPkA1vsr5zpc1XuPDn/wFntq5mG+4E0Y=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_modern_go_concurrent",
    importpath = "github.com/modern-go/concurrent",
    sum = "h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=",
    version = "v0.0.0-20180306012644-bacd9c7ef1dd",
)

go_repository(
    name = "com_github_modern_go_reflect2",
    importpath = "github.com/modern-go/reflect2",
    sum = "h1:9f412s+6RmYXLWZSEzVVgPGK7C2PphHj5RJrvfx9AWI=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_mwitkow_go_conntrack",
    importpath = "github.com/mwitkow/go-conntrack",
    sum = "h1:F9x/1yl3T2AeKLr2AMdilSD8+f9bvMnNN8VS5iDtovc=",
    version = "v0.0.0-20161129095857-cc309e4a2223",
)

go_repository(
    name = "com_github_onsi_ginkgo",
    importpath = "github.com/onsi/ginkgo",
    sum = "h1:VkHVNpR4iVnU8XQR6DBm8BqYjN7CRzw+xKUbVVbbW9w=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_onsi_gomega",
    importpath = "github.com/onsi/gomega",
    sum = "h1:izbySO9zDPmjJ8rDjLvkA2zJHIo+HkYXHnf7eN7SSyo=",
    version = "v1.5.0",
)

go_repository(
    name = "com_github_openzipkin_zipkin_go",
    importpath = "github.com/openzipkin/zipkin-go",
    sum = "h1:yXiysv1CSK7Q5yjGy1710zZGnsbMUIjluWBxtLXHPBo=",
    version = "v0.1.6",
)

go_repository(
    name = "com_github_pborman_uuid",
    importpath = "github.com/pborman/uuid",
    sum = "h1:MUyE44mTvnI5A0xrxIxaMqoWFzPfQvtE2IWUollMDMs=",
    version = "v0.0.0-20170612153648-e790cca94e6c",
)

go_repository(
    name = "com_github_pierrec_lz4",
    importpath = "github.com/pierrec/lz4",
    sum = "h1:2xWsjqPFWcplujydGg4WmhC/6fZqK42wMM8aXeqhl0I=",
    version = "v2.0.5+incompatible",
)

go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    sum = "h1:iURUrRGxPUNPdy5/HRSm+Yj6okJ6UtLINN0Q9M4+h3I=",
    version = "v0.8.1",
)

go_repository(
    name = "com_github_pmezard_go_difflib",
    importpath = "github.com/pmezard/go-difflib",
    sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_prometheus_client_golang",
    importpath = "github.com/prometheus/client_golang",
    sum = "h1:D+CiwcpGTW6pL6bv6KI3KbyEyCKyS+1JWS2h8PNDnGA=",
    version = "v0.9.3-0.20190127221311-3c4408c8b829",
)

go_repository(
    name = "com_github_prometheus_client_model",
    importpath = "github.com/prometheus/client_model",
    sum = "h1:BVwpUVJDADN2ufcGik7W992pyps0wZ888b/y9GXcLTU=",
    version = "v0.0.0-20190115171406-56726106282f",
)

go_repository(
    name = "com_github_prometheus_common",
    importpath = "github.com/prometheus/common",
    sum = "h1:kUZDBDTdBVBYBj5Tmh2NZLlF60mfjA27rM34b+cVwNU=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_prometheus_procfs",
    importpath = "github.com/prometheus/procfs",
    sum = "h1:/K3IL0Z1quvmJ7X0A1AwNEK7CRkVK3YwfOU/QAL4WGg=",
    version = "v0.0.0-20190117184657-bf6a532e95b1",
)

go_repository(
    name = "com_github_rcrowley_go_metrics",
    importpath = "github.com/rcrowley/go-metrics",
    sum = "h1:9ZKAASQSHhDYGoxY8uLVpewe1GDZ2vu2Tr/vTdVAkFQ=",
    version = "v0.0.0-20181016184325-3113b8401b8a",
)

go_repository(
    name = "com_github_rogpeppe_fastuuid",
    importpath = "github.com/rogpeppe/fastuuid",
    sum = "h1:INyGLmTCMGFr6OVIb977ghJvABML2CMVjPoRfNDdYDo=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_shopify_sarama",
    importpath = "github.com/Shopify/sarama",
    sum = "h1:9oksLxC6uxVPHPVYUmq6xhr1BOF/hHobWH2UzO67z1s=",
    version = "v1.19.0",
)

go_repository(
    name = "com_github_shopify_toxiproxy",
    importpath = "github.com/Shopify/toxiproxy",
    sum = "h1:TKdv8HiTLgE5wdJuEML90aBgNWsokNbMijUGhmcoBJc=",
    version = "v2.1.4+incompatible",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    importpath = "github.com/sirupsen/logrus",
    sum = "h1:juTguoYk5qI21pwyTXY3B3Y5cOTH3ZUyZCg1v/mihuo=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_spf13_afero",
    importpath = "github.com/spf13/afero",
    sum = "h1:5jhuqJyZCZf2JRofRvN/nIFgIWNzPa3/Vz8mYylgbWc=",
    version = "v1.2.2",
)

go_repository(
    name = "com_github_spf13_pflag",
    importpath = "github.com/spf13/pflag",
    sum = "h1:zPAT6CGy6wXeQ7NtTnaTerfKOsV6V6F8agHXFiazDkg=",
    version = "v1.0.3",
)

go_repository(
    name = "com_github_stretchr_objx",
    importpath = "github.com/stretchr/objx",
    sum = "h1:2vfRuCMp5sSVIDSqO8oNnWJq7mPa6KVP3iPIwFBuy8A=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_stretchr_testify",
    importpath = "github.com/stretchr/testify",
    sum = "h1:TivCn/peBQ7UY8ooIcPgZFpTNSz0Q2U6UrFlUfqbe0Q=",
    version = "v1.3.0",
)

go_repository(
    name = "com_google_cloud_go",
    importpath = "cloud.google.com/go",
    sum = "h1:eOI3/cP2VTU6uZLDYAoic+eyzzB9YyGmJ7eIjl8rOPg=",
    version = "v0.34.0",
)

go_repository(
    name = "in_gopkg_alecthomas_kingpin_v2",
    importpath = "gopkg.in/alecthomas/kingpin.v2",
    sum = "h1:jMFz6MfLP0/4fUyZle81rXUoxOBFi19VUFKVDOQfozc=",
    version = "v2.2.6",
)

go_repository(
    name = "in_gopkg_check_v1",
    importpath = "gopkg.in/check.v1",
    sum = "h1:qIbj1fsPNlZgppZ+VLlY7N33q108Sa+fhmuc+sWQYwY=",
    version = "v1.0.0-20180628173108-788fd7840127",
)

go_repository(
    name = "in_gopkg_fsnotify_v1",
    importpath = "gopkg.in/fsnotify.v1",
    sum = "h1:xOHLXZwVvI9hhs+cLKq5+I5onOuwQLhQwiu63xxlHs4=",
    version = "v1.4.7",
)

go_repository(
    name = "in_gopkg_inf_v0",
    importpath = "gopkg.in/inf.v0",
    sum = "h1:73M5CoZyi3ZLMOyDlQh031Cx6N9NDJ2Vvfl76EDAgDc=",
    version = "v0.9.1",
)

go_repository(
    name = "in_gopkg_resty_v1",
    importpath = "gopkg.in/resty.v1",
    sum = "h1:CuXP0Pjfw9rOuY6EP+UvtNvt5DSqHpIxILZKT/quCZI=",
    version = "v1.12.0",
)

go_repository(
    name = "in_gopkg_tomb_v1",
    importpath = "gopkg.in/tomb.v1",
    sum = "h1:uRGJdciOHaEIrze2W8Q3AKkepLTh2hOroT7a+7czfdQ=",
    version = "v1.0.0-20141024135613-dd632973f1e7",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    sum = "h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=",
    version = "v2.2.2",
)

go_repository(
    name = "io_k8s_api",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/api",
    sum = "h1:aBGgKJUM9Hk/3AE8WaZIApnTxG35kbuQba2w+SXqezo=",
    version = "v0.0.0-20190409021203-6e4e0e4f393b",
)

go_repository(
    name = "io_k8s_apiextensions_apiserver",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/apiextensions-apiserver",
    sum = "h1:q1Qvjzs/iEdXF6A1a8H3AKVFDzJNcJn3nXMs6R6qFtA=",
    version = "v0.0.0-20190409022649-727a075fdec8",
)

go_repository(
    name = "io_k8s_apimachinery",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/apimachinery",
    sum = "h1:Jmdtdt1ZnoGfWWIIik61Z7nKYgO3J+swQJtPYsP9wHA=",
    version = "v0.0.0-20190404173353-6a84e37a896d",
)

go_repository(
    name = "io_k8s_client_go",
    importpath = "k8s.io/client-go",
    sum = "h1:U5Bt+dab9K8qaUmXINrkXO135kA11/i5Kg1RUydgaMQ=",
    version = "v11.0.1-0.20190409021438-1a26190bd76a+incompatible",
)

go_repository(
    name = "io_k8s_klog",
    importpath = "k8s.io/klog",
    sum = "h1:0VPpR+sizsiivjIfIAQH/rl8tan6jvWkS7lU+0di3lE=",
    version = "v0.3.0",
)

go_repository(
    name = "io_k8s_kube_openapi",
    importpath = "k8s.io/kube-openapi",
    sum = "h1:3KSCztE7gPitlZmWbNwue/2U0YruD65DqX3INopDAQM=",
    version = "v0.0.0-20180731170545-e3762e86a74c",
)

go_repository(
    name = "io_k8s_sigs_controller_runtime",
    importpath = "sigs.k8s.io/controller-runtime",
    replace = "github.com/alexeldeib/controller-runtime",
    sum = "h1:FmuzrUfhIdK+fXoCG9TXragSdiGvOu+5DNvAjUmwWrw=",
    version = "v0.2.0-beta.4-master-opts",
)

go_repository(
    name = "io_k8s_sigs_testing_frameworks",
    importpath = "sigs.k8s.io/testing_frameworks",
    sum = "h1:cP2l8fkA3O9vekpy5Ks8mmA0NW/F7yBdXf8brkWhVrs=",
    version = "v0.1.1",
)

go_repository(
    name = "io_k8s_sigs_yaml",
    importpath = "sigs.k8s.io/yaml",
    sum = "h1:4A07+ZFc2wgJwo8YNlQpr1rVlgUDlxXHhPJciaPY5gs=",
    version = "v1.1.0",
)

go_repository(
    name = "io_k8s_utils",
    importpath = "k8s.io/utils",
    sum = "h1:VBM/0P5TWxwk+Nw6Z+lAw3DKgO76g90ETOiA6rfLV1Y=",
    version = "v0.0.0-20190506122338-8fab8cb257d5",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    sum = "h1:NAfh7zF0/3/HqtMvJNZ/RFrSlCE6ZTlHmKfhL/Dm1Jk=",
    version = "v0.20.2",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_ocagent",
    importpath = "contrib.go.opencensus.io/exporter/ocagent",
    sum = "h1:jGFvw3l57ViIVEPKKEUXPcLYIXJmQxLUh6ey1eJhwyc=",
    version = "v0.4.12",
)

go_repository(
    name = "org_golang_google_api",
    importpath = "google.golang.org/api",
    sum = "h1:oJra/lMfmtm13/rgY/8i3MzjFWYXvQIAKjQ3HqofMk8=",
    version = "v0.3.1",
)

go_repository(
    name = "org_golang_google_appengine",
    importpath = "google.golang.org/appengine",
    sum = "h1:/wp5JvzpHIxhs/dumFmF7BXTf3Z+dd4uXta4kVyO508=",
    version = "v1.4.0",
)

go_repository(
    name = "org_golang_google_genproto",
    importpath = "google.golang.org/genproto",
    sum = "h1:Lj2SnHtxkRGJDqnGaSjo+CCdIieEnwVazbOXILwQemk=",
    version = "v0.0.0-20190307195333-5fe7a883aa19",
)

go_repository(
    name = "org_golang_x_crypto",
    importpath = "golang.org/x/crypto",
    sum = "h1:rlLehGeYg6jfoyz/eDqDU1iRXLKfR42nnNh57ytKEWo=",
    version = "v0.0.0-20190506204251-e1dfcc566284",
)

go_repository(
    name = "org_golang_x_exp",
    importpath = "golang.org/x/exp",
    sum = "h1:c2HOrn5iMezYjSlGPncknSEr/8x5LELb/ilJbXi9DEA=",
    version = "v0.0.0-20190121172915-509febef88a4",
)

go_repository(
    name = "org_golang_x_lint",
    importpath = "golang.org/x/lint",
    sum = "h1:hX65Cu3JDlGH3uEdK7I99Ii+9kjD6mvnnpfLdEAH0x4=",
    version = "v0.0.0-20190301231843-5614ed5bae6f",
)

go_repository(
    name = "org_golang_x_oauth2",
    importpath = "golang.org/x/oauth2",
    sum = "h1:Wo7BWFiOk0QRFMLYMqJGFMd9CgUAcGx7V+qEg/h5IBI=",
    version = "v0.0.0-20190226205417-e64efc72b421",
)

go_repository(
    name = "org_golang_x_sync",
    importpath = "golang.org/x/sync",
    sum = "h1:8gQV6CLnAEikrhgkHFbMAEhagSSnXWGV915qUMm9mrU=",
    version = "v0.0.0-20190423024810-112230192c58",
)

go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    sum = "h1:hDn6jm7snBX2O7+EeTk6Q4WXJfKt7MWgtiCCRi1rBoY=",
    version = "v0.0.0-20190508220229-2d0786266e9c",
)

go_repository(
    name = "org_golang_x_time",
    importpath = "golang.org/x/time",
    sum = "h1:+DCIGbF/swA92ohVg0//6X2IVY3KZs6p9mix0ziNYJM=",
    version = "v0.0.0-20180412165947-fbb02b2291d2",
)

go_repository(
    name = "org_golang_x_tools",
    importpath = "golang.org/x/tools",
    sum = "h1:D7DVZUZEUgsSIDTivnUtVeGfN5AvhDIKtdIZAqx0ieE=",
    version = "v0.0.0-20190501045030-23463209683d",
)

go_repository(
    name = "org_uber_go_atomic",
    importpath = "go.uber.org/atomic",
    sum = "h1:2Oa65PReHzfn29GpvgsYwloV9AVFHPDk8tYxt2c2tr4=",
    version = "v1.3.2",
)

go_repository(
    name = "org_uber_go_multierr",
    importpath = "go.uber.org/multierr",
    sum = "h1:HoEmRHQPVSqub6w2z2d2EOVs2fjyFRGyofhKuyDq0QI=",
    version = "v1.1.0",
)

go_repository(
    name = "org_uber_go_zap",
    importpath = "go.uber.org/zap",
    sum = "h1:XCJQEf3W6eZaVwhRBof6ImoYGJSITeKWsyeh3HFu/5o=",
    version = "v1.9.1",
)

go_repository(
    name = "xyz_gomodules_jsonpatch_v2",
    importpath = "gomodules.xyz/jsonpatch/v2",
    sum = "h1:OyHbl+7IOECpPKfVK42oFr6N7+Y2dR+Jsb/IiDV3hOo=",
    version = "v2.0.0",
)

go_repository(
    name = "com_github_sanity_io_litter",
    importpath = "github.com/sanity-io/litter",
    sum = "h1:BllcKWa3VbZmOZbDCoszYLk7zCsKHz5Beossi8SUcTc=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_hashicorp_errwrap",
    importpath = "github.com/hashicorp/errwrap",
    sum = "h1:hLrqtEDnRye3+sgx6z4qVLNuviH3MR5aQ0ykNJa/UYA=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_go_multierror",
    importpath = "github.com/hashicorp/go-multierror",
    sum = "h1:iVjPR7a6H0tWELX5NxNe7bYopibicUzc7uPribsnS6o=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_fatih_color",
    importpath = "github.com/fatih/color",
    sum = "h1:DkWD4oS2D8LGGgTQ6IvwJJXSL5Vp2ffcQg58nFV38Ys=",
    version = "v1.7.0",
)

go_repository(
    name = "com_github_gobuffalo_flect",
    importpath = "github.com/gobuffalo/flect",
    sum = "h1:xpKq9ap8MbYfhuPCF0dBH854Gp9CxZjr/IocxELFflo=",
    version = "v0.1.5",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    importpath = "github.com/inconshreveable/mousetrap",
    sum = "h1:Z8tu5sraLXCXIcARxBp/8cbvlwVa7Z1NHg9XEKhtSvM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    importpath = "github.com/mattn/go-colorable",
    sum = "h1:/bC9yWikZXAL9uJdulbSfyVNIR3n3trXl+v8+1sx8mU=",
    version = "v0.1.2",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    importpath = "github.com/mattn/go-isatty",
    sum = "h1:HLtExJ+uU2HOZ+wI0Tt5DtUDrx8yhUqDcp7fYERX4CE=",
    version = "v0.0.8",
)

go_repository(
    name = "com_github_spf13_cobra",
    importpath = "github.com/spf13/cobra",
    sum = "h1:ZlrZ4XsMRm04Fr5pSFxBgfND2EBVa1nLpiy1stUsX/8=",
    version = "v0.0.3",
)

go_repository(
    name = "io_k8s_sigs_controller_tools",
    importpath = "sigs.k8s.io/controller-tools",
    sum = "h1:W+coTe+nkVNclQrikwlRp6GJKwgcrHzvIQZ9kCaak5A=",
    version = "v0.2.0-beta.4",
)

go_repository(
    name = "com_github_alcortesm_tgz",
    importpath = "github.com/alcortesm/tgz",
    sum = "h1:uSoVVbwJiQipAclBbw+8quDsfcvFjOpI5iCf4p/cqCs=",
    version = "v0.0.0-20161220082320-9c5fe88206d7",
)

go_repository(
    name = "com_github_anmitsu_go_shlex",
    importpath = "github.com/anmitsu/go-shlex",
    sum = "h1:kFOfPq6dUM1hTo4JG6LR5AXSUEsOjtdm0kw0FtQtMJA=",
    version = "v0.0.0-20161002113705-648efa622239",
)

go_repository(
    name = "com_github_apex_log",
    importpath = "github.com/apex/log",
    sum = "h1:BwhRZ0qbjYtTob0I+2M+smavV0kOC8XgcnGZcyL9liA=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_aphistic_golf",
    importpath = "github.com/aphistic/golf",
    sum = "h1:2KLQMJ8msqoPHIPDufkxVcoTtcmE5+1sL9950m4R9Pk=",
    version = "v0.0.0-20180712155816-02c07f170c5a",
)

go_repository(
    name = "com_github_aphistic_sweet",
    importpath = "github.com/aphistic/sweet",
    sum = "h1:I4z+fAUqvKfvZV/CHi5dV0QuwbmIvYYFDjG0Ss5QpAs=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_autonomy_conform",
    importpath = "github.com/autonomy/conform",
    sum = "h1:6kNvr8yfygieKIK4VzTSU4NXHdMxP0zAImw9u6beIao=",
    version = "v0.1.0-alpha.16",
)

go_repository(
    name = "com_github_aws_aws_sdk_go",
    importpath = "github.com/aws/aws-sdk-go",
    sum = "h1:kmy4Gvdlyez1fV4kw5RYxZzWKVyuHZHgPWeU/YvRsV4=",
    version = "v1.20.6",
)

go_repository(
    name = "com_github_aybabtme_rgbterm",
    importpath = "github.com/aybabtme/rgbterm",
    sum = "h1:WWB576BN5zNSZc/M9d/10pqEx5VHNhaQ/yOVAkmj5Yo=",
    version = "v0.0.0-20170906152045-cc83f3b3ce59",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    importpath = "github.com/deckarep/golang-set",
    sum = "h1:SCQV0S6gTtp6itiFrTqI+pfmJ4LN85S1YzhDf9rTHJQ=",
    version = "v1.7.1",
)

go_repository(
    name = "com_github_flynn_go_shlex",
    importpath = "github.com/flynn/go-shlex",
    sum = "h1:BHsljHzVlRcyQhjrss6TZTdY2VfCqZPbv5k3iBFa2ZQ=",
    version = "v0.0.0-20150515145356-3f9db97f8568",
)

go_repository(
    name = "com_github_gliderlabs_ssh",
    importpath = "github.com/gliderlabs/ssh",
    sum = "h1:j3L6gSLQalDETeEg/Jg0mGY0/y/N6zI2xX1978P0Uqw=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_google_go_github",
    importpath = "github.com/google/go-github",
    sum = "h1:N0LgJ1j65A7kfXrZnUDaYCs/Sf4rEjNlfyDHW9dolSY=",
    version = "v17.0.0+incompatible",
)

go_repository(
    name = "com_github_google_go_querystring",
    importpath = "github.com/google/go-querystring",
    sum = "h1:Xkwi/a1rcvNg1PPYe5vI8GbeBY/jrVuDX5ASuANWTrk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_google_uuid",
    importpath = "github.com/google/uuid",
    sum = "h1:Gkbcsh/GbpXz7lPftLA3P6TYMwjCLYm83jiFQZF/3gY=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_hashicorp_hcl",
    importpath = "github.com/hashicorp/hcl",
    sum = "h1:KJWs1uTCkN3E/J5ofCH9Pf8KKsibTFc3fv0CA9+WsVo=",
    version = "v0.0.0-20170509225359-392dba7d905e",
)

go_repository(
    name = "com_github_jbenet_go_context",
    importpath = "github.com/jbenet/go-context",
    sum = "h1:BQSFePA1RWJOlocH6Fxy8MmwDt+yVQYULKfN0RoTN8A=",
    version = "v0.0.0-20150711004518-d14ea06fba99",
)

go_repository(
    name = "com_github_jmespath_go_jmespath",
    importpath = "github.com/jmespath/go-jmespath",
    sum = "h1:pmfjZENx5imkbgOkpRUYLnmbU7UEFbjtDA2hxJ1ichM=",
    version = "v0.0.0-20180206201540-c2b33e8439af",
)

go_repository(
    name = "com_github_jpillora_backoff",
    importpath = "github.com/jpillora/backoff",
    sum = "h1:K//n/AqR5HjG3qxbrBCL4vJPW0MVFSs9CPK1OOJdRME=",
    version = "v0.0.0-20180909062703-3050d21c67d7",
)

go_repository(
    name = "com_github_kevinburke_ssh_config",
    importpath = "github.com/kevinburke/ssh_config",
    sum = "h1:4qB7kGgjot2tlCOW66sJ+ai5tv81oIDM9t6cvyFTKLM=",
    version = "v0.0.0-20170525151105-fa48d7ff1cfb",
)

go_repository(
    name = "com_github_magiconair_properties",
    importpath = "github.com/magiconair/properties",
    sum = "h1:de14gtQSJmD380aERiT7g/BPsA0iDosrVRaU+EhZAio=",
    version = "v1.7.2",
)

go_repository(
    name = "com_github_mgutz_ansi",
    importpath = "github.com/mgutz/ansi",
    sum = "h1:j7+1HpAFS1zy5+Q4qx1fWh90gTKwiN4QCGoY9TWyyO4=",
    version = "v0.0.0-20170206155736-9520e82c474b",
)

go_repository(
    name = "com_github_mingrammer_commonregex",
    importpath = "github.com/mingrammer/commonregex",
    sum = "h1:0nTEyFI+CKWog0IWbyP8jFwgdd+JZ30UYfYce/lG/9w=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_mapstructure",
    importpath = "github.com/mitchellh/mapstructure",
    sum = "h1:W7VHAEVflA5/eTyRvQ53Lz5j8bhRd1myHZlI/IZFvbU=",
    version = "v0.0.0-20170523030023-d0303fe80992",
)

go_repository(
    name = "com_github_montanaflynn_stats",
    importpath = "github.com/montanaflynn/stats",
    sum = "h1:2EkzeTSqBB4V4bJwWrt5gIIrZmpJBcoIRGS2kWLgzmk=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_neurosnap_sentences",
    importpath = "github.com/neurosnap/sentences",
    sum = "h1:iBVUivNtlwGkYsJblWV8GGVFmXzZzak907Ci8aA0VTE=",
    version = "v1.0.6",
)

go_repository(
    name = "com_github_pelletier_go_buffruneio",
    importpath = "github.com/pelletier/go-buffruneio",
    sum = "h1:U4t4R6YkofJ5xHm3dJzuRpPZ0mr5MMCoAWooScCR7aA=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_pelletier_go_toml",
    importpath = "github.com/pelletier/go-toml",
    sum = "h1:QFDlmAXZrfPXEF6c9+15fMqhQIS3O0pxszhnk936vg4=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_satori_go_uuid",
    importpath = "github.com/satori/go.uuid",
    sum = "h1:0uYX9dsZ2yD7q2RtLRtPSdGDWzjeM3TbMJP9utgA0ww=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_sergi_go_diff",
    importpath = "github.com/sergi/go-diff",
    sum = "h1:Kpca3qRNrduNnOQeazBd0ysaKrUJiIuISHxogkT9RPQ=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_smartystreets_assertions",
    importpath = "github.com/smartystreets/assertions",
    sum = "h1:UVQPSSmc3qtTi+zPPkCXvZX9VvW/xT/NsRvKfwY81a8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_smartystreets_go_aws_auth",
    importpath = "github.com/smartystreets/go-aws-auth",
    sum = "h1:hp2CYQUINdZMHdvTdXtPOY2ainKl4IoMcpAXEf2xj3Q=",
    version = "v0.0.0-20180515143844-0c1422d1fdb9",
)

go_repository(
    name = "com_github_smartystreets_gunit",
    importpath = "github.com/smartystreets/gunit",
    sum = "h1:RyPDUFcJbvtXlhJPk7v+wnxZRY2EUokhEYl2EJOPToI=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_spf13_cast",
    importpath = "github.com/spf13/cast",
    sum = "h1:0Rhw4d6C8J9VPu6cjZLIhZ8+aAOHcDvGeKn+cq5Aq3k=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_spf13_jwalterweatherman",
    importpath = "github.com/spf13/jwalterweatherman",
    sum = "h1:RpxSq53NruItMGgp6q5MsDYoZynisJgEpisQdWJ7PyM=",
    version = "v0.0.0-20170523133247-0efa5202c046",
)

go_repository(
    name = "com_github_spf13_viper",
    importpath = "github.com/spf13/viper",
    sum = "h1:qJXvCTq/ZNWp7kk3MGbyDByJaCWFLT7uXIbksXFxITQ=",
    version = "v0.0.0-20170619124313-c1de95864d73",
)

go_repository(
    name = "com_github_src_d_gcfg",
    importpath = "github.com/src-d/gcfg",
    sum = "h1:2BEDr8r0I0b8h/fOqwtxCEiq2HJu8n2JGZJQFGXWLjg=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_tj_assert",
    importpath = "github.com/tj/assert",
    sum = "h1:Rw8kxzWo1mr6FSaYXjQELRe88y2KdfynXdnK72rdjtA=",
    version = "v0.0.0-20171129193455-018094318fb0",
)

go_repository(
    name = "com_github_tj_go_elastic",
    importpath = "github.com/tj/go-elastic",
    sum = "h1:eGaGNxrtoZf/mBURsnNQKDR7u50Klgcf2eFDQEnc8Bc=",
    version = "v0.0.0-20171221160941-36157cbbebc2",
)

go_repository(
    name = "com_github_tj_go_kinesis",
    importpath = "github.com/tj/go-kinesis",
    sum = "h1:m74UWYy+HBs+jMFR9mdZU6shPewugMyH5+GV6LNgW8w=",
    version = "v0.0.0-20171128231115-08b17f58cb1b",
)

go_repository(
    name = "com_github_tj_go_spin",
    importpath = "github.com/tj/go-spin",
    sum = "h1:lhdWZsvImxvZ3q1C5OIB7d72DuOwP4O2NdBg9PyzNds=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_xanzy_ssh_agent",
    importpath = "github.com/xanzy/ssh-agent",
    sum = "h1:lOhdXLxtmYjaHc76ZtNmJWPg948y/RnT+3N3cvKWFzY=",
    version = "v0.1.0",
)

go_repository(
    name = "in_gopkg_jdkato_prose_v2",
    importpath = "gopkg.in/jdkato/prose.v2",
    sum = "h1:yDDVJ347kU7Ro+H2CRgKmAEbFXfHwjyPuXgVd6H+3N8=",
    version = "v2.0.0-20180825173540-767a23049b9e",
)

go_repository(
    name = "in_gopkg_neurosnap_sentences_v1",
    importpath = "gopkg.in/neurosnap/sentences.v1",
    sum = "h1:v7ElyP020iEZQONyLld3fHILHWOPs+ntzuQTNPkul8E=",
    version = "v1.0.6",
)

go_repository(
    name = "in_gopkg_src_d_go_billy_v4",
    importpath = "gopkg.in/src-d/go-billy.v4",
    sum = "h1:iMxwQPj2cuKRyaIZ985zxClkcdTtT5VpXYf4PTJc0Ek=",
    version = "v4.0.1",
)

go_repository(
    name = "in_gopkg_src_d_go_git_fixtures_v3",
    importpath = "gopkg.in/src-d/go-git-fixtures.v3",
    sum = "h1:XWW/s5W18RaJpmo1l0IYGqXKuJITWRFuA45iOf1dKJs=",
    version = "v3.1.1",
)

go_repository(
    name = "in_gopkg_src_d_go_git_v4",
    importpath = "gopkg.in/src-d/go-git.v4",
    sum = "h1:9ZRNKHuhaTaJRGcGaH6Qg7uUORO2X0MNB5WL/CDdqto=",
    version = "v4.0.0",
)

go_repository(
    name = "in_gopkg_warnings_v0",
    importpath = "gopkg.in/warnings.v0",
    sum = "h1:XM28wIgFzaBmeZ5dNHIpWLQpt/9DGKxk+rCg/22nnYE=",
    version = "v0.1.1",
)

go_repository(
    name = "org_gonum_v1_gonum",
    importpath = "gonum.org/v1/gonum",
    sum = "h1:n0S76j3sXlixKdU/XYIBVb1s3hPsZCiZ+oQmPH+r3gQ=",
    version = "v0.0.0-20190119014124-d54847ab4dca",
)

go_repository(
    name = "org_gonum_v1_netlib",
    importpath = "gonum.org/v1/netlib",
    sum = "h1:sWo7pRubEwovyVnbgs3VJpe4ppUGSLbu/zNPyCBXtHE=",
    version = "v0.0.0-20190119082159-9be13e02fd56",
)
