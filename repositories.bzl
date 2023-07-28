load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_repositories():
    go_repository(
        name = "go_sys",
        build_file_proto_mode = "disable_global",
        importpath = "golang.org/x/sys",
        sum = "h1:SqMFp9UcQJZa+pmYuAKjd9xq1f0j5rLcDIk0mj4qAsA=",
        version = "v0.10.0",
    )
