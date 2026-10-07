#!/usr/bin/env python3
"""Export read contract v1 replies from a real `hero mcp` as fixture JSON.

Writes hero_work.json, hero_handoff.json and hero_spec/<slug>.json (one per
work item) into OUT_DIR, plus manifest.json recording the replying Hero's version, the
revision, and the exported project's commit (not Hero's source commit). Clients (e.g. hero-harness's fixture server)
serve these instead of running Hero in tests.

Usage: scripts/export-read-contract-fixture.py [--hero PATH] [--project DIR] OUT_DIR
"""
import argparse
import json
import os
import subprocess
import sys


def call(proc, msg_id, method, params):
    proc.stdin.write(json.dumps({"jsonrpc": "2.0", "id": msg_id, "method": method, "params": params}) + "\n")
    proc.stdin.flush()
    while True:
        line = proc.stdout.readline()
        if not line:
            sys.exit(f"hero mcp exited before answering {method}")
        reply = json.loads(line)
        if reply.get("id") == msg_id:
            if "error" in reply:
                sys.exit(f"{method}: {reply['error']}")
            return reply["result"]


def tool(proc, msg_id, name, args):
    result = call(proc, msg_id, "tools/call", {"name": name, "arguments": args})
    text = result["content"][0]["text"]
    if result.get("isError"):
        sys.exit(f"{name}: {text}")
    return json.loads(text)


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--hero", default="hero", help="hero binary (default: hero on PATH)")
    ap.add_argument("--project", default=".", help="Hero project root (default: .)")
    ap.add_argument("out_dir")
    args = ap.parse_args()

    os.makedirs(os.path.join(args.out_dir, "hero_spec"), exist_ok=True)
    proc = subprocess.Popen([args.hero, "mcp"], cwd=args.project, stdin=subprocess.PIPE,
                            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
    try:
        call(proc, 1, "initialize", {"protocolVersion": "2024-11-05", "capabilities": {},
                                     "clientInfo": {"name": "read-contract-fixture", "version": "1"}})
        work = tool(proc, 2, "hero_work", {})
        handoff = tool(proc, 3, "hero_handoff", {})
        for i, item in enumerate(work["items"]):
            spec = tool(proc, 100 + i, "hero_spec", {"slug": item["slug"]})
            with open(os.path.join(args.out_dir, "hero_spec", item["slug"] + ".json"), "w") as f:
                json.dump(spec, f, indent=2)
    finally:
        proc.stdin.close()
        proc.wait(timeout=10)

    for name, value in (("hero_work.json", work), ("hero_handoff.json", handoff)):
        with open(os.path.join(args.out_dir, name), "w") as f:
            json.dump(value, f, indent=2)
    commit = subprocess.run(["git", "-C", args.project, "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
    manifest = {"schema_version": work["schema_version"], "hero_version": work["hero_version"],
                "revision": work["revision"], "project_commit": commit or None,
                "items": len(work["items"]), "generated_at": work["generated_at"]}
    with open(os.path.join(args.out_dir, "manifest.json"), "w") as f:
        json.dump(manifest, f, indent=2)
    print(json.dumps(manifest))


if __name__ == "__main__":
    main()
