#!/usr/bin/env python3
"""Build and validate the revision-identifiable Hero landing artifact."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import struct
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from html.parser import HTMLParser
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
LANDING_DIR = SCRIPT_DIR.parent
REPO_ROOT = LANDING_DIR.parents[1]
DEFAULT_SOURCE = LANDING_DIR / "site"
DEFAULT_ARTIFACT = LANDING_DIR / "dist"
SOURCE_REVISION = "BUILD_TIME_SOURCE_REVISION"
SOURCE_COMMIT = "BUILD_TIME_SOURCE_COMMIT"
SOURCE_DIGEST = "BUILD_TIME_SOURCE_DIGEST"
SOURCE_DIRTY = "BUILD_TIME_SOURCE_DIRTY"
GENERATED_AT = "BUILD_TIME_GENERATED_AT"
METADATA_PLACEHOLDERS = (
    SOURCE_REVISION,
    SOURCE_COMMIT,
    SOURCE_DIGEST,
    SOURCE_DIRTY,
    GENERATED_AT,
)
CANONICAL_URL = "https://heroengine.ai/"
DOCS_ORIGIN = "https://docs.heroengine.ai"
CANONICAL_LOGO_SHA256 = "05d31357bc2f73c8844d4d34d9568744781826637b86e3e9900d1dd3beb68622"
APPROVED_SOCIAL_CARD_SHA256 = "4b5fc06f86674df7e7a5df96e48a6dca1939a5542a3e2ed14b3c4df61d51794d"
OLD_BOLT_PATH = "M52 8 L22 46 L40 46 L34 82 L68 42 L50 42 Z"
ALLOWED_EXTERNAL_ORIGINS = {
    "https://heroengine.ai",
    DOCS_ORIGIN,
    "https://github.com",
}
FORBIDDEN_CLAIMS = {
    "v0.9": "stale release copy",
    "open source": "license and visibility gates have not landed",
    "open source · mit": "superseded license claim",
    "hero · sidekick brain": "superseded positioning",
    "installs as slash commands": "workflows are harness-native",
    "cloud and team mode": "unsupported public roadmap copy",
    "agent outposts": "unsupported public roadmap copy",
    "diagnosing": "stale fictional status output",
    "approval-aware agent jobs": "no shipped end-to-end approval pause/resume bridge",
    "your agents finish.": "superseded canonical tagline",
}


class LandingParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.tags: list[tuple[str, dict[str, str]]] = []
        self.links: list[str] = []
        self.ids: set[str] = set()
        self.heading_counts = {"h1": 0, "h2": 0, "h3": 0}
        self.stack: list[str] = []
        self.structure_errors: list[str] = []

    def handle_starttag(
        self, tag: str, attrs: list[tuple[str, str | None]]
    ) -> None:
        values = {name: value or "" for name, value in attrs}
        self.tags.append((tag, values))
        if tag in self.heading_counts:
            self.heading_counts[tag] += 1
        if values.get("id"):
            self.ids.add(values["id"])
        if tag in {"a", "link"} and values.get("href"):
            self.links.append(values["href"])
        if tag in {"img", "script"} and values.get("src"):
            self.links.append(values["src"])
        if tag not in {
            "area",
            "base",
            "br",
            "col",
            "embed",
            "hr",
            "img",
            "input",
            "link",
            "meta",
            "param",
            "source",
            "track",
            "wbr",
        }:
            self.stack.append(tag)

    def handle_endtag(self, tag: str) -> None:
        if tag in {
            "area",
            "base",
            "br",
            "col",
            "embed",
            "hr",
            "img",
            "input",
            "link",
            "meta",
            "param",
            "source",
            "track",
            "wbr",
        }:
            return
        if not self.stack:
            self.structure_errors.append(f"unexpected closing tag: </{tag}>")
            return
        opened = self.stack.pop()
        if opened != tag:
            self.structure_errors.append(
                f"mismatched closing tag: expected </{opened}>, found </{tag}>"
            )

    def close(self) -> None:
        super().close()
        if self.stack:
            self.structure_errors.append(
                "unclosed tags: " + ", ".join(f"<{tag}>" for tag in self.stack)
            )


def _git(repo_root: Path, *args: str) -> str:
    return subprocess.check_output(
        ["git", "-C", str(repo_root), *args], text=True
    ).strip()


def source_tree_digest(source_dir: Path) -> str:
    digest = hashlib.sha256()
    files = sorted(
        candidate for candidate in source_dir.rglob("*") if candidate.is_file()
    )
    for path in files:
        relative = path.relative_to(source_dir).as_posix().encode("utf-8")
        content = path.read_bytes()
        digest.update(relative)
        digest.update(b"\0")
        digest.update(len(content).to_bytes(8, "big"))
        digest.update(content)
    return digest.hexdigest()


def _source_is_dirty(source_dir: Path, repo_root: Path) -> bool:
    try:
        relative = source_dir.resolve().relative_to(repo_root.resolve())
    except ValueError as exc:
        raise ValueError("landing source must be inside the Git worktree") from exc
    status = _git(
        repo_root,
        "status",
        "--porcelain=v1",
        "--untracked-files=all",
        "--",
        relative.as_posix(),
    )
    return bool(status)


def build_metadata(
    source_dir: Path = DEFAULT_SOURCE, repo_root: Path = REPO_ROOT
) -> dict[str, str | bool]:
    commit = _git(repo_root, "rev-parse", "HEAD")
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError(f"current Git HEAD is not an exact 40-character SHA: {commit!r}")
    digest = source_tree_digest(source_dir)
    dirty = _source_is_dirty(source_dir, repo_root)
    explicit_revision = os.environ.get("HERO_LANDING_REVISION")
    if explicit_revision:
        if not re.fullmatch(r"[0-9a-f]{40}", explicit_revision):
            raise ValueError("HERO_LANDING_REVISION must be an exact 40-character Git SHA")
        if explicit_revision != commit:
            raise ValueError(
                "HERO_LANDING_REVISION must equal the checked-out Git HEAD exactly"
            )
        if dirty:
            raise ValueError(
                "explicit landing revisions require web/landing/site to be clean, including untracked files"
            )
        revision = commit
    else:
        revision = f"{commit}+worktree:{digest}" if dirty else commit
    generated_at = os.environ.get("HERO_LANDING_BUILT_AT") or datetime.now(
        timezone.utc
    ).replace(microsecond=0).isoformat().replace("+00:00", "Z")
    return {
        "source_revision": revision,
        "source_commit": commit,
        "source_digest": digest,
        "source_dirty": dirty,
        "generated_at": generated_at,
        "canonical_url": CANONICAL_URL,
    }


def build_artifact(
    source_dir: Path = DEFAULT_SOURCE,
    artifact_dir: Path = DEFAULT_ARTIFACT,
    repo_root: Path = REPO_ROOT,
) -> dict[str, str | bool]:
    metadata = build_metadata(source_dir, repo_root)
    if artifact_dir.exists():
        shutil.rmtree(artifact_dir)
    shutil.copytree(source_dir, artifact_dir)

    for path in artifact_dir.rglob("*"):
        if not path.is_file() or path.suffix.lower() not in {".html", ".json"}:
            continue
        text = path.read_text(encoding="utf-8")
        text = text.replace(SOURCE_REVISION, metadata["source_revision"])
        text = text.replace(GENERATED_AT, metadata["generated_at"])
        path.write_text(text, encoding="utf-8")

    (artifact_dir / "revision.json").write_text(
        json.dumps(metadata, indent=2) + "\n", encoding="utf-8"
    )
    return metadata


def _resolve_local_asset(root: Path, href: str) -> Path | None:
    parsed = urllib.parse.urlparse(href)
    if parsed.scheme or parsed.netloc or href.startswith(("#", "mailto:")):
        return None
    path = parsed.path
    if not path:
        return None
    return root / path.lstrip("/")


def _docs_source_for(url: str) -> Path | None:
    parsed = urllib.parse.urlparse(url)
    if f"{parsed.scheme}://{parsed.netloc}" != DOCS_ORIGIN:
        return None
    relative = parsed.path.strip("/")
    if not relative:
        relative = "index"
    candidate = REPO_ROOT / "web" / "docs" / "src" / f"{relative}.md"
    if candidate.exists():
        return candidate
    candidate = REPO_ROOT / "web" / "docs" / "src" / relative / "index.md"
    return candidate if candidate.exists() else None


def _check_external(url: str) -> str | None:
    request = urllib.request.Request(
        url,
        headers={"User-Agent": "hero-landing-link-check/1.0", "Range": "bytes=0-0"},
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


def validate_site(
    root: Path,
    *,
    built: bool,
    external: bool = False,
    source_dir: Path = DEFAULT_SOURCE,
    repo_root: Path = REPO_ROOT,
) -> list[str]:
    errors: list[str] = []
    index = root / "index.html"
    if not index.exists():
        return [f"missing landing document: {index}"]

    html = index.read_text(encoding="utf-8")
    lowered = html.lower()
    parser = LandingParser()
    parser.feed(html)
    parser.close()

    errors.extend(parser.structure_errors)

    if parser.heading_counts["h1"] != 1:
        errors.append(f"expected exactly one h1, found {parser.heading_counts['h1']}")
    for landmark in ("header", "nav", "main", "footer"):
        if not any(tag == landmark for tag, _ in parser.tags):
            errors.append(f"missing semantic landmark: {landmark}")
    if "skip-link" not in html or 'href="#main"' not in html:
        errors.append("missing keyboard skip link to #main")
    if ":focus-visible" not in html:
        errors.append("missing visible keyboard focus treatment")
    if "prefers-reduced-motion" not in html:
        errors.append("missing reduced-motion accommodation")
    if not re.search(r"@media\s*\(max-width:", html):
        errors.append("missing responsive small-screen rules")
    if "overflow-wrap" not in html:
        errors.append("missing overflow protection for long revision or command text")
    if html.count('src="/hero-logo.svg"') < 2:
        errors.append("visible landing brand must use the canonical /hero-logo.svg")

    canonical_logo = root / "hero-logo.svg"
    if not canonical_logo.exists():
        errors.append("missing canonical Hero logo: /hero-logo.svg")
    else:
        logo_bytes = canonical_logo.read_bytes()
        logo = logo_bytes.decode("utf-8")
        digest = hashlib.sha256(logo_bytes).hexdigest()
        if digest != CANONICAL_LOGO_SHA256:
            errors.append("canonical Hero logo geometry or serialization changed")
        if 'viewBox="0 0 90 90"' not in logo or logo.count('fill="#90b9e2"') != 2:
            errors.append("canonical Hero logo must retain its 90x90 geometry and #90b9e2 color")
    for svg in sorted(root.glob("*.svg")):
        if OLD_BOLT_PATH in svg.read_text(encoding="utf-8"):
            errors.append(f"obsolete bolt mark remains in landing asset: {svg.name}")

    required_copy = (
        "Your project remembers.",
        "Your agents deliver.",
        "Project memory that survives the session",
        "Verified delivery, not ceremonial specs",
        "Illustrative",
        "Shipped",
        "Optional",
        "Preview",
        "Planned",
    )
    for phrase in required_copy:
        if phrase not in html:
            errors.append(f"missing required message or label: {phrase}")
    for phrase, reason in FORBIDDEN_CLAIMS.items():
        if phrase in lowered:
            errors.append(f"forbidden landing claim {phrase!r}: {reason}")

    if re.search(r"https://github\.com/hero-engine/hero(?!-releases)(?:[\"/#?])", html):
        errors.append("public source link is not gated while the repository is private")
    if '<link rel="canonical" href="https://heroengine.ai/"' not in html:
        errors.append("missing canonical heroengine.ai metadata")
    expected_social_metadata = (
        '<meta property="og:image" content="https://heroengine.ai/og-image.png"',
        '<meta name="twitter:image" content="https://heroengine.ai/og-image.png"',
    )
    for metadata in expected_social_metadata:
        if metadata not in html:
            errors.append(f"missing approved social-card metadata: {metadata}")

    social_card = root / "og-image.png"
    if not social_card.exists():
        errors.append("missing approved social card: /og-image.png")
    else:
        data = social_card.read_bytes()
        if len(data) < 24 or data[:8] != b"\x89PNG\r\n\x1a\n":
            errors.append("social card is not a valid PNG")
        else:
            width, height = struct.unpack(">II", data[16:24])
            if width < 1200 or height < 630:
                errors.append(
                    f"social card is too small for large previews: {width}x{height}"
                )
            if hashlib.sha256(data).hexdigest() != APPROVED_SOCIAL_CARD_SHA256:
                errors.append("social card does not match the approved canonical-brand asset")

    for href in parser.links:
        if href.startswith("#"):
            if href[1:] not in parser.ids:
                errors.append(f"missing in-page link target: {href}")
            continue
        local = _resolve_local_asset(root, href)
        if local is not None and not local.exists():
            errors.append(f"missing local asset: {href}")
            continue
        parsed = urllib.parse.urlparse(href)
        if parsed.scheme in {"http", "https"}:
            origin = f"{parsed.scheme}://{parsed.netloc}"
            if origin not in ALLOWED_EXTERNAL_ORIGINS:
                errors.append(f"unapproved external link origin: {origin}")
                continue
            if origin == DOCS_ORIGIN and _docs_source_for(href) is None:
                errors.append(f"docs link has no matching tracked source page: {href}")
            if external:
                error = _check_external(href)
                if error:
                    errors.append(error)

    revision_path = root / "revision.json"
    if not revision_path.exists():
        errors.append("missing revision.json")
    else:
        try:
            metadata = json.loads(revision_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError as exc:
            errors.append(f"invalid revision.json: {exc}")
        else:
            expected = {
                "source_revision",
                "source_commit",
                "source_digest",
                "source_dirty",
                "generated_at",
                "canonical_url",
            }
            missing = expected - metadata.keys()
            if missing:
                errors.append(f"revision.json missing fields: {', '.join(sorted(missing))}")
            if metadata.get("canonical_url") != CANONICAL_URL:
                errors.append("revision.json canonical_url is not heroengine.ai")

    if built:
        for path in sorted(root.rglob("*")):
            if not path.is_file() or path.suffix.lower() not in {
                ".html",
                ".json",
                ".txt",
                ".xml",
            }:
                continue
            text = path.read_text(encoding="utf-8")
            if any(placeholder in text for placeholder in METADATA_PLACEHOLDERS):
                errors.append(
                    f"built landing artifact still contains metadata placeholders: {path}"
                )
        if revision_path.exists():
            metadata = json.loads(revision_path.read_text(encoding="utf-8"))
            revision = metadata.get("source_revision", "")
            commit = metadata.get("source_commit", "")
            digest = metadata.get("source_digest", "")
            dirty = metadata.get("source_dirty")
            if not re.fullmatch(r"[0-9a-f]{40}", commit):
                errors.append("built source_commit is not an exact 40-character Git SHA")
            elif commit != _git(repo_root, "rev-parse", "HEAD"):
                errors.append("built source_commit does not match the checked-out Git HEAD")
            if not re.fullmatch(r"[0-9a-f]{64}", digest):
                errors.append("built source_digest is not a SHA-256 digest")
            expected_digest = source_tree_digest(source_dir)
            if digest != expected_digest:
                errors.append("built source_digest does not match the current landing source tree")
            if not isinstance(dirty, bool):
                errors.append("built source_dirty must be a boolean")
            elif dirty:
                expected_revision = f"{commit}+worktree:{digest}"
                if revision != expected_revision:
                    errors.append("dirty artifact revision must include commit plus source-tree digest")
            elif revision != commit:
                errors.append("clean artifact revision must equal source_commit")
            if revision not in html:
                errors.append("visible landing revision does not match revision.json")
    else:
        revision_template = revision_path.read_text(encoding="utf-8") if revision_path.exists() else ""
        template_text = html + revision_template
        if any(placeholder not in template_text for placeholder in METADATA_PLACEHOLDERS):
            errors.append("tracked landing template is missing build-time metadata placeholders")

    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)
    subparsers.add_parser("build")
    source = subparsers.add_parser("check-source")
    source.add_argument("--external", action="store_true")
    artifact = subparsers.add_parser("check-artifact")
    artifact.add_argument("--external", action="store_true")
    args = parser.parse_args(argv)

    if args.command == "build":
        metadata = build_artifact()
        errors = validate_site(DEFAULT_ARTIFACT, built=True)
        if not errors:
            print(json.dumps(metadata, sort_keys=True))
    elif args.command == "check-source":
        errors = validate_site(DEFAULT_SOURCE, built=False, external=args.external)
    else:
        errors = validate_site(DEFAULT_ARTIFACT, built=True, external=args.external)

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
