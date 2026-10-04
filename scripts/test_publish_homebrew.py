import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("publish_homebrew", Path(__file__).with_name("publish-homebrew.py"))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


class PublishHomebrewTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.formula = Path(self.directory.name) / "bibi.rb"
        self.current = 'class Bibi < Formula\n  version "0.5.0"\nend\n'
        self.formula.write_text(self.current)

    def test_invalid_release_never_calls_github(self):
        with patch.object(publisher.subprocess, "run") as call:
            with self.assertRaises(ValueError):
                publisher.publish(self.formula, "v0.5.0-rc1", "thofma/homebrew-tap")
            with self.assertRaises(ValueError):
                publisher.publish(self.formula, "v0.6.0", "thofma/homebrew-tap")
            call.assert_not_called()

    def test_already_published_formula_is_not_written_again(self):
        self.assertFalse(publisher.check_update(self.current, self.current, "v0.5.0"))

    def test_out_of_order_release_cannot_downgrade_tap(self):
        newer = self.current.replace("0.5.0", "0.6.0")
        with self.assertRaises(ValueError):
            publisher.check_update(newer, self.current, "v0.5.0")

    def test_copied_keys_can_be_read_by_openssh(self):
        source = Path(self.directory.name) / "source-key"
        subprocess.run(["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", str(source)], check=True)
        original = source.read_text()
        destination = Path(self.directory.name) / "deploy-key"
        for value in (original.rstrip("\n"), original.replace("\n", "\r\n")):
            with self.subTest(line_endings="CRLF" if "\r" in value else "missing final newline"):
                publisher.write_deploy_key(destination, value)
                self.assertEqual(destination.read_text(), original)
                self.assertEqual(destination.stat().st_mode & 0o777, 0o600)

    def test_invalid_key_fails_before_contacting_github(self):
        run = subprocess.run

        def local_validation(args, **kwargs):
            self.assertEqual(args[0], "ssh-keygen", "Invalid keys must fail before any network request")
            return run(args, **kwargs)

        with patch.dict(os.environ, {"HOMEBREW_TAP_SSH_KEY": "not a private key"}), \
                patch.object(publisher.subprocess, "run", side_effect=local_validation), \
                patch.object(publisher.subprocess, "check_output") as call:
            with self.assertRaisesRegex(ValueError, "complete, unencrypted private key"):
                publisher.publish(self.formula, "v0.5.0", "thofma/homebrew-tap")
            call.assert_not_called()

    def test_update_pushes_to_default_branch_without_force(self):
        pushes = []

        def git(args, **kwargs):
            if "clone" in args:
                repo = Path(args[-1])
                (repo / "Formula").mkdir(parents=True)
                (repo / "Formula/bibi.rb").write_text(self.current.replace("0.5.0", "0.4.0"))
            if "commit" in args:
                self.assertEqual((Path(args[2]) / "Formula/bibi.rb").read_text(), self.current)
            if "push" in args:
                self.assertEqual(args[-2:], ["git@github.com:thofma/homebrew-tap.git", "HEAD:main"])
                self.assertNotIn("--force", args)
                pushes.append(args)
            return subprocess.CompletedProcess(args, 0)

        def metadata(args, **kwargs):
            return '{"ssh_keys":["ssh-ed25519 fixture"]}' if args[0] == "gh" else "main\n"

        with patch.dict(os.environ, {"HOMEBREW_TAP_SSH_KEY": "fixture-key"}), \
                patch.object(publisher.subprocess, "check_output", side_effect=metadata), \
                patch.object(publisher.subprocess, "run", side_effect=git):
            publisher.publish(self.formula, "v0.5.0", "thofma/homebrew-tap")
        self.assertEqual(len(pushes), 1)


if __name__ == "__main__":
    unittest.main()
