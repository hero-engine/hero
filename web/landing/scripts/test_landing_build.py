from __future__ import annotations

import json
import os
import shutil
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import landing_build


HEAD = "a" * 40
OTHER_HEAD = "b" * 40
BUILT_AT = "2026-08-23T12:00:00Z"


class LandingBuildTests(unittest.TestCase):
    def _build_clean(self, source: Path, artifact: Path) -> dict[str, str | bool]:
        with (
            mock.patch.dict(
                os.environ,
                {
                    "HERO_LANDING_REVISION": HEAD,
                    "HERO_LANDING_BUILT_AT": BUILT_AT,
                },
            ),
            mock.patch.object(landing_build, "_git", return_value=HEAD),
            mock.patch.object(landing_build, "_source_is_dirty", return_value=False),
        ):
            return landing_build.build_artifact(source, artifact)

    def test_tracked_source_passes_claim_accessibility_link_and_asset_checks(self) -> None:
        self.assertEqual(
            landing_build.validate_site(landing_build.DEFAULT_SOURCE, built=False), []
        )

    def test_clean_explicit_build_binds_commit_and_exact_source_digest(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / "artifact"
            metadata = self._build_clean(landing_build.DEFAULT_SOURCE, artifact)

            self.assertEqual(metadata["source_revision"], HEAD)
            self.assertEqual(metadata["source_commit"], HEAD)
            self.assertEqual(
                metadata["source_digest"],
                landing_build.source_tree_digest(landing_build.DEFAULT_SOURCE),
            )
            self.assertIs(metadata["source_dirty"], False)
            with mock.patch.object(landing_build, "_git", return_value=HEAD):
                self.assertEqual(landing_build.validate_site(artifact, built=True), [])
            rendered = (artifact / "index.html").read_text(encoding="utf-8")
            self.assertIn(HEAD, rendered)
            self.assertNotIn(landing_build.SOURCE_REVISION, rendered)
            self.assertEqual(
                json.loads((artifact / "revision.json").read_text(encoding="utf-8")),
                metadata,
            )

    def test_dirty_local_build_uses_commit_plus_worktree_digest(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / "artifact"
            with (
                mock.patch.dict(
                    os.environ,
                    {
                        "HERO_LANDING_REVISION": "",
                        "HERO_LANDING_BUILT_AT": BUILT_AT,
                    },
                ),
                mock.patch.object(landing_build, "_git", return_value=HEAD),
                mock.patch.object(
                    landing_build, "_source_is_dirty", return_value=True
                ),
            ):
                metadata = landing_build.build_artifact(
                    landing_build.DEFAULT_SOURCE, artifact
                )

            digest = landing_build.source_tree_digest(landing_build.DEFAULT_SOURCE)
            self.assertEqual(
                metadata["source_revision"], f"{HEAD}+worktree:{digest}"
            )
            self.assertEqual(metadata["source_commit"], HEAD)
            self.assertEqual(metadata["source_digest"], digest)
            self.assertIs(metadata["source_dirty"], True)
            with mock.patch.object(landing_build, "_git", return_value=HEAD):
                self.assertEqual(landing_build.validate_site(artifact, built=True), [])

    def test_explicit_revision_rejects_arbitrary_mismatch_and_dirty_source(self) -> None:
        cases = (
            ("abcdef", HEAD, False, "exact 40-character"),
            (OTHER_HEAD, HEAD, False, "equal the checked-out Git HEAD"),
            (HEAD, HEAD, True, "require web/landing/site to be clean"),
        )
        for explicit, current, dirty, message in cases:
            with self.subTest(explicit=explicit, dirty=dirty):
                with (
                    mock.patch.dict(
                        os.environ,
                        {
                            "HERO_LANDING_REVISION": explicit,
                            "HERO_LANDING_BUILT_AT": BUILT_AT,
                        },
                    ),
                    mock.patch.object(landing_build, "_git", return_value=current),
                    mock.patch.object(
                        landing_build, "_source_is_dirty", return_value=dirty
                    ),
                ):
                    with self.assertRaisesRegex(ValueError, message):
                        landing_build.build_metadata()

    def test_validator_rejects_stale_claim_and_missing_asset(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            site = Path(directory)
            for source in landing_build.DEFAULT_SOURCE.iterdir():
                if source.is_file():
                    (site / source.name).write_bytes(source.read_bytes())
            index = site / "index.html"
            index.write_text(
                index.read_text(encoding="utf-8").replace(
                    "Local-first project context",
                    "v0.9 open source · MIT approval-aware agent jobs. Your agents finish. "
                    "Preview outcome: the continuity demonstration is still being proven. "
                    "Repository boundary: Artifact revision BUILD_TIME_GENERATED_AT.",
                ).replace('/favicon.svg', '/missing.svg', 1),
                encoding="utf-8",
            )
            logo = site / "hero-logo.svg"
            logo.write_text(
                '<svg viewBox="0 0 90 90"><path d="'
                + landing_build.OLD_BOLT_PATH
                + '"/></svg>',
                encoding="utf-8",
            )

            errors = landing_build.validate_site(site, built=False)
            self.assertTrue(any("v0.9" in error for error in errors))
            self.assertTrue(any("open source" in error for error in errors))
            self.assertTrue(any("approval-aware agent jobs" in error for error in errors))
            self.assertTrue(any("open source · mit" in error for error in errors))
            self.assertTrue(any("your agents finish" in error for error in errors))
            self.assertTrue(any("preview outcome" in error for error in errors))
            self.assertTrue(any("continuity demonstration" in error for error in errors))
            self.assertTrue(any("repository boundary" in error for error in errors))
            self.assertTrue(any("artifact revision" in error for error in errors))
            self.assertTrue(any("build_time_generated_at" in error for error in errors))
            self.assertTrue(any("missing local asset" in error for error in errors))
            self.assertTrue(any("canonical Hero logo geometry" in error for error in errors))
            self.assertTrue(any("obsolete bolt mark" in error for error in errors))

    def test_built_artifact_fails_closed_on_unresolved_metadata(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / "artifact"
            self._build_clean(landing_build.DEFAULT_SOURCE, artifact)
            index = artifact / "index.html"
            index.write_text(
                index.read_text(encoding="utf-8").replace(
                    HEAD, landing_build.SOURCE_REVISION, 1
                ),
                encoding="utf-8",
            )

            with mock.patch.object(landing_build, "_git", return_value=HEAD):
                errors = landing_build.validate_site(artifact, built=True)
            self.assertTrue(
                any("metadata placeholders" in error for error in errors), errors
            )

    def test_artifact_rejects_source_digest_mismatch(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "source"
            artifact = root / "artifact"
            shutil.copytree(landing_build.DEFAULT_SOURCE, source)
            self._build_clean(source, artifact)
            with (source / "robots.txt").open("a", encoding="utf-8") as file:
                file.write("# changed after build\n")

            with mock.patch.object(landing_build, "_git", return_value=HEAD):
                errors = landing_build.validate_site(
                    artifact, built=True, source_dir=source
                )
            self.assertTrue(
                any("does not match the current landing source tree" in error for error in errors),
                errors,
            )

    def test_artifact_rejects_arbitrary_source_commit_override(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / "artifact"
            self._build_clean(landing_build.DEFAULT_SOURCE, artifact)
            revision_path = artifact / "revision.json"
            metadata = json.loads(revision_path.read_text(encoding="utf-8"))
            metadata["source_commit"] = OTHER_HEAD
            metadata["source_revision"] = OTHER_HEAD
            revision_path.write_text(
                json.dumps(metadata, indent=2) + "\n", encoding="utf-8"
            )
            index = artifact / "index.html"
            index.write_text(
                index.read_text(encoding="utf-8").replace(HEAD, OTHER_HEAD),
                encoding="utf-8",
            )

            with mock.patch.object(landing_build, "_git", return_value=HEAD):
                errors = landing_build.validate_site(artifact, built=True)
            self.assertTrue(
                any("does not match the checked-out Git HEAD" in error for error in errors),
                errors,
            )


if __name__ == "__main__":
    unittest.main()
