import importlib.util
import json
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("release_candidate.py")
SPEC = importlib.util.spec_from_file_location("release_candidate", SCRIPT)
assert SPEC and SPEC.loader
release_candidate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release_candidate)


class ReleaseCandidateTests(unittest.TestCase):
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


if __name__ == "__main__":
    unittest.main()
