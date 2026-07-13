"""Tests for generate_release_notes.py.

Stdlib unittest only — no pytest dependency (see requirements-docs.txt,
which only ships mkdocs + mkdocs-material). No real network calls are made;
fetch_releases is exercised with a mocked urllib.request.urlopen.
"""

from __future__ import annotations

import json
import os
import tempfile
import unittest
from unittest import mock

import generate_release_notes as gen


GROUPED_BODY = (
    "### Major Features\n"
    "- feat(next): add hero next install-hooks\n"
    "- feat(peer): ship hero peer surface\n"
    "\n"
    "### Fixes\n"
    "- fix(next): make NEXT.md projection deterministic\n"
)

UNGROUPED_BODY = (
    "- feat(scan): add deep-enrichment pass\n"
    "- fix(import): fix pagination cutoff\n"
)


class ParseReleaseBodyTests(unittest.TestCase):
    def test_grouped_parsing(self):
        sections = gen.parse_release_body(GROUPED_BODY)
        self.assertEqual(
            sections,
            {
                "Major Features": [
                    "feat(next): add hero next install-hooks",
                    "feat(peer): ship hero peer surface",
                ],
                "Fixes": ["fix(next): make NEXT.md projection deterministic"],
            },
        )

    def test_ungrouped_falls_back_to_single_section(self):
        sections = gen.parse_release_body(UNGROUPED_BODY)
        self.assertEqual(
            sections,
            {
                "Changes": [
                    "feat(scan): add deep-enrichment pass",
                    "fix(import): fix pagination cutoff",
                ]
            },
        )

    def test_empty_body_produces_no_sections(self):
        self.assertEqual(gen.parse_release_body(""), {})

    def test_default_goreleaser_changelog_heading_falls_back_to_changes(self):
        # Real hero-releases data today (before Change 1 lands on that repo)
        # comes back with GoReleaser's default "### Changelog" wrapper, not
        # no heading at all. That must still collapse into the fallback
        # bucket, not be treated as its own section.
        body = "### Changelog\n\n- feat(x): thing\n- fix(y): other\n"
        sections = gen.parse_release_body(body)
        self.assertEqual(sections, {"Changes": ["feat(x): thing", "fix(y): other"]})

    def test_heading_with_no_bullets_is_dropped(self):
        body = "### Major Features\n\n### Fixes\n- fix(x): only fix present\n"
        sections = gen.parse_release_body(body)
        self.assertNotIn("Major Features", sections)
        self.assertEqual(sections["Fixes"], ["fix(x): only fix present"])


class FormatReleaseDateTests(unittest.TestCase):
    def test_formats_iso8601_timestamp(self):
        self.assertEqual(gen.format_release_date("2026-07-09T14:32:10Z"), "2026-07-09")


class FormatReleaseSectionTests(unittest.TestCase):
    def test_grouped_heading_and_bullets(self):
        sections = {"Major Features": ["feat(x): thing"], "Fixes": ["fix(y): other"]}
        rendered = gen.format_release_section("v0.24.1", "2026-07-09T14:32:10Z", sections)
        self.assertEqual(
            rendered,
            "## v0.24.1 — 2026-07-09\n\n"
            "### Major Features\n\n- feat(x): thing\n\n"
            "### Fixes\n\n- fix(y): other\n",
        )

    def test_fallback_section_renders_as_changes_heading(self):
        sections = {"Changes": ["feat(x): thing"]}
        rendered = gen.format_release_section("v0.22.0", "2026-05-28T09:00:00Z", sections)
        self.assertEqual(
            rendered, "## v0.22.0 — 2026-05-28\n\n### Changes\n\n- feat(x): thing\n"
        )

    def test_bullet_mentioning_hero_command_carries_drift_ignore_marker(self):
        # Verbatim commit subjects that mention a `hero <command>` sequence
        # must not be treated as prescriptive CLI examples by the markdown
        # drift gate — the generator marks the line so the gate skips it.
        sections = {
            "Changes": [
                "feat(verify): hero verify becomes the load-bearing checkpoint",
                "fix(build): tighten packaging step",
            ]
        }
        rendered = gen.format_release_section("v0.16.3", "2026-06-06T00:00:00Z", sections)
        self.assertEqual(
            rendered,
            "## v0.16.3 — 2026-06-06\n\n### Changes\n\n"
            "- feat(verify): hero verify becomes the load-bearing checkpoint "
            "<!-- drift-test:ignore -->\n"
            "- fix(build): tighten packaging step\n",
        )


class SortReleasesNewestFirstTests(unittest.TestCase):
    def test_sorts_by_published_at_descending(self):
        releases = [
            gen.Release("v0.23.0", "2026-06-11T00:00:00Z", ""),
            gen.Release("v0.24.1", "2026-07-09T00:00:00Z", ""),
            gen.Release("v0.24.0", "2026-07-02T00:00:00Z", ""),
        ]
        ordered = gen.sort_releases_newest_first(releases)
        self.assertEqual([r.tag_name for r in ordered], ["v0.24.1", "v0.24.0", "v0.23.0"])


class GeneratePageTests(unittest.TestCase):
    def test_page_lists_releases_newest_first_with_header(self):
        releases = [
            gen.Release("v0.23.0", "2026-06-11T00:00:00Z", UNGROUPED_BODY),
            gen.Release("v0.24.1", "2026-07-09T00:00:00Z", GROUPED_BODY),
        ]
        page = gen.generate_page(releases)
        self.assertTrue(page.startswith("# Releases\n"))
        v24_pos = page.index("## v0.24.1")
        v23_pos = page.index("## v0.23.0")
        self.assertLess(v24_pos, v23_pos, "newest release must render before older ones")
        self.assertIn("### Major Features", page)
        self.assertIn("### Fixes", page)
        self.assertIn("### Changes", page)


