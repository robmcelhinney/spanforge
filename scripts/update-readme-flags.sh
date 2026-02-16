#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
README_PATH="${ROOT_DIR}/README.md"
BEGIN_MARKER="<!-- BEGIN AUTO-GENERATED FLAGS -->"
END_MARKER="<!-- END AUTO-GENERATED FLAGS -->"

if ! grep -Fq "${BEGIN_MARKER}" "${README_PATH}"; then
  echo "missing begin marker in README: ${BEGIN_MARKER}" >&2
  exit 1
fi

if ! grep -Fq "${END_MARKER}" "${README_PATH}"; then
  echo "missing end marker in README: ${END_MARKER}" >&2
  exit 1
fi

FLAGS_OUTPUT="$(cd "${ROOT_DIR}" && go run ./cmd/spanforge --help | awk '/^Flags:/{emit=1} emit{print}')"

TMP_FILE="$(mktemp)"
trap 'rm -f "${TMP_FILE}"' EXIT

in_generated_block=0
while IFS= read -r line; do
  if [[ "${line}" == "${BEGIN_MARKER}" ]]; then
    printf "%s\n" "${BEGIN_MARKER}" >>"${TMP_FILE}"
    printf '```console\n' >>"${TMP_FILE}"
    printf "%s\n" "${FLAGS_OUTPUT}" >>"${TMP_FILE}"
    printf '```\n' >>"${TMP_FILE}"
    in_generated_block=1
    continue
  fi
  if [[ "${line}" == "${END_MARKER}" ]]; then
    printf "%s\n" "${END_MARKER}" >>"${TMP_FILE}"
    in_generated_block=0
    continue
  fi
  if [[ ${in_generated_block} -eq 1 ]]; then
    continue
  fi
  printf "%s\n" "${line}" >>"${TMP_FILE}"
done <"${README_PATH}"

mv "${TMP_FILE}" "${README_PATH}"
echo "Updated README flags block from spanforge --help"
