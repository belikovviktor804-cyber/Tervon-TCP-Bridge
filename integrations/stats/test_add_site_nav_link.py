"""Test that navigation insertion only touches the existing nav, no whole-page rewrite."""
import importlib.util
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("site_nav", ROOT / "add-site-nav-link.py")
site_nav = importlib.util.module_from_spec(spec)
spec.loader.exec_module(site_nav)


class NavTest(unittest.TestCase):
    def test_insert_once_preserves_page(self):
        original = """<html><nav class="site-nav" aria-label="Навигация"><a href="/admin/">Admin</a></nav><main>content</main></html>"""
        result, status = site_nav.updated(original)
        self.assertEqual(status, "READY")
        self.assertEqual(result.count('href="/bridge/"'), 1)
        self.assertEqual(result.replace(site_nav.LINK + "\n", ""), original)
        again, status = site_nav.updated(result)
        self.assertEqual(status, "ALREADY_PRESENT")
        self.assertEqual(again, result)

    def test_no_nav_fails_closed(self):
        self.assertEqual(site_nav.updated("<html>no nav</html>")[1], "BLOCKED_NO_UNIQUE_SITE_NAV")

    def test_multiple_navs_fail_closed(self):
        self.assertEqual(site_nav.updated('<nav class="site-nav"></nav><nav class="site-nav"></nav>')[1],
                         "BLOCKED_NO_UNIQUE_SITE_NAV")

    def test_ignores_other_navigation(self):
        self.assertEqual(site_nav.updated('<nav class="other"></nav>')[1], "BLOCKED_NO_UNIQUE_SITE_NAV")

    def test_existing_bridge_not_duplicated(self):
        html = '<nav class="site-nav"><a href="/bridge">Bridge</a></nav>'
        self.assertEqual(site_nav.updated(html)[1], "ALREADY_PRESENT")


if __name__ == "__main__":
    unittest.main()
