#!/usr/bin/env python3
"""Stage approved M6 runtime ACL refresh on a CLEAN release-v1.0.1 source copy.

Apply to the verified source commit, not to the stale work/m6 prototype nor
to either running Linux bridge. Produces source only; builds/deploys nothing.
"""
import argparse
from pathlib import Path
import shutil
import subprocess
import sys

RELEASE_COMMIT = "96c136702d10b6fb9a16aa2893f6b978e8b14e86"
HELPERS = ("acl_snapshot.go", "acl_snapshot_test.go")


def git(root, *args):
    return subprocess.check_output(["git", "-C", str(root), *args], text=True).strip()


def replace_one(text, before, after, label):
    if text.count(before) != 1:
        raise ValueError(f"release source mismatch: {label}")
    return text.replace(before, after)


def patch_bridge(raw):
    raw = replace_one(
        raw,
        "\tAuthorizedServices map[[32]byte]map[string]bool\n",
        "\tAuthorizedServices map[[32]byte]map[string]bool\n"
        "\t// Optional immutable ACL lookup for M6; legacy server behavior unchanged.\n"
        "\tAuthorizationLookup func([32]byte, string) bool\n",
        "ACL lookup field",
    )
    raw = replace_one(
        raw,
        "\tif !s.AuthorizedServices[fingerprint][service] {\n",
        "\tallowed := s.AuthorizedServices[fingerprint][service]\n"
        "\tif s.AuthorizationLookup != nil {\n"
        "\t\tallowed = s.AuthorizationLookup(fingerprint, service)\n"
        "\t}\n"
        "\tif !allowed {\n",
        "ACL admission",
    )
    return raw


def patch_main(raw):
    raw = replace_one(
        raw,
        '\t"net/url"\n',
        '\t"net/url"\n\t"reflect"\n',
        "reflect import",
    )
    anchor = "\thandler := &bridge.Server{Routes: c.Routes, AuthorizedServices: authorized, MaxConcurrent: c.MaxStreams, Metrics: metrics}\n"
    replacement = """\t// M6: publish immutable ACL snapshots; keep the already released streaming
\t// code, routes, mTLS and padding/jitter behavior unchanged.
\tacl, err := bridge.NewRuntimeACL(c.Authorized, c.Routes)
\tif err != nil { log.Fatal(err) }
\t""" + anchor.lstrip("\t").replace(
        "Metrics: metrics}", "Metrics: metrics, AuthorizationLookup: acl.Allowed}"
    ) + """\t// SIGHUP only reloads the certificate ACL. A changed backend, key,
\t// listener, CA, padding, jitter or any other field is rejected.
\t// No restart or active-session teardown occurs for valid updates.
\thup := make(chan os.Signal, 1)
\tsignal.Notify(hup, syscall.SIGHUP)
\tdefer signal.Stop(hup)
\tgo func() {
\t\tfor {
\t\t\tselect {
\t\t\tcase <-ctx.Done():
\t\t\t\treturn
\t\t\tcase <-hup:
\t\t\t\tfresh, e := load(*path)
\t\t\t\tif e != nil {
\t\t\t\t\tlog.Printf("M6_ACL_RELOAD=FAIL invalid config: %v", e)
\t\t\t\t\tcontinue
\t\t\t\t}
\t\t\t\toldNonACL, newNonACL := c, fresh
\t\t\t\toldNonACL.Authorized, newNonACL.Authorized = nil, nil
\t\t\t\tif !reflect.DeepEqual(oldNonACL, newNonACL) {
\t\t\t\t\tlog.Print("M6_ACL_RELOAD=FAIL non-ACL config changed")
\t\t\t\t\tcontinue
\t\t\t\t}
\t\t\t\tif e := acl.Replace(fresh.Authorized, c.Routes); e != nil {
\t\t\t\t\tlog.Printf("M6_ACL_RELOAD=FAIL rejected ACL: %v", e)
\t\t\t\t\tcontinue
\t\t\t\t}
\t\t\t\taclJSON, _ := json.Marshal(fresh.Authorized)
\t\t\t\taclDigest := sha256.Sum256(aclJSON)
\t\t\t\tlog.Printf("M6_ACL_RELOAD=PASS digest=%x", aclDigest)
\t\t\t}
\t\t}
\t}()
"""
    return replace_one(raw, anchor, replacement, "server ACL setup and HUP reload")


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--release-root", type=Path, required=True)
    mode = p.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    a = p.parse_args()
    root = a.release_root.resolve()
    if git(root, "rev-parse", "HEAD") != RELEASE_COMMIT:
        raise ValueError("source HEAD does not match published v1.0.1 build commit")
    if git(root, "status", "--porcelain"):
        raise ValueError("release source checkout is not clean")
    go = root / "prototype"
    if not (go / "go.mod").is_file():
        raise ValueError("release source Go module is missing")
    current = {
        go / "bridge.go": patch_bridge((go / "bridge.go").read_text()),
        go / "cmd/bridge/main.go": patch_main((go / "cmd/bridge/main.go").read_text()),
    }
    for name in HELPERS:
        target = go / name
        if target.exists() or target.is_symlink():
            raise ValueError("refusing to overwrite existing source file")
        source = Path(__file__).resolve().parent / name
        if not source.is_file():
            raise ValueError("M6 snapshot source missing")
        current[target] = source.read_text()
    print("M6_RUNTIME_ACL_RELEASE_SOURCE=MATCH")
    if a.check:
        print("M6_RUNTIME_ACL_SOURCE_CHANGED=NO")
        return
    for path, content in current.items():
        path.write_text(content)
    print("M6_RUNTIME_ACL_SOURCE=STAGED")
    print("M6_BINARIES_AND_RUNNING_SERVICES=UNCHANGED")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print("M6_RUNTIME_ACL=FAIL: " + str(exc), file=sys.stderr)
        sys.exit(1)
