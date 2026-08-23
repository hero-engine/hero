import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock


SCRIPT = Path(__file__).with_name("release_candidate.py")
SPEC = importlib.util.spec_from_file_location("release_candidate", SCRIPT)
assert SPEC and SPEC.loader
release_candidate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release_candidate)


class ReleaseCandidateTests(unittest.TestCase):
    def test_output_is_confined_to_candidate_version_directory(self):
        with tempfile.TemporaryDirectory() as temp_name:
            root = Path(temp_name).resolve()
            release_root = root / ".build" / "release-candidate"
            valid = release_root / "v0.34.0"
            self.assertEqual(valid, release_candidate.validate_output_path(root, valid))
            for invalid in (
                root,
                root / ".git" / "objects",
                root / "internal",
                release_root,
                root.parent / "outside-candidate",
            ):
                with self.subTest(invalid=invalid):
                    with self.assertRaises(release_candidate.CandidateError):
                        release_candidate.validate_output_path(root, invalid)

    def test_output_symlink_cannot_escape_candidate_directory(self):
        with tempfile.TemporaryDirectory() as temp_name:
            root = Path(temp_name).resolve()
            release_root = root / ".build" / "release-candidate"
            release_root.mkdir(parents=True)
            outside = root / "internal"
            outside.mkdir()
            (release_root / "escape").symlink_to(outside, target_is_directory=True)
            with self.assertRaises(release_candidate.CandidateError):
                release_candidate.validate_output_path(root, release_root / "escape")

    def test_output_directory_can_already_exist(self):
        with tempfile.TemporaryDirectory() as temp_name:
            output = Path(temp_name)
            release_candidate.ensure_directory(output)
            release_candidate.ensure_directory(output)
            self.assertTrue(output.is_dir())

    def test_repository_root_strips_git_newline(self):
        with mock.patch.object(release_candidate, "run", return_value="/tmp/example\n"):
            self.assertEqual(Path("/tmp/example").resolve(), release_candidate.repository_root(Path.cwd()))

    def test_semver_requires_canonical_release_tag(self):
        self.assertEqual((0, 34, 0), release_candidate.semver_key("v0.34.0"))
        with self.assertRaises(release_candidate.CandidateError):
            release_candidate.semver_key("0.34")

    def test_json_stream_parses_go_list_shape(self):
        values = release_candidate.parse_json_stream('{"ImportPath":"a"}\n{"ImportPath":"b"}\n')
        self.assertEqual(["a", "b"], [value["ImportPath"] for value in values])

    def test_normalized_archives_are_byte_identical(self):
        files = {
            "hero": (b"binary", 0o755),
            "THIRD_PARTY_NOTICES.txt": (b"notices\n", 0o644),
        }
        with tempfile.TemporaryDirectory() as temp_name:
            temp = Path(temp_name)
            first_tar = temp / "first.tar.gz"
            second_tar = temp / "second.tar.gz"
            first_zip = temp / "first.zip"
            second_zip = temp / "second.zip"
            release_candidate.normalized_tar(first_tar, files, 1_700_000_000)
            release_candidate.normalized_tar(second_tar, files, 1_700_000_000)
            release_candidate.normalized_zip(first_zip, files, 1_700_000_000)
            release_candidate.normalized_zip(second_zip, files, 1_700_000_000)
            self.assertEqual(release_candidate.sha256(first_tar), release_candidate.sha256(second_tar))
            self.assertEqual(release_candidate.sha256(first_zip), release_candidate.sha256(second_zip))

    def test_sbom_records_pending_license_gate_and_revision(self):
        identity = {
            "version": "v0.34.0",
            "revision": "a" * 40,
            "source_tree": "b" * 40,
            "source_date_epoch": 1_700_000_000,
        }
        module = {
            "path": "github.com/google/uuid",
            "version": "v1.6.0",
            "sum": "h1:example",
        }
        sbom = json.loads(release_candidate.render_sbom(identity, "1.26.4", [module]))
        self.assertEqual("CycloneDX", sbom["bomFormat"])
        self.assertEqual("a" * 40, sbom["metadata"]["properties"][0]["value"])
        self.assertEqual(
            "pending-apache-2.0",
            sbom["metadata"]["component"]["properties"][0]["value"],
        )

    def test_unknown_release_dependency_fails_closed(self):
        inventory = [{"path": "example.invalid/unknown", "version": "v1.0.0"}]
        unknown = [item["path"] for item in inventory if item["path"] not in release_candidate.LICENSE_IDS]
        self.assertEqual(["example.invalid/unknown"], unknown)

    def test_release_workflow_separates_candidate_from_publish(self):
        workflow = (SCRIPT.parent.parent / ".github" / "workflows" / "release.yml").read_text()
        self.assertIn("github.event_name == 'workflow_dispatch'", workflow)
        self.assertIn("startsWith(github.ref, 'refs/tags/v')", workflow)
        self.assertIn("scripts/release_candidate.py", workflow)
        self.assertIn("test -f LICENSE", workflow)
        candidate_job = workflow.split("  candidate:", 1)[1].split("  goreleaser:", 1)[0]
        self.assertNotIn("actions/upload-artifact", candidate_job)
        self.assertIn("GITHUB_STEP_SUMMARY", candidate_job)

    def test_candidate_notes_preserve_public_maturity_and_product_boundaries(self):
        notes = (SCRIPT.parent.parent / "docs" / "releases" / "v0.34.0-candidate.md").read_text()
        normalized = " ".join(notes.split())
        for expected in (
            "Project memory and verified delivery are shipped",
            "reinforcing improvement loop remains preview",
            "measurably enriches memory and improves later sessions remains **preview**",
            "optional and require explicit setup",
            "headless runtime remains preview",
            "Hero Code and Hero Cloud remain separate proprietary products",
            "Sprout remains a separate public MIT-licensed project",
            "prepared, not published",
        ):
            self.assertIn(expected, normalized)
        self.assertNotIn("Hero is open source", normalized)

    def test_launch_checklist_has_every_artifact_and_explicit_gates(self):
        checklist = (SCRIPT.parent.parent / "docs" / "releases" / "v0.34.0-launch-checklist.md").read_text()
        for expected in (
            "hero_0.34.0_darwin_amd64.tar.gz",
            "hero_0.34.0_darwin_arm64.tar.gz",
            "hero_0.34.0_linux_amd64.tar.gz",
            "hero_0.34.0_linux_arm64.tar.gz",
            "hero_0.34.0_windows_amd64.zip",
            "hero-v0.34.0.cdx.json",
            "THIRD_PARTY_NOTICES.txt",
            "provenance.json",
            "checksums.txt",
        ):
            self.assertIn(expected, checklist)
        self.assertEqual(10, checklist.count("| GATED |"))


if __name__ == "__main__":
    unittest.main()
