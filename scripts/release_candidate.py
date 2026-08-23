#!/usr/bin/env python3
"""Build a reproducible, unpublished Hero release candidate."""

from __future__ import annotations

import argparse
import datetime as dt
import gzip
import hashlib
import json
import os
import platform
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import zipfile
from pathlib import Path
from typing import Any, Iterable


TARGETS = (
    ("darwin", "amd64"),
    ("darwin", "arm64"),
    ("linux", "amd64"),
    ("linux", "arm64"),
    ("windows", "amd64"),
)

LICENSE_IDS = {
    "github.com/BurntSushi/toml": "MIT",
    "github.com/dustin/go-humanize": "MIT",
    "github.com/google/uuid": "BSD-3-Clause",
    "github.com/inconshreveable/mousetrap": "Apache-2.0",
    "github.com/mattn/go-isatty": "MIT",
    "github.com/ncruces/go-strftime": "MIT",
    "github.com/remyoudompheng/bigfft": "BSD-3-Clause",
    "github.com/spf13/cobra": "Apache-2.0",
    "github.com/spf13/pflag": "BSD-3-Clause",
    "golang.org/x/crypto": "BSD-3-Clause",
    "golang.org/x/sys": "BSD-3-Clause",
    "golang.org/x/term": "BSD-3-Clause",
    "gopkg.in/yaml.v3": "MIT",
    "modernc.org/libc": "BSD-3-Clause",
    "modernc.org/mathutil": "BSD-3-Clause",
    "modernc.org/memory": "BSD-3-Clause",
    "modernc.org/sqlite": "BSD-3-Clause",
}


class CandidateError(RuntimeError):
    pass


def run(
    args: Iterable[str],
    *,
    cwd: Path,
    env: dict[str, str] | None = None,
    text: bool = True,
) -> str | bytes:
    command = [str(arg) for arg in args]
    try:
        result = subprocess.run(
            command,
            cwd=cwd,
            env=env,
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=text,
        )
    except subprocess.CalledProcessError as exc:
        detail = exc.stderr.decode() if isinstance(exc.stderr, bytes) else exc.stderr
        raise CandidateError(f"{' '.join(command)} failed: {detail.strip()}") from exc
    return result.stdout


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def parse_json_stream(payload: str) -> list[dict[str, Any]]:
    decoder = json.JSONDecoder()
    values: list[dict[str, Any]] = []
    position = 0
    while position < len(payload):
        while position < len(payload) and payload[position].isspace():
            position += 1
        if position >= len(payload):
            break
        value, position = decoder.raw_decode(payload, position)
        values.append(value)
    return values


def semver_key(tag: str) -> tuple[int, int, int]:
    match = re.fullmatch(r"v(\d+)\.(\d+)\.(\d+)", tag)
    if not match:
        raise CandidateError(f"release tag must be vMAJOR.MINOR.PATCH, got {tag!r}")
    return tuple(int(part) for part in match.groups())  # type: ignore[return-value]


def source_identity(root: Path, version: str, base: str) -> dict[str, Any]:
    target = semver_key(version)
    baseline = semver_key(base)
    if target <= baseline:
        raise CandidateError(f"candidate {version} must be newer than baseline {base}")

    tags = str(run(["git", "tag", "--list", "v*"], cwd=root)).splitlines()
    release_tags = sorted((tag for tag in tags if re.fullmatch(r"v\d+\.\d+\.\d+", tag)), key=semver_key)
    if not release_tags or release_tags[-1] != base:
        latest = release_tags[-1] if release_tags else "none"
        raise CandidateError(f"baseline {base} is not the latest release tag (latest: {latest})")
    if version in tags:
        raise CandidateError(f"candidate tag already exists: {version}")

    status = str(run(["git", "status", "--porcelain", "--untracked-files=all"], cwd=root))
    if status.strip():
        raise CandidateError("release candidates must be built from a clean checkout")

    revision = str(run(["git", "rev-parse", "HEAD"], cwd=root)).strip()
    tree = str(run(["git", "rev-parse", "HEAD^{tree}"], cwd=root)).strip()
    epoch = int(str(run(["git", "show", "-s", "--format=%ct", "HEAD"], cwd=root)).strip())
    return {
        "version": version,
        "baseline": base,
        "revision": revision,
        "source_tree": tree,
        "source_date_epoch": epoch,
    }


