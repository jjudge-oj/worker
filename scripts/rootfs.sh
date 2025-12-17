#!/usr/bin/env bash

set -euo pipefail

IMAGE_REF=${CASTLETOWN_IMAGE_REF:-"gcc:15-bookworm"}
IMAGE_NAME=${CASTLETOWN_IMAGE_NAME:-"gcc-15-bookworm"}
# Prefer the env var name used by internal/config (JUDGE_IMAGES_DIR), but
# fall back to the legacy IMAGES_DIR if set.
IMAGES_DIR=${JUDGE_IMAGES_DIR:-${IMAGES_DIR:-"/var/castletown/images"}}
TMP_DIR=${CASTLETOWN_TMP_DIR:-"/tmp"}

mkdir -p "${IMAGES_DIR}"
mkdir -p "${TMP_DIR}"

if [[ -d "${IMAGES_DIR}/${IMAGE_NAME}" ]]; then
    echo "[rootfs] ${IMAGES_DIR}/${IMAGE_NAME} already exists, nothing to do"
    exit 0
fi

if ! command -v skopeo >/dev/null 2>&1; then
    echo "[rootfs] skopeo is required but not installed" >&2
    exit 1
fi

if ! command -v umoci >/dev/null 2>&1; then
    echo "[rootfs] umoci is required but not installed" >&2
    exit 1
fi

oci_dir=$(mktemp -d -p "${TMP_DIR}" castletown-oci-XXXXXX)
rootfs_dir=$(mktemp -d -p "${TMP_DIR}" castletown-rootfs-XXXXXX)
trap 'rm -rf "${oci_dir}" "${rootfs_dir}"' EXIT

echo "[rootfs] downloading ${IMAGE_REF}"
skopeo copy "docker://${IMAGE_REF}" "oci:${oci_dir}:${IMAGE_NAME}"

echo "[rootfs] unpacking image"
umoci raw unpack --rootless \
    --image "${oci_dir}:${IMAGE_NAME}" \
    "${rootfs_dir}"

source_dir="${rootfs_dir}/rootfs"
if [[ ! -d "${source_dir}" ]]; then
    # umoci versions prior to 0.5 place the rootfs directly at the destination
    source_dir="${rootfs_dir}"
fi

if [[ ! -d "${source_dir}" ]]; then
    echo "[rootfs] unpack failed: ${source_dir} not found" >&2
    exit 1
fi

mkdir -p "${IMAGES_DIR}/${IMAGE_NAME}"
cp -r "${source_dir}/." "${IMAGES_DIR}/${IMAGE_NAME}/"
mkdir -p "${IMAGES_DIR}/${IMAGE_NAME}/box"

echo "[rootfs] prepared ${IMAGES_DIR}/${IMAGE_NAME}"
