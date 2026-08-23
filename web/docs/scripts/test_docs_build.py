"""Tests for hosted documentation build preparation."""

from __future__ import annotations

import json
import os
import re
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest import mock

import docs_build


class MetadataTests(unittest.TestCase):
    def test_metadata_is_written_to_human_and_machine_readable_surfaces(self):
        env = {
            "HERO_DOCS_REVISION": "abc123def456",
            "HERO_DOCS_RELEASE": "v9.8.7",
            "HERO_DOCS_BUILT_AT": "2026-08-23T12:00:00Z",
        }
        with tempfile.TemporaryDirectory() as tmpdir, mock.patch.dict(
            os.environ, env, clear=False
        ):
            source = Path(tmpdir)
            metadata = docs_build.write_metadata(source)
            marker = json.loads((source / "revision.json").read_text())
            page = (source / "about" / "build.md").read_text()

        self.assertEqual(metadata, marker)
        self.assertIn("v9.8.7", page)
        self.assertIn("abc123def456", page)
        self.assertIn("2026-08-23T12:00:00Z", page)
        for placeholder in docs_build.PLACEHOLDER_METADATA.values():
            self.assertNotIn(placeholder, page)

    def test_committed_metadata_surfaces_are_build_time_placeholders(self):
        source = docs_build.DEFAULT_SOURCE
        marker = json.loads((source / "revision.json").read_text())
        page = (source / "about" / "build.md").read_text()

        self.assertEqual(marker, docs_build.PLACEHOLDER_METADATA)
        for placeholder in docs_build.PLACEHOLDER_METADATA.values():
            self.assertIn(placeholder, page)


class WorkflowTests(unittest.TestCase):
    def test_placeholder_tests_run_before_metadata_and_build(self):
        workflow = (
            docs_build.REPO_ROOT / ".github" / "workflows" / "docs.yml"
        ).read_text(encoding="utf-8")

        tests = workflow.index(
            "python -m unittest discover -s scripts -p 'test_*.py'"
        )
        metadata = workflow.index("python scripts/docs_build.py metadata")
        build = workflow.index("mkdocs build --strict")

        self.assertLess(tests, metadata)
        self.assertLess(metadata, build)


class SanitizerTests(unittest.TestCase):
    def test_removes_unused_language_bundles_and_source_maps(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            site = Path(tmpdir)
            lunr = site / "assets" / "javascripts" / "lunr"
            lunr.mkdir(parents=True)
            (lunr / "wordcut.js").write_text("LGPL", encoding="utf-8")
            worker = site / "assets" / "worker.js"
            worker.write_text(
                'MIT;function Ie(t){return load("../lunr/wordcut.js", '
                '"../lunr/tinyseg.js")}function Fe(t){return t}'
                "\n//# sourceMappingURL=worker.js.map",
                encoding="utf-8",
            )
            source_map = site / "assets" / "worker.js.map"
            source_map.write_text("{}", encoding="utf-8")

            docs_build.sanitize_site(site)

            self.assertFalse(lunr.exists())
            self.assertFalse(source_map.exists())
            self.assertTrue(worker.exists())
            sanitized_worker = worker.read_text(encoding="utf-8")
            self.assertNotIn("wordcut.js", sanitized_worker)
            self.assertNotIn("tinyseg.js", sanitized_worker)
            self.assertNotIn("sourceMappingURL", sanitized_worker)
            self.assertEqual(docs_build.verify_sanitized_site(site), [])

    def test_verifier_rejects_references_to_removed_assets(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            site = Path(tmpdir)
            worker = site / "worker.js"
            worker.write_text(
                'importScripts("../lunr/wordcut.js")\n'
                "//# sourceMappingURL=worker.js.map\n",
                encoding="utf-8",
            )

            errors = docs_build.verify_sanitized_site(site)

        self.assertEqual(len(errors), 1)
        self.assertIn("reference to removed generated asset", errors[0])

    def test_sanitized_javascript_passes_node_syntax_check(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            site = Path(tmpdir)
            worker = site / "worker.js"
            worker.write_text(
                'function Ie(t){return load("../lunr/wordcut.js")}function Fe(t){return t}',
                encoding="utf-8",
            )

            docs_build.sanitize_site(site)
            errors = docs_build.check_javascript_syntax(site)

        self.assertEqual(errors, [])

    def test_javascript_syntax_checker_rejects_invalid_artifact(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            site = Path(tmpdir)
            (site / "broken.js").write_text("function () {", encoding="utf-8")

            errors = docs_build.check_javascript_syntax(site)

        self.assertEqual(len(errors), 1)
        self.assertIn("invalid generated JavaScript", errors[0])


class LinkTests(unittest.TestCase):
    def test_checks_internal_files_and_heading_anchors(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            source = Path(tmpdir)
            (source / "index.md").write_text(
                "# Home\n\n[Details](guide.md#safe-actions)\n", encoding="utf-8"
            )
            (source / "guide.md").write_text(
                "# Guide\n\n## Safe actions\n", encoding="utf-8"
            )
            self.assertEqual(docs_build.check_links(source), [])

            (source / "index.md").write_text(
                "# Home\n\n[Missing](guide.md#missing)\n", encoding="utf-8"
            )
            errors = docs_build.check_links(source)

        self.assertEqual(len(errors), 1)
        self.assertIn("missing anchor", errors[0])

    def test_definite_broken_external_statuses_fail(self):
        for status in (404, 410, 500):
            with self.subTest(status=status), mock.patch(
                "urllib.request.urlopen",
                side_effect=urllib.error.HTTPError(
                    "https://docs.example.invalid/missing",
                    status,
                    "failed",
                    None,
                    None,
                ),
            ):
                error = docs_build._check_external(
                    "https://host.invalid/missing"
                )
                self.assertIn(f"returned {status}", error)

    def test_auth_and_bot_block_statuses_are_treated_as_reachable(self):
        for status in (401, 403, 429):
            with self.subTest(status=status), mock.patch(
                "urllib.request.urlopen",
                side_effect=urllib.error.HTTPError(
                    "https://host.invalid/protected",
                    status,
                    "protected",
                    None,
                    None,
                ),
            ):
                self.assertIsNone(
                    docs_build._check_external("https://host.invalid/protected")
                )


class PublicConfigExampleTests(unittest.TestCase):
    def test_documented_full_example_matches_decoder_fixture(self):
        docs_page = (
            docs_build.REPO_ROOT / "web" / "docs" / "src" / "configuration" / "hero-json.md"
        ).read_text(encoding="utf-8")
        match = re.search(
            r"## Decoder-backed full example.*?```json\n(.*?)\n```",
            docs_page,
            flags=re.DOTALL,
        )
        self.assertIsNotNone(match)
        documented = json.loads(match.group(1))
        fixture = json.loads(
            (
                docs_build.REPO_ROOT
                / "internal"
                / "config"
                / "testdata"
                / "public-hero.json"
            ).read_text(encoding="utf-8")
        )
        self.assertEqual(documented, fixture)


if __name__ == "__main__":
    unittest.main()