def go_environment(root: Path) -> tuple[dict[str, str], str]:
    version_output = str(run(["go", "version"], cwd=root)).strip()
    match = re.search(r"\bgo(\d+\.\d+(?:\.\d+)?)\b", version_output)
    if not match:
        raise CandidateError(f"cannot parse Go version from {version_output!r}")
    env = os.environ.copy()
    env.update({"CGO_ENABLED": "0", "GOFLAGS": "-mod=readonly"})
    return env, match.group(1)


def module_inventory(root: Path, base_env: dict[str, str]) -> list[dict[str, str]]:
    modules: dict[tuple[str, str], dict[str, str]] = {}
    for goos, goarch in TARGETS:
        env = base_env | {"GOOS": goos, "GOARCH": goarch}
        payload = str(run(["go", "list", "-deps", "-json", "./cmd/hero"], cwd=root, env=env))
        for package in parse_json_stream(payload):
            module = package.get("Module")
            if not module or module.get("Main"):
                continue
            replacement = module.get("Replace") or module
            path = module["Path"]
            version = module.get("Version", "")
            key = (path, version)
            modules[key] = {
                "path": path,
                "version": version,
                "dir": replacement.get("Dir", ""),
                "sum": module.get("Sum", ""),
            }

    inventory = [modules[key] for key in sorted(modules)]
    unknown = sorted(item["path"] for item in inventory if item["path"] not in LICENSE_IDS)
    if unknown:
        raise CandidateError("release dependency has no reviewed license mapping: " + ", ".join(unknown))
    return inventory


def license_files(module: dict[str, str]) -> list[Path]:
    directory = Path(module["dir"])
    if not directory.is_dir():
        raise CandidateError(f"module directory unavailable for {module['path']} {module['version']}")
    files = sorted(
        path
        for path in directory.iterdir()
        if path.is_file() and re.match(r"(?i)^(license|copying|notice)", path.name)
    )
    if not files:
        raise CandidateError(f"no license/notice file found for {module['path']} {module['version']}")
    return files


def render_notices(root: Path, modules: list[dict[str, str]]) -> bytes:
    sections = [
        "Hero release candidate — third-party notices\n",
        "Third-party components retain their own licenses. This packet does not license Hero itself.\n",
    ]
    go_license = root / "release" / "third-party" / "go-BSD-3-Clause.txt"
    model_license = root / "internal" / "embeddings" / "defaultmodel" / "potion-base-8M-MIT.txt"
    for title, identifier, files in (
        ("Go runtime and standard library", "BSD-3-Clause", [go_license]),
        ("Embedded hero-embed-v1 model lineage", "MIT", [model_license]),
    ):
        sections.append(f"\n{'=' * 78}\n{title}\nLicense: {identifier}\n{'=' * 78}\n")
        for path in files:
            sections.append(path.read_text(encoding="utf-8").rstrip() + "\n")

    for module in modules:
        sections.append(
            f"\n{'=' * 78}\n{module['path']} {module['version']}\n"
            f"License: {LICENSE_IDS[module['path']]}\n{'=' * 78}\n"
        )
        for path in license_files(module):
            sections.append(f"\n--- {path.name} ---\n")
            sections.append(path.read_text(encoding="utf-8", errors="replace").rstrip() + "\n")
    return "".join(sections).encode("utf-8")


