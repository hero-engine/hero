#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 [--current|--history|--all] [repository]" >&2
}

scope=all
case "${1:-}" in
  --current) scope=current; shift ;;
  --history) scope=history; shift ;;
  --all) shift ;;
  -*) usage; exit 64 ;;
esac

repo=${1:-.}
git -C "$repo" rev-parse --is-inside-work-tree >/dev/null
repo=$(cd "$repo" && pwd)
baseline="$repo/scripts/public-readiness-baseline.tsv"
scratch=$(mktemp -d "${TMPDIR:-/tmp}/hero-public-readiness.XXXXXX")
trap 'rm -rf "$scratch"' EXIT HUP INT TERM
findings="$scratch/findings.tsv"
: > "$findings"

fingerprint() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 | awk '{print substr($1, 1, 16)}'
  else
    sha256sum | awk '{print substr($1, 1, 16)}'
  fi
}

emit() {
  severity=$1
  scan_scope=$2
  object_ref=$3
  file_name=$4
  line_number=$5
  finding_type=$6
  finding_fingerprint=$7
  evidence="<redacted:$finding_type>"
  if [ -f "$baseline" ] && awk -F '\t' -v file="$file_name" \
    -v type="$finding_type" -v fp="$finding_fingerprint" \
    '$1 == file && $2 == type && $3 == fp {found=1} END {exit found ? 0 : 1}' \
    "$baseline"; then
    severity=reviewed
    evidence="<redacted:$finding_type:reviewed-baseline>"
  fi
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$severity" "$scan_scope" "$object_ref" "$file_name" "$line_number" \
    "$finding_type" "$finding_fingerprint" "$evidence" >> "$findings"
}

scan_text() {
  scan_scope=$1
  object_ref=$2
  file_name=$3
  content_file=$4
  matches="$scratch/matches"
  LC_ALL=C awk '
    BEGIN { OFS="\t" }
    /-----BEGIN (RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----/ {
      print NR, "blocker", "private-key", $0
    }
    /(github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{16}|xox[baprs]-[A-Za-z0-9-]{10,}|[rs]k_live_[A-Za-z0-9]{16,})/ {
      print NR, "blocker", "provider-token", $0
    }
    /https?:\/\/[^[:space:]\/:@]+:[^[:space:]@]+@/ {
      print NR, "blocker", "embedded-credential-url", $0
    }
    /(api[_-]?key|client[_-]?secret|access[_-]?token|auth[_-]?token|password)[[:space:]]*[:=][[:space:]]*["\047]?[A-Za-z0-9+\/=_-]{16,}/ {
      print NR, "review", "credential-assignment", $0
    }
    /(\/Users\/[^\/[:space:]]+|\/home\/[^\/[:space:]]+)/ {
      print NR, "review", "absolute-user-path", $0
    }
    /(https?:\/\/[^[:space:]]+\.(internal|local)([:\/]|$)|https?:\/\/(10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.))/ {
      print NR, "review", "internal-endpoint", $0
    }
  ' "$content_file" > "$matches"
  while IFS=$'\t' read -r line_number severity finding_type matched_line; do
    [ -n "${line_number:-}" ] || continue
    finding_fingerprint=$(printf '%s' "$matched_line" | fingerprint)
    emit "$severity" "$scan_scope" "$object_ref" "$file_name" \
      "$line_number" "$finding_type" "$finding_fingerprint"
  done < "$matches"
}

scan_current() {
  git -C "$repo" ls-files -z > "$scratch/current-files"
  while IFS= read -r -d '' file_name; do
    content_file="$repo/$file_name"
    [ -f "$content_file" ] || continue
    case "$file_name" in
      cloud/*|cmd/hero-cloud/*)
        finding_fingerprint=$(printf '%s' "$file_name" | fingerprint)
        emit blocker current HEAD "$file_name" 0 proprietary-path "$finding_fingerprint"
        ;;
      .hero/sessions/*|.hero/hero.local.json|.env|.env.*|.envrc)
        finding_fingerprint=$(printf '%s' "$file_name" | fingerprint)
        emit blocker current HEAD "$file_name" 0 machine-local-path "$finding_fingerprint"
        ;;
      dist/*|bin/*|.build/*|.hero/cache/*|.hero/graph.db|.hero/index.db)
        finding_fingerprint=$(printf '%s' "$file_name" | fingerprint)
        emit blocker current HEAD "$file_name" 0 generated-artifact "$finding_fingerprint"
        ;;
    esac
    if [ "$(wc -c < "$content_file" | tr -d ' ')" -gt 5242880 ]; then
      finding_fingerprint=$(git -C "$repo" hash-object "$file_name" | fingerprint)
      emit review current HEAD "$file_name" 0 large-tracked-file "$finding_fingerprint"
    fi
    LC_ALL=C grep -Iq . "$content_file" 2>/dev/null || continue
    scan_text current HEAD "$file_name" "$content_file"
  done < "$scratch/current-files"
}

scan_history() {
  git -C "$repo" rev-list --objects --all \
    | git -C "$repo" cat-file --batch-check='%(objectname) %(objecttype) %(objectsize) %(rest)' \
    > "$scratch/history-objects"
  while IFS=' ' read -r object_id object_type object_size file_name; do
    [ "$object_type" = blob ] || continue
    [ -n "${file_name:-}" ] || continue
    case "$file_name" in
      cloud/*|cmd/hero-cloud/*)
        finding_fingerprint=$(printf '%s' "$object_id" | fingerprint)
        emit blocker history "object:$object_id" "$file_name" 0 proprietary-path "$finding_fingerprint"
        ;;
      .hero/sessions/*|.hero/hero.local.json|.env|.env.*|.envrc)
        finding_fingerprint=$(printf '%s' "$object_id" | fingerprint)
        emit blocker history "object:$object_id" "$file_name" 0 machine-local-path "$finding_fingerprint"
        ;;
      dist/*|bin/*|.build/*|.hero/cache/*|.hero/graph.db|.hero/index.db)
        finding_fingerprint=$(printf '%s' "$object_id" | fingerprint)
        emit blocker history "object:$object_id" "$file_name" 0 generated-artifact "$finding_fingerprint"
        ;;
    esac
    if [ "$object_size" -gt 5242880 ]; then
      finding_fingerprint=$(printf '%s' "$object_id" | fingerprint)
      emit review history "object:$object_id" "$file_name" 0 large-history-blob "$finding_fingerprint"
      continue
    fi
    content_file="$scratch/blob"
    git -C "$repo" cat-file blob "$object_id" > "$content_file"
    LC_ALL=C grep -Iq . "$content_file" 2>/dev/null || continue
    scan_text history "object:$object_id" "$file_name" "$content_file"
  done < "$scratch/history-objects"
}

case "$scope" in
  current) scan_current ;;
  history) scan_history ;;
  all) scan_current; scan_history ;;
esac

printf 'severity\tscope\tref\tfile\tline\ttype\tfingerprint\tevidence\n'
LC_ALL=C sort -u "$findings"

if awk -F '\t' '$1 == "blocker" {found=1} END {exit found ? 0 : 1}' "$findings"; then
  exit 2
fi
