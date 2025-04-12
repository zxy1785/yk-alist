#!/bin/bash

appName="alist"
builtAt="$(date +'%F %T %z')"
goVersion=$(go version | sed 's/go version //')
gitAuthor="Xhofe <i@nn.ci>"
gitCommit=$(git log --pretty=format:"%h" -1)
version=$(git describe --long --tags --dirty --always)
webVersion=$(wget -qO- -t1 -T2 "https://api.github.com/repos/ykxVK8yL5L/alist-web/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')

ldflags="\
-w -s \
-X 'github.com/alist-org/alist/v3/internal/conf.BuiltAt=$builtAt' \
-X 'github.com/alist-org/alist/v3/internal/conf.GoVersion=$goVersion' \
-X 'github.com/alist-org/alist/v3/internal/conf.GitAuthor=$gitAuthor' \
-X 'github.com/alist-org/alist/v3/internal/conf.GitCommit=$gitCommit' \
-X 'github.com/alist-org/alist/v3/internal/conf.Version=$version' \
-X 'github.com/alist-org/alist/v3/internal/conf.WebVersion=$webVersion' \
"


OS_ARCHES=(amd64 arm64 i386)
GO_ARCHES=(amd64 arm64 386)
CGO_ARGS=(x86_64-unknown-freebsd14.1 aarch64-unknown-freebsd14.1 i386-unknown-freebsd14.1)


os_arch=${OS_ARCHES[0]}
cgo_cc="clang --target=${CGO_ARGS[0]} --sysroot=/opt/freebsd/${os_arch}"
echo building for freebsd-${os_arch}
sudo mkdir -p "/opt/freebsd/${os_arch}"
wget -q https://download.freebsd.org/releases/${os_arch}/14.1-RELEASE/base.txz
sudo tar -xf ./base.txz -C /opt/freebsd/${os_arch}
rm base.txz
export GOOS=freebsd
export GOARCH=${GO_ARCHES[0]}
export CC=${cgo_cc}
export CGO_ENABLED=1
export CGO_LDFLAGS="-fuse-ld=lld"


go build -ldflags="$ldflags" -tags=jsoniter .