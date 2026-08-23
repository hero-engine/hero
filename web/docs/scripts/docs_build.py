#!/usr/bin/env python3
"""Prepare and validate the hosted Hero documentation artifact."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
DOCS_DIR = SCRIPT_DIR.parent
REPO_ROOT = DOCS_DIR.parents[1]
DEFAULT_SOURCE = DOCS_DIR / "src"
DEFAULT_SITE = DOCS_DIR / "site"
PLACEHOLDER_METADATA = {
    "source_revision": "BUILD_TIME_SOURCE_REVISION",
    "current_release": "BUILD_TIME_CURRENT_RELEASE",
    "generated_at": "BUILD_TIME_GENERATED_AT",
}
MARKDOWN_LINK_RE = re.compile(r"(?<!!)\[[^]]+\]\(([^)]+)\)")
HEADING_RE = re.compile(r"^#{1,6}\s+(.+?)\s*$")
LANGUAGE_LOADER_RE = re.compile(
    r"function Ie\(t\)\{.*?\}function Fe\(t\)", re.DOTALL
)
FORBIDDEN_ASSET_REFERENCE_RE = re.compile(
    r"(?:wordcut|tinyseg)\.js|(?:^|[/'\"])(?:min/)?lunr\.[a-z]{2,}(?:\.min)?\.js|"
    r"sourceMappingURL=[^\s*]+\.map",
    re.IGNORECASE,
)


def _git(*args: str) -> str:
    return subprocess.check_output(
        ["git", "-C", str(REPO_ROOT), *args], text=True
    ).strip()


def build_metadata() -> dict[str, str]:
    revision = os.environ.get("HERO_DOCS_REVISION") or _git("rev-parse", "HEAD")
    release = os.environ.get("HERO_DOCS_RELEASE") or _git(
        "describe", "--tags", "--abbrev=0"
    )
    generated_at = os.environ.get("HERO_DOCS_BUILT_AT") or datetime.now(
        timezone.utc
    ).replace(microsecond=0).isoformat().replace("+00:00", "Z")
    return {
        "source_revision": revision,
        "current_release": release,
        "generated_at": generated_at,
    }


def write_metadata(source_dir: Path = DEFAULT_SOURCE) -> dict[str, str]:
    metadata = build_metadata()
    about_dir = source_dir / "about"
    about_dir.mkdir(parents=True, exist_ok=True)
    revision_short = metadata["source_revision"][:12]
    page = f"""# Build information

This page is generated during the documentation build. Compare its source
revision with the revision you expect to confirm that the deployed site is not
serving an older artifact.

| Field | Value |
|---|---|
| Current released version | `{metadata['current_release']}` |
| Documentation source revision | `{metadata['source_revision']}` |
| Artifact generated at | `{metadata['generated_at']}` |

Machine-readable marker: [`/revision.json`](../revision.json).

The release value is derived from the latest reachable Git tag. The revision
is the checked-out commit used by the build (`{revision_short}`). A local
strict build proves source consistency only; deployed parity is established by
reading these values from the production site after deployment.
"""
    (about_dir / "build.md").write_text(page, encoding="utf-8")
    (source_dir / "revision.json").write_text(
        json.dumps(metadata, indent=2) + "\n", encoding="utf-8"
    )
    return metadata


def sanitize_site(site_dir: Path = DEFAULT_SITE) -> list[Path]:
    removed: list[Path] = []
    language_bundles = site_dir / "assets" / "javascripts" / "lunr"
    if language_bundles.exists():
        removed.append(language_bundles)
        shutil.rmtree(language_bundles)
    for source_map in site_dir.rglob("*.map"):
        removed.append(source_map)
        source_map.unlink()
    for asset in list(site_dir.rglob("*.js")) + list(site_dir.rglob("*.css")):
        content = asset.read_text(encoding="utf-8")
        if "wordcut.js" in content or "tinyseg.js" in content:
            content = LANGUAGE_LOADER_RE.sub(
                "function Ie(){return Promise.resolve()}function Fe(t)",
                content,
                count=1,
            )
        content = re.sub(r"\n?//# sourceMappingURL=[^\r\n]*", "", content)
        content = re.sub(r"/\*# sourceMappingURL=[^*]*\*/", "", content)
        asset.write_text(content, encoding="utf-8")
    return removed


def verify_sanitized_site(site_dir: Path = DEFAULT_SITE) -> list[str]:
    errors: list[str] = []
    forbidden_names = ("wordcut", "tinyseg")
    for path in site_dir.rglob("*"):
        if not path.is_file():
            continue
        lowered = path.name.lower()
        if any(name in lowered for name in forbidden_names):
            errors.append(f"forbidden generated bundle remains: {path}")
        if re.fullmatch(r"lunr\.[a-z]{2,}(?:\.min)?\.js", lowered):
            errors.append(f"unused Lunr language bundle remains: {path}")
        if path.suffix.lower() in {".css", ".html", ".js", ".json"}:
            content = path.read_text(encoding="utf-8")
            if match := FORBIDDEN_ASSET_REFERENCE_RE.search(content):
                errors.append(
                    f"reference to removed generated asset remains in {path}: "
                    f"{match.group(0)}"
                )
    return errors


def check_javascript_syntax(site_dir: Path = DEFAULT_SITE) -> list[str]:
    if shutil.which("node") is None:
        return ["Node.js is required to syntax-check generated JavaScript"]

    errors: list[str] = []
    for asset in sorted(site_dir.rglob("*.js")):
        result = subprocess.run(
            ["node", "--check", str(asset)],
            capture_output=True,
            text=True,
            check=False,
        )
        if result.returncode:
            detail = (result.stderr or result.stdout).strip()
            errors.append(f"invalid generated JavaScript in {asset}:\n{detail}")
    return errors


def _anchors(markdown: str) -> set[str]:
    anchors: set[str] = set()
    counts: dict[str, int] = {}
    in_fence = False
    for line in markdown.splitlines():
        if line.lstrip().startswith("```"):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        match = HEADING_RE.match(line)
        if not match:
            continue
        heading = re.sub(r"\s+#+$", "", match.group(1))
        heading = re.sub(r"<[^>]+>", "", heading)
        heading = re.sub(r"[`*_]", "", heading).strip().lower()
        slug = re.sub(r"[^\w\- ]", "", heading, flags=re.UNICODE)
        slug = re.sub(r"[\s-]+", "-", slug).strip("-")
        duplicate = counts.get(slug, 0)
        counts[slug] = duplicate + 1
        anchors.add(slug if duplicate == 0 else f"{slug}_{duplicate}")
    return anchors


def check_links(source_dir: Path = DEFAULT_SOURCE, external: bool = False) -> list[str]:
    errors: list[str] = []
    for page in sorted(source_dir.rglob("*.md")):
        text = page.read_text(encoding="utf-8")
        for target in MARKDOWN_LINK_RE.findall(text):
            target = target.strip().split()[0].strip("<>")
            if target.startswith(("mailto:", "#")):
                if target.startswith("#") and target[1:] not in _anchors(text):
                    errors.append(f"{page}: missing local anchor {target}")
                continue
            if target.startswith(("http://", "https://")):
                if external:
                    error = _check_external(target)
                    if error:
                        errors.append(f"{page}: {error}")
                continue
            path_part, _, fragment = target.partition("#")
            resolved = (page.parent / path_part).resolve() if path_part else page
            if path_part.endswith("/"):
                resolved = resolved / "index.md"
            if resolved.is_dir():
                resolved = resolved / "index.md"
            if not resolved.exists():
                errors.append(f"{page}: missing link target {target}")
                continue
            if fragment and resolved.suffix == ".md":
                destination = resolved.read_text(encoding="utf-8")
                if fragment not in _anchors(destination):
                    errors.append(f"{page}: missing anchor {target}")
    return errors


def _check_external(url: str) -> str | None:
    if "example." in url or "localhost" in url:
        return None
    request = urllib.request.Request(
        url,
        headers={"User-Agent": "hero-docs-link-check/1.0", "Range": "bytes=0-0"},
    )
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            if response.status >= 400 and response.status not in {401, 403, 429}:
                return f"external link returned {response.status}: {url}"
    except urllib.error.HTTPError as exc:
        status = exc.code
        exc.close()
        if status >= 400 and status not in {401, 403, 429}:
            return f"external link returned {status}: {url}"
    except urllib.error.URLError as exc:
        return f"external link failed: {url}: {exc.reason}"
    return None


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)
    subparsers.add_parser("metadata")
    subparsers.add_parser("sanitize")
    subparsers.add_parser("check-js")
    check = subparsers.add_parser("check")
    check.add_argument("--external", action="store_true")
    args = parser.parse_args(argv)

    if args.command == "metadata":
        metadata = write_metadata()
        print(json.dumps(metadata, sort_keys=True))
        return 0
    if args.command == "sanitize":
        sanitize_site()
        errors = verify_sanitized_site()
    elif args.command == "check-js":
        errors = check_javascript_syntax()
    else:
        errors = check_links(external=args.external)
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
