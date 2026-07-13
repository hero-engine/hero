#!/usr/bin/env python3
"""Generate web/docs/src/releases/index.md from hero-engine/hero-releases.

Fetches published GitHub Releases from ``hero-engine/hero-releases`` (the
repo GoReleaser actually publishes to — see ``.goreleaser.yaml``) and
renders them into a single mkdocs page, newest release first. Safe to
re-run on every docs build: it always rewrites ``releases/index.md`` from
the current set of published releases, so repeated runs against the same
release set produce identical output.

Each release body is expected to look like one of two shapes, depending on
whether it was published before or after ``.goreleaser.yaml``'s
``changelog.groups`` config landed:

    Grouped (current):
        ### Major Features
        - feat(x): ...

        ### Fixes
        - fix(y): ...

    Ungrouped (historical, pre-Change-1):
        - feat(x): ...
        - fix(y): ...

The ungrouped shape is rendered under a single fallback "Changes" section
instead of erroring.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass

GITHUB_API_URL = "https://api.github.com/repos/{repo}/releases"
GITHUB_API_PER_PAGE = 100
DEFAULT_REPO = "hero-engine/hero-releases"
DEFAULT_OUTPUT = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "..", "src", "releases", "index.md"
)

FALLBACK_SECTION_TITLE = "Changes"

# Only these two group titles (as written by .goreleaser.yaml's
# changelog.groups) are passed through as their own section. Anything
# else — no heading at all, or GoReleaser's un-grouped default
# "### Changelog" heading used before Change 1 landed — collapses into
# the single FALLBACK_SECTION_TITLE bucket.
_KNOWN_GROUP_TITLES = {"major features": "Major Features", "fixes": "Fixes"}

_HEADING_RE = re.compile(r"^#{2,4}\s+(.+?)\s*$")
_BULLET_RE = re.compile(r"^[-*]\s+(.*\S)\s*$")

# Changelog bullets are verbatim commit subjects, not prescriptive CLI docs.
# Many mention a `hero <command>` sequence in prose ("hero verify becomes the
# load-bearing checkpoint", "hero peer ... --peer fallback"), and some name
# commands that have since been renamed. The markdown drift gate
# (internal/cli/markdown_drift_test.go's
# TestMarkdownInvocationsResolveAgainstRootCmd) scans this generated page and
# tries to resolve every such sequence against the live CLI tree, so those
# historical, unfixable subjects would fail it. Bullets that mention a hero
# command therefore carry the gate's documented per-line ignore marker; the
# HTML comment is invisible in rendered output.
_HERO_INVOCATION_RE = re.compile(r"\bhero\s+[a-z]")
DRIFT_IGNORE_MARKER = "<!-- drift-test:ignore -->"

PAGE_HEADER = """# Releases

Every release published to `hero-engine/hero-releases`, generated
automatically at docs build time. Pick a version in the left nav to jump
to its notes.

"""


@dataclass
class Release:
    tag_name: str
    published_at: str  # ISO 8601, e.g. "2026-07-09T14:32:10Z"
    body: str


def parse_release_body(body: str) -> "dict[str, list[str]]":
    """Group a release body's bullets by their nearest heading.

    Returns an ordered mapping of section title -> bullet lines. Only
    "Major Features" and "Fixes" headings (as written by
    ``changelog.groups``) are recognized as their own section — any other
    heading (including GoReleaser's default "### Changelog" wrapper used
    before Change 1 landed) or no heading at all collapses into a single
    ``"Changes"`` fallback section rather than erroring. Headings that
    capture no bullets are dropped.
    """
    sections: "dict[str, list[str]]" = {}
    current = FALLBACK_SECTION_TITLE

    for raw_line in (body or "").splitlines():
        heading_match = _HEADING_RE.match(raw_line)
        if heading_match:
            title = heading_match.group(1).strip()
            current = _KNOWN_GROUP_TITLES.get(title.lower(), FALLBACK_SECTION_TITLE)
            sections.setdefault(current, [])
            continue

        bullet_match = _BULLET_RE.match(raw_line)
        if bullet_match:
            sections.setdefault(current, []).append(bullet_match.group(1))

    return {title: bullets for title, bullets in sections.items() if bullets}


def format_release_date(published_at: str) -> str:
    """Format an ISO 8601 ``published_at`` timestamp as ``YYYY-MM-DD``."""
    return published_at.split("T", 1)[0]


def format_bullet(bullet: str) -> str:
    """Render one changelog bullet, appending the markdown drift gate's ignore
    marker when the commit subject mentions a ``hero <command>`` sequence.

    See ``_HERO_INVOCATION_RE`` for why these verbatim subjects must be exempt
    from CLI-invocation resolution.
    """
    if _HERO_INVOCATION_RE.search(bullet):
        return f"- {bullet} {DRIFT_IGNORE_MARKER}"
    return f"- {bullet}"


def format_release_section(
    tag_name: str, published_at: str, sections: "dict[str, list[str]]"
) -> str:
    """Render one release as a ``## {tag} — {date}`` section with grouped sub-sections."""
    date = format_release_date(published_at)
    lines = [f"## {tag_name} — {date}", ""]

    for title, bullets in sections.items():
        lines.append(f"### {title}")
        lines.append("")
        for bullet in bullets:
            lines.append(format_bullet(bullet))
        lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def sort_releases_newest_first(releases: "list[Release]") -> "list[Release]":
    return sorted(releases, key=lambda r: r.published_at, reverse=True)


