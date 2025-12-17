FROM golang:1.25 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /usr/local/bin/castletown ./main.go

FROM debian:13-slim AS rootfs
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    skopeo \
    umoci \
    uidmap \
    && rm -rf /var/lib/apt/lists/*

COPY scripts/rootfs.sh /tmp/rootfs.sh
RUN chmod +x /tmp/rootfs.sh \
    && JUDGE_IMAGES_DIR=/var/castletown/images /tmp/rootfs.sh

FROM debian:13-slim AS runtime
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    curl \
    fuse-overlayfs \
    iptables \
    tini \
    uidmap \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/castletown

COPY --from=builder /usr/local/bin/castletown /usr/local/bin/castletown
COPY --from=rootfs /var/castletown/images /var/castletown/images
COPY scripts/rootfs.sh /opt/castletown/scripts/rootfs.sh
COPY docker/entrypoint.sh /usr/local/bin/castletown-entrypoint

RUN chmod +x /usr/local/bin/castletown-entrypoint /opt/castletown/scripts/rootfs.sh

ENV CASTLETOWN_SKIP_ROOTFS=1 \
    BLOB_ROOT=/var/castletown/blobs \
    WORK_ROOT=/tmp/castletown/work \
    JUDGE_IMAGES_DIR=/var/castletown/images \
    JUDGE_OVERLAYFS_DIR=/tmp/castletown/overlayfs \
    STORAGE_DIR=/tmp/castletown/storage \
    JUDGE_LIBCONTAINER_DIR=/tmp/castletown/libcontainer \
    JUDGE_ROOTFS_DIR=/tmp/castletown/rootfs \
    JUDGE_DISK_CACHE_DIR=/var/castletown/testcases \
    PROBLEM_CACHE_DIR=/var/castletown/problems

ENTRYPOINT ["/usr/bin/tini","--","castletown-entrypoint"]
CMD ["castletown","start"]