def render_sbom(
    identity: dict[str, Any],
    go_version: str,
    modules: list[dict[str, str]],
) -> bytes:
    timestamp = dt.datetime.fromtimestamp(identity["source_date_epoch"], dt.timezone.utc).isoformat().replace("+00:00", "Z")
    components: list[dict[str, Any]] = [
        {
            "type": "application",
            "name": "hero",
            "version": identity["version"].removeprefix("v"),
            "bom-ref": f"pkg:golang/github.com/hero-engine/hero@{identity['version'].removeprefix('v')}",
            "properties": [{"name": "hero:license-gate", "value": "pending-apache-2.0"}],
        },
        {
            "type": "framework",
            "name": "Go standard library",
            "version": go_version,
            "bom-ref": f"pkg:golang/stdlib@{go_version}",
            "licenses": [{"license": {"id": "BSD-3-Clause"}}],
        },
        {
            "type": "data",
            "name": "hero-embed-v1",
            "version": "bf8b056651a2c21b8d2565580b8569da283cab23",
            "bom-ref": "hero:model:hero-embed-v1",
            "licenses": [{"license": {"id": "MIT"}}],
        },
    ]
    for module in modules:
        component: dict[str, Any] = {
            "type": "library",
            "name": module["path"],
            "version": module["version"],
            "bom-ref": f"pkg:golang/{module['path']}@{module['version']}",
            "purl": f"pkg:golang/{module['path']}@{module['version']}",
            "licenses": [{"license": {"id": LICENSE_IDS[module["path"]]}}],
        }
        if module["sum"]:
            component["properties"] = [{"name": "go:module-sum", "value": module["sum"]}]
        components.append(component)

    document = {
        "bomFormat": "CycloneDX",
        "specVersion": "1.5",
        "version": 1,
        "metadata": {
            "timestamp": timestamp,
            "tools": {"components": [{"type": "application", "name": "Hero release candidate builder"}]},
            "component": components[0],
            "properties": [
                {"name": "hero:source-revision", "value": identity["revision"]},
                {"name": "hero:source-tree", "value": identity["source_tree"]},
            ],
        },
        "components": components[1:],
    }
    return (json.dumps(document, indent=2, sort_keys=True) + "\n").encode("utf-8")


def candidate_readme(identity: dict[str, Any]) -> bytes:
    return (
        f"Hero {identity['version']} release candidate\n"
        f"Source revision: {identity['revision']}\n"
        f"Source tree: {identity['source_tree']}\n"
        f"Baseline: {identity['baseline']}\n"
        "Publication status: unpublished\n"
        "Hero license gate: pending Apache-2.0 approval\n"
        "This candidate is for verification only and must be rebuilt after the license gate.\n"
    ).encode("utf-8")


def normalized_tar(path: Path, files: dict[str, tuple[bytes, int]], epoch: int) -> None:
    with path.open("wb") as raw:
        with gzip.GzipFile(filename="", mode="wb", fileobj=raw, compresslevel=9, mtime=epoch) as compressed:
            with tarfile.open(fileobj=compressed, mode="w", format=tarfile.PAX_FORMAT) as archive:
                for name in sorted(files):
                    content, mode = files[name]
                    info = tarfile.TarInfo(name)
                    info.size = len(content)
                    info.mode = mode
                    info.mtime = epoch
                    info.uid = 0
                    info.gid = 0
                    info.uname = "root"
                    info.gname = "root"
                    archive.addfile(info, __import__("io").BytesIO(content))


