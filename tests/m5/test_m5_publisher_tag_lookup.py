#!/usr/bin/env python3
"""Regress the GitHub CLI 404 error-body handling without network or assets."""
import pathlib
import re
import subprocess
import unittest

PUBLISHER = pathlib.Path(__file__).resolve().parents[2] / "install/publish-v1.0.1.sh"
HEAD = "a" * 40
OTHER = "b" * 40

def tag_logic():
    script = PUBLISHER.read_text(encoding="utf-8")
    start = script.index('if REMOTE_TAG="$(gh api')
    end = script.index('echo "M5_PUBLISH_TAG_SHA=$HEAD_SHA"', start)
    return script[start:end]

def check(scenario):
    command = """set -Eeuo pipefail
REPO=tervon-software/Tervon-TCP-Bridge
TAG=v1.0.1
HEAD_SHA=%s
gh() {
  case "$1 $2" in
    "api repos/tervon-software/Tervon-TCP-Bridge/git/ref/tags/v1.0.1")
      case "%s" in
        missing) printf '??? GitHub 404 body ???\\n'; return 1 ;;
        current) printf '%s\\n'; return 0 ;;
        other) printf '%s\\n'; return 0 ;;
        malformed) printf 'not-a-sha\\n'; return 0 ;;
      esac
      ;;
    "api --method")
      printf 'CREATED_TAG\\n'
      return 0 ;;
  esac
  printf 'UNEXPECTED_GH_CALL\\n' >&2
  return 30
}
""" % (HEAD, scenario, HEAD, OTHER) + tag_logic()
    return subprocess.run(["bash", "-c", command], text=True, capture_output=True)

class PublisherTagLookup(unittest.TestCase):
    def test_http_404_body_is_not_misread_as_tag(self):
        r=check("missing")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("CREATED_TAG",r.stdout)
        self.assertNotIn("BLOCKED_TAG_POINTS_TO_OTHER_COMMIT",r.stdout)

    def test_matching_tag_reused_without_post(self):
        r=check("current")
        self.assertEqual(r.returncode,0,r.stderr)
        self.assertNotIn("CREATED_TAG",r.stdout)

    def test_different_tag_is_not_overwritten(self):
        r=check("other")
        self.assertEqual(r.returncode,3,r.stderr)
        self.assertIn("BLOCKED_TAG_POINTS_TO_OTHER_COMMIT",r.stdout)
        self.assertNotIn("CREATED_TAG",r.stdout)

    def test_malformed_success_response_fails_closed(self):
        r=check("malformed")
        self.assertEqual(r.returncode,3,r.stderr)
        self.assertIn("BLOCKED_UNEXPECTED_TAG_RESPONSE",r.stdout)
        self.assertNotIn("CREATED_TAG",r.stdout)

if __name__=="__main__":
    unittest.main()