class FetchReleasesTests(unittest.TestCase):
    def test_fetch_releases_parses_api_payload_without_network(self):
        payload = [
            {
                "tag_name": "v0.24.1",
                "published_at": "2026-07-09T14:32:10Z",
                "body": GROUPED_BODY,
            }
        ]

        fake_response = mock.MagicMock()
        fake_response.read.return_value = json.dumps(payload).encode("utf-8")
        fake_response.__enter__.return_value = fake_response
        fake_response.__exit__.return_value = False

        with mock.patch("urllib.request.urlopen", return_value=fake_response) as urlopen:
            releases = gen.fetch_releases(repo="hero-engine/hero-releases", token="tok123")

        self.assertEqual(len(releases), 1)
        self.assertEqual(releases[0].tag_name, "v0.24.1")
        request = urlopen.call_args[0][0]
        self.assertIn("hero-engine/hero-releases", request.full_url)
        self.assertEqual(request.headers.get("Authorization"), "Bearer tok123")

    def test_fetch_releases_omits_auth_header_without_token(self):
        fake_response = mock.MagicMock()
        fake_response.read.return_value = b"[]"
        fake_response.__enter__.return_value = fake_response
        fake_response.__exit__.return_value = False

        with mock.patch("urllib.request.urlopen", return_value=fake_response) as urlopen:
            releases = gen.fetch_releases(repo="hero-engine/hero-releases", token=None)

        self.assertEqual(releases, [])
        request = urlopen.call_args[0][0]
        self.assertNotIn("Authorization", request.headers)

    def test_fetch_releases_requests_max_page_size(self):
        fake_response = mock.MagicMock()
        fake_response.read.return_value = b"[]"
        fake_response.__enter__.return_value = fake_response
        fake_response.__exit__.return_value = False

        with mock.patch("urllib.request.urlopen", return_value=fake_response) as urlopen:
            gen.fetch_releases(repo="hero-engine/hero-releases", token=None)

        request = urlopen.call_args[0][0]
        self.assertIn("per_page=100", request.full_url)
        self.assertIn("page=1", request.full_url)

    def test_fetch_releases_pages_past_first_page_of_100(self):
        # Regression test for the pagination bug: a repo with more releases
        # than a single page must not be silently truncated. Simulate three
        # pages — two full pages of 100 (the max per_page) plus a partial
        # third page — and confirm fetch_releases follows page=N until a
        # short page signals the end, rather than stopping at any fixed
        # count like GitHub's un-paginated default of 30.
        page_1 = [
            {"tag_name": f"v0.{i}.0", "published_at": "2026-01-01T00:00:00Z", "body": ""}
            for i in range(100)
        ]
        page_2 = [
            {"tag_name": f"v1.{i}.0", "published_at": "2026-01-01T00:00:00Z", "body": ""}
            for i in range(100)
        ]
        page_3 = [
            {"tag_name": f"v2.{i}.0", "published_at": "2026-01-01T00:00:00Z", "body": ""}
            for i in range(43)
        ]
        pages = [page_1, page_2, page_3]

        responses = []
        for page_payload in pages:
            fake_response = mock.MagicMock()
            fake_response.read.return_value = json.dumps(page_payload).encode("utf-8")
            fake_response.__enter__.return_value = fake_response
            fake_response.__exit__.return_value = False
            responses.append(fake_response)

        with mock.patch("urllib.request.urlopen", side_effect=responses) as urlopen:
            releases = gen.fetch_releases(repo="hero-engine/hero-releases", token=None)

        self.assertEqual(len(releases), 243, "must not stop at GitHub's 30-item default page")
        self.assertEqual(urlopen.call_count, 3, "must stop paging once a short page is returned")
        requested_pages = [
            call.args[0].full_url.split("&page=")[1] for call in urlopen.call_args_list
        ]
        self.assertEqual(requested_pages, ["1", "2", "3"])


class MainWithInputFileTests(unittest.TestCase):
    def test_main_writes_page_from_fixture_and_is_idempotent(self):
        payload = [
            {
                "tag_name": "v0.24.1",
                "published_at": "2026-07-09T14:32:10Z",
                "body": GROUPED_BODY,
            },
            {
                "tag_name": "v0.22.0",
                "published_at": "2026-05-28T09:00:00Z",
                "body": UNGROUPED_BODY,
            },
        ]

        with tempfile.TemporaryDirectory() as tmpdir:
            fixture_path = os.path.join(tmpdir, "releases.json")
            output_path = os.path.join(tmpdir, "releases", "index.md")
            with open(fixture_path, "w", encoding="utf-8") as f:
                json.dump(payload, f)

            exit_code_1 = gen.main(["--input-file", fixture_path, "--output", output_path])
            with open(output_path, encoding="utf-8") as f:
                first_run = f.read()

            exit_code_2 = gen.main(["--input-file", fixture_path, "--output", output_path])
            with open(output_path, encoding="utf-8") as f:
                second_run = f.read()

        self.assertEqual(exit_code_1, 0)
        self.assertEqual(exit_code_2, 0)
        self.assertEqual(first_run, second_run, "re-running against the same input must be idempotent")
        self.assertIn("## v0.24.1", first_run)
        self.assertIn("## v0.22.0", first_run)

    def test_main_errors_on_empty_release_list(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            fixture_path = os.path.join(tmpdir, "releases.json")
            output_path = os.path.join(tmpdir, "releases", "index.md")
            with open(fixture_path, "w", encoding="utf-8") as f:
                json.dump([], f)

            exit_code = gen.main(["--input-file", fixture_path, "--output", output_path])

        self.assertEqual(exit_code, 1)


if __name__ == "__main__":
    unittest.main()