def generate_page(releases: "list[Release]") -> str:
    """Render the full releases/index.md page, newest release first."""
    ordered = sort_releases_newest_first(releases)
    sections = [
        format_release_section(
            release.tag_name, release.published_at, parse_release_body(release.body)
        )
        for release in ordered
    ]
    return PAGE_HEADER + "\n".join(sections)


def fetch_releases(repo: str = DEFAULT_REPO, token: "str | None" = None) -> "list[Release]":
    """Fetch all releases from the GitHub REST API, paging through results.

    Uses ``token`` for rate-limit headroom when provided, and falls back to
    an unauthenticated request otherwise — these are public releases.

    GitHub's Releases API defaults to 30 items per page and caps at 100.
    Requesting ``per_page=100`` alone isn't sufficient once a repo passes
    100 releases, so this loops with ``page=N`` until a page comes back
    empty (or short of a full page), rather than assuming any fixed
    release count.
    """
    base_url = GITHUB_API_URL.format(repo=repo)
    headers = {
        "Accept": "application/vnd.github+json",
        "User-Agent": "hero-release-notes-generator",
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"

    payload: "list[dict]" = []
    page = 1
    while True:
        url = f"{base_url}?per_page={GITHUB_API_PER_PAGE}&page={page}"
        request = urllib.request.Request(url, headers=headers)
        with urllib.request.urlopen(request, timeout=30) as response:
            page_payload = json.loads(response.read().decode("utf-8"))

        payload.extend(page_payload)
        if len(page_payload) < GITHUB_API_PER_PAGE:
            break
        page += 1

    return _releases_from_payload(payload)


def load_releases_from_file(path: str) -> "list[Release]":
    """Load releases from a local JSON fixture shaped like the GitHub Releases API response."""
    with open(path, encoding="utf-8") as f:
        payload = json.load(f)
    return _releases_from_payload(payload)


def _releases_from_payload(payload: "list[dict]") -> "list[Release]":
    return [
        Release(
            tag_name=item["tag_name"],
            published_at=item["published_at"],
            body=item.get("body") or "",
        )
        for item in payload
    ]


def write_page(content: str, output_path: str) -> None:
    os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(content)


def main(argv: "list[str] | None" = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--repo", default=DEFAULT_REPO, help="owner/repo to fetch GitHub Releases from"
    )
    parser.add_argument(
        "--output", default=DEFAULT_OUTPUT, help="path to write the generated releases page"
    )
    parser.add_argument(
        "--input-file",
        help="path to a JSON fixture of a GitHub Releases API response, used instead of a "
        "live network call (for local/CI validation without network/auth)",
    )
    args = parser.parse_args(argv)

    if args.input_file:
        releases = load_releases_from_file(args.input_file)
    else:
        token = os.environ.get("GITHUB_TOKEN")
        try:
            releases = fetch_releases(repo=args.repo, token=token)
        except urllib.error.URLError as exc:
            print(f"error: failed to fetch releases from {args.repo}: {exc}", file=sys.stderr)
            return 1

    if not releases:
        print(f"error: no releases found for {args.repo}", file=sys.stderr)
        return 1

    write_page(generate_page(releases), args.output)
    print(f"wrote {len(releases)} release(s) to {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
