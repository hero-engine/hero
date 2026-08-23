#!/bin/sh

set -eu

repo_root=$(git rev-parse --show-toplevel)
scanner="$repo_root/scripts/public-readiness-scan.sh"
scratch=$(mktemp -d "${TMPDIR:-/tmp}/hero-public-readiness-test.XXXXXX")
trap 'rm -rf "$scratch"' EXIT HUP INT TERM

git -C "$scratch" init -q
git -C "$scratch" config user.name "Readiness Test"
git -C "$scratch" config user.email "readiness@example.invalid"

mkdir -p "$scratch/cloud/api" "$scratch/.hero/sessions/session-a"
printf '%s\n' 'package api' > "$scratch/cloud/api/private.go"
printf '%s\n' 'sqlite fixture' > "$scratch/.hero/sessions/session-a/refs.db"
token_prefix=github_pat_
token_suffix=abcdefghijklmnopqrstuvwxyz123456
printf '%s\n' "TOKEN=${token_prefix}${token_suffix}" > "$scratch/config.txt"
git -C "$scratch" add .
git -C "$scratch" commit -qm "add unsafe fixtures"
rm -rf "$scratch/cloud" "$scratch/.hero/sessions"
git -C "$scratch" add -u
git -C "$scratch" commit -qm "remove unsafe fixtures"

set +e
"$scanner" --all "$scratch" > "$scratch/report.tsv"
status=$?
set -e
[ "$status" -eq 2 ]

grep -Fq 'cloud/api/private.go' "$scratch/report.tsv"
grep -Fq '.hero/sessions/session-a/refs.db' "$scratch/report.tsv"
grep -Fq 'provider-token' "$scratch/report.tsv"
grep -Fq '<redacted:provider-token>' "$scratch/report.tsv"
if grep -Fq "${token_prefix}${token_suffix}" "$scratch/report.tsv"; then
  echo "scanner leaked matched secret material" >&2
  exit 1
fi

mkdir -p "$scratch/scripts"
token_fingerprint=$(awk -F '\t' '$6 == "provider-token" {print $7; exit}' "$scratch/report.tsv")
printf '%s\t%s\t%s\t%s\n' \
  'config.txt' 'provider-token' "$token_fingerprint" 'synthetic scanner fixture' \
  > "$scratch/scripts/public-readiness-baseline.tsv"
set +e
"$scanner" --current "$scratch" > "$scratch/baselined-report.tsv"
status=$?
set -e
[ "$status" -eq 0 ]
grep -Fq 'reviewed-baseline' "$scratch/baselined-report.tsv"

printf '%s\n' "TOKEN=${token_prefix}${token_suffix}changed" > "$scratch/config.txt"
set +e
"$scanner" --current "$scratch" > "$scratch/mutated-report.tsv"
status=$?
set -e
[ "$status" -eq 2 ]
grep -Fq 'provider-token' "$scratch/mutated-report.tsv"

clean="$scratch-clean"
mkdir -p "$clean"
git -C "$clean" init -q
git -C "$clean" config user.name "Readiness Test"
git -C "$clean" config user.email "readiness@example.invalid"
printf '%s\n' 'safe public content' > "$clean/README.md"
git -C "$clean" add README.md
git -C "$clean" commit -qm "safe fixture"
"$scanner" --all "$clean" > "$clean/report.tsv"

echo "public-readiness scanner tests passed"