def normalized_zip(path: Path, files: dict[str, tuple[bytes, int]], epoch: int) -> None:
    stamp = dt.datetime.fromtimestamp(max(epoch, 315532800), dt.timezone.utc)
    date_time = (stamp.year, stamp.month, stamp.day, stamp.hour, stamp.minute, stamp.second)
    with zipfile.ZipFile(path, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for name in sorted(files):
            content, mode = files[name]
            info = zipfile.ZipInfo(name, date_time=date_time)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.create_system = 3
            info.external_attr = (mode & 0xFFFF) << 16
            archive.writestr(info, content)


def build_binary(
    root: Path,
    destination: Path,
    version: str,
    goos: str,
    goarch: str,
    base_env: dict[str, str],
    epoch: int,
) -> None:
    env = base_env | {
        "GOOS": goos,
        "GOARCH": goarch,
        "SOURCE_DATE_EPOCH": str(epoch),
    }
    run(
        [
            "go",
            "build",
            "-trimpath",
            "-buildvcs=false",
            "-ldflags",
            f"-s -w -X main.version={version}",
            "-o",
            str(destination),
            "./cmd/hero",
        ],
        cwd=root,
        env=env,
    )


def build_once(
    root: Path,
    output: Path,
    identity: dict[str, Any],
    base_env: dict[str, str],
    go_version: str,
    modules: list[dict[str, str]],
) -> dict[str, str]:
    output.mkdir(parents=True)
    notices = render_notices(root, modules)
    sbom = render_sbom(identity, go_version, modules)
    readme = candidate_readme(identity)
    (output / "THIRD_PARTY_NOTICES.txt").write_bytes(notices)
    (output / "hero-v0.34.0.cdx.json").write_bytes(sbom)

    artifact_hashes: dict[str, str] = {}
    for goos, goarch in TARGETS:
        executable = "hero.exe" if goos == "windows" else "hero"
        binary_path = output / f".{goos}-{goarch}-{executable}"
        build_binary(root, binary_path, identity["version"], goos, goarch, base_env, identity["source_date_epoch"])
        files = {
            executable: (binary_path.read_bytes(), 0o755),
            "THIRD_PARTY_NOTICES.txt": (notices, 0o644),
            "RELEASE-CANDIDATE.txt": (readme, 0o644),
        }
        stem = f"hero_{identity['version'].removeprefix('v')}_{goos}_{goarch}"
        archive_path = output / (f"{stem}.zip" if goos == "windows" else f"{stem}.tar.gz")
        if goos == "windows":
            normalized_zip(archive_path, files, identity["source_date_epoch"])
        else:
            normalized_tar(archive_path, files, identity["source_date_epoch"])
        binary_path.unlink()
        artifact_hashes[archive_path.name] = sha256(archive_path)

    artifact_hashes["THIRD_PARTY_NOTICES.txt"] = sha256(output / "THIRD_PARTY_NOTICES.txt")
    artifact_hashes["hero-v0.34.0.cdx.json"] = sha256(output / "hero-v0.34.0.cdx.json")
    provenance = {
        **identity,
        "go_version": go_version,
        "targets": [f"{goos}/{goarch}" for goos, goarch in TARGETS],
        "artifacts": artifact_hashes,
        "reproducible_build": True,
        "publication_status": "unpublished",
        "hero_license_gate": "pending-apache-2.0",
    }
    provenance_path = output / "provenance.json"
    provenance_path.write_text(json.dumps(provenance, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    artifact_hashes[provenance_path.name] = sha256(provenance_path)
    checksums = "".join(f"{digest}  {name}\n" for name, digest in sorted(artifact_hashes.items()))
    (output / "checksums.txt").write_text(checksums, encoding="utf-8")
    return artifact_hashes | {"checksums.txt": sha256(output / "checksums.txt")}


def compare_outputs(first: Path, second: Path) -> None:
    first_files = {path.name: sha256(path) for path in first.iterdir() if path.is_file()}
    second_files = {path.name: sha256(path) for path in second.iterdir() if path.is_file()}
    if first_files != second_files:
        differing = sorted(set(first_files) | set(second_files))
        detail = [name for name in differing if first_files.get(name) != second_files.get(name)]
        raise CandidateError("candidate build is not reproducible: " + ", ".join(detail))


def host_target() -> tuple[str, str] | None:
    systems = {"Darwin": "darwin", "Linux": "linux", "Windows": "windows"}
    machines = {"x86_64": "amd64", "AMD64": "amd64", "arm64": "arm64", "aarch64": "arm64"}
    goos = systems.get(platform.system())
    goarch = machines.get(platform.machine())
    return (goos, goarch) if goos and goarch else None


def smoke_candidate(output: Path, identity: dict[str, Any]) -> None:
    target = host_target()
    if target not in TARGETS:
        raise CandidateError(f"no candidate smoke target for host {platform.system()}/{platform.machine()}")
    goos, goarch = target
    stem = f"hero_{identity['version'].removeprefix('v')}_{goos}_{goarch}"
    archive = output / (f"{stem}.zip" if goos == "windows" else f"{stem}.tar.gz")
    with tempfile.TemporaryDirectory(prefix="hero-candidate-smoke-") as temp_name:
        temp = Path(temp_name)
        if archive.suffix == ".zip":
            with zipfile.ZipFile(archive) as bundle:
                bundle.extractall(temp / "candidate")
        else:
            with tarfile.open(archive, "r:gz") as bundle:
                bundle.extractall(temp / "candidate", filter="data")
        binary = temp / "candidate" / ("hero.exe" if goos == "windows" else "hero")
        version_output = str(run([str(binary), "--version"], cwd=temp)).strip()
        if identity["version"] not in version_output:
            raise CandidateError(f"candidate version smoke failed: {version_output}")

        project = temp / "project"
        project.mkdir()
        run(["git", "init", "-q"], cwd=project)
        isolated_env = os.environ.copy() | {
            "HOME": str(temp / "home"),
            "XDG_CONFIG_HOME": str(temp / "xdg-config"),
            "XDG_CACHE_HOME": str(temp / "xdg-cache"),
            "XDG_DATA_HOME": str(temp / "xdg-data"),
        }
        run([str(binary), "init", "--no-hooks"], cwd=project, env=isolated_env)
        run(
            [str(binary), "install", "project", ".", "--target", "codex", "--no-hooks"],
            cwd=project,
            env=isolated_env,
        )
        run([str(binary), "status"], cwd=project, env=isolated_env)
        run([str(binary), "check"], cwd=project, env=isolated_env)
        required = [
            project / ".hero" / "hero.json",
            project / ".agents" / "skills" / "command-deliver" / "SKILL.md",
            project / ".codex" / "agents" / "engineer.toml",
        ]
        missing = [str(path.relative_to(project)) for path in required if not path.is_file()]
        if missing:
            raise CandidateError("candidate install smoke missed: " + ", ".join(missing))


def build_candidate(root: Path, output: Path, version: str, base: str, smoke: bool) -> dict[str, Any]:
    identity = source_identity(root, version, base)
    base_env, go_version = go_environment(root)
    modules = module_inventory(root, base_env)
    with tempfile.TemporaryDirectory(prefix="hero-candidate-a-") as first_name, tempfile.TemporaryDirectory(
        prefix="hero-candidate-b-"
    ) as second_name:
        first = Path(first_name)
        second = Path(second_name)
        build_once(root, first, identity, base_env, go_version, modules)
        build_once(root, second, identity, base_env, go_version, modules)
        compare_outputs(first, second)
        if output.exists():
            shutil.rmtree(output)
        shutil.copytree(first, output)
    if smoke:
        smoke_candidate(output, identity)
    return identity | {"output": str(output), "module_count": len(modules), "smoke": smoke}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", default="v0.34.0")
    parser.add_argument("--base", default="v0.33.0")
    parser.add_argument("--output", default=".build/release-candidate/v0.34.0")
    parser.add_argument("--no-smoke", action="store_true", help="skip the native clean-install smoke")
    args = parser.parse_args()
    try:
        root = Path(str(run(["git", "rev-parse", "--show-toplevel"], cwd=Path.cwd()))).resolve()
        output = (root / args.output).resolve() if not Path(args.output).is_absolute() else Path(args.output).resolve()
        if output == root or root not in output.parents:
            raise CandidateError("output must be a child of the repository")
        result = build_candidate(root, output, args.version, args.base, not args.no_smoke)
    except CandidateError as exc:
        print(f"release candidate failed: {exc}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
