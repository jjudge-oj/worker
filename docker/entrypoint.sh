#!/usr/bin/env bash
set -euo pipefail

log() {
  echo "[castletown-entrypoint] $*"
}

: "${BLOB_ROOT:=/var/castletown/blobs}"
: "${WORK_ROOT:=/tmp/castletown/work}"
# Prefer config env names (JUDGE_*) but fall back to legacy variables if present.
: "${JUDGE_IMAGES_DIR:=${IMAGES_DIR:-/var/castletown/images}}"
: "${JUDGE_OVERLAYFS_DIR:=${OVERLAYFS_DIR:-/tmp/castletown/overlayfs}}"
: "${STORAGE_DIR:=/tmp/castletown/storage}"
: "${JUDGE_LIBCONTAINER_DIR:=${LIBCONTAINER_DIR:-/tmp/castletown/libcontainer}}"
: "${JUDGE_ROOTFS_DIR:=${ROOTFS_DIR:-/tmp/castletown/rootfs}}"
: "${JUDGE_DISK_CACHE_DIR:=${DISK_CACHE_DIR:-/var/castletown/testcases}}"
: "${PROBLEM_CACHE_DIR:=/var/castletown/problems}"
: "${CASTLETOWN_SKIP_ROOTFS:=1}"

mkdir -p \
  "${BLOB_ROOT}" \
  "${WORK_ROOT}" \
  "${JUDGE_IMAGES_DIR}" \
  "${JUDGE_OVERLAYFS_DIR}" \
  "${STORAGE_DIR}" \
  "${JUDGE_LIBCONTAINER_DIR}" \
  "${JUDGE_ROOTFS_DIR}" \
  "${JUDGE_DISK_CACHE_DIR}" \
  "${PROBLEM_CACHE_DIR}"

if [[ "${CASTLETOWN_SKIP_ROOTFS}" != "1" ]]; then
  if ! command -v skopeo >/dev/null 2>&1 || ! command -v umoci >/dev/null 2>&1; then
    log "rootfs bootstrap requested but skopeo/umoci are not installed in this image"
    exit 1
  fi
  if [[ ! -d "${JUDGE_IMAGES_DIR}/gcc-15-bookworm" ]]; then
    log "bootstraping gcc-15-bookworm rootfs into ${JUDGE_IMAGES_DIR}"
    JUDGE_IMAGES_DIR="${JUDGE_IMAGES_DIR}" /opt/castletown/scripts/rootfs.sh
  else
    log "rootfs already present in ${JUDGE_IMAGES_DIR}, skipping bootstrap"
  fi
else
  log "CASTLETOWN_SKIP_ROOTFS=1, skipping rootfs bootstrap"
fi

# Ensure a writable /work mountpoint exists inside each rootfs image so OCI runtimes
# don't attempt to create it on a read-only lowerdir.
if [[ -d "${JUDGE_IMAGES_DIR}" ]]; then
  while IFS= read -r -d '' img; do
    if [[ -d "${img}" ]]; then
      mkdir -p "${img}/work" || log "warning: could not create ${img}/work"
    fi
  done < <(find "${JUDGE_IMAGES_DIR}" -mindepth 1 -maxdepth 1 -type d -print0)
fi

log "starting castletown: $*"
exec "$@"
