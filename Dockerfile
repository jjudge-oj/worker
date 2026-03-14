FROM golang:1.25 AS go-builder

WORKDIR /workspace

# Copy shared modules
COPY jjudge-api /workspace/jjudge-api
COPY jjudge-grader /workspace/jjudge-grader

# Copy worker module
WORKDIR /workspace/jjudge-worker
COPY jjudge-worker/go.mod jjudge-worker/go.sum ./
RUN go mod download

COPY jjudge-worker/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker .

# Build lime binary
FROM gcc:14 AS lime-builder

WORKDIR /lime
COPY lime/ ./
RUN mkdir -p build && make

# Runtime image
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    uidmap \
    python3 \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Create rootfs from the running system
RUN mkdir -p /rootfs/usr/bin /rootfs/usr/lib /rootfs/usr/local/bin \
    /rootfs/bin /rootfs/lib /rootfs/lib64 /rootfs/dev \
    /rootfs/proc /rootfs/sys /rootfs/tmp /rootfs/work \
    /rootfs/etc \
    && cp -a /usr/bin/python3 /rootfs/usr/bin/ \
    && cp -a /usr/bin/g++ /rootfs/usr/bin/ \
    && cp -a /usr/bin/gcc /rootfs/usr/bin/ 2>/dev/null || true \
    && cp -a /usr/lib/x86_64-linux-gnu /rootfs/usr/lib/ 2>/dev/null || true \
    && cp -a /lib/x86_64-linux-gnu /rootfs/lib/ 2>/dev/null || true \
    && cp -a /lib64/ld-linux-x86-64.so.2 /rootfs/lib64/ 2>/dev/null || true \
    && echo "root:x:0:0:root:/root:/bin/sh" > /rootfs/etc/passwd \
    && echo "root:x:0:" > /rootfs/etc/group \
    && chmod 1777 /rootfs/tmp

WORKDIR /app
COPY --from=go-builder /out/worker /usr/local/bin/worker
COPY --from=lime-builder /lime/build/lime /usr/local/bin/lime
COPY jjudge-worker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create work directories
RUN mkdir -p /tmp/judge/submissions /tmp/judge/work /tmp/judge/overlayfs /tmp/judge/rootfs

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
