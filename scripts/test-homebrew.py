#!/usr/bin/env python3
"""Exercise a generated source formula in a disposable tap."""

import functools
import http.server
import json
import os
from pathlib import Path
import re
import signal
import shutil
import subprocess
import sys
import tempfile
import threading


class ArtifactHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def copyfile(self, source, output):
        try:
            super().copyfile(source, output)
        except (BrokenPipeError, ConnectionResetError):
            pass  # Homebrew may close its initial download probe early.


def command(*args, **kwargs):
    timeout = kwargs.pop("timeout", 300)
    with subprocess.Popen(args, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          start_new_session=True, **kwargs) as process:
        try:
            stdout, stderr = process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            stdout, stderr = process.communicate()
            raise RuntimeError(f"{args} timed out: {stderr[-2000:]} {stdout[-2000:]}") from None
        if process.returncode:
            raise RuntimeError(f"{args} failed: {stderr[-2000:]} {stdout[-2000:]}")
        return stdout.strip()


def test(dist):
    subprocess.run(["go", "run", "./tools/release", "verify", str(dist)], check=True)
    command("go", "mod", "download")
    generated = (dist / "bibi.rb").read_text()
    version = re.search(r'^  version "([^"]+)"$', generated, re.M).group(1)
    prefix = Path(command("brew", "--prefix"))
    api_cache = Path(command("brew", "--cache")) / "api"
    module_proxy = (Path(command("go", "env", "GOMODCACHE")) / "cache/download").as_uri()
    binary = prefix / "bin/bibi"
    completions = (prefix / "etc/bash_completion.d/bibi", prefix / "share/zsh/site-functions/_bibi",
                   prefix / "share/fish/vendor_completions.d/bibi.fish")
    if binary.exists() or binary.is_symlink():
        raise RuntimeError(f"Refusing to replace an existing installation at {binary}")
    for completion in completions:
        if completion.exists() or completion.is_symlink():
            raise RuntimeError(f"Refusing to replace an existing completion at {completion}")

    env = dict(os.environ, HOMEBREW_NO_AUTO_UPDATE="1", HOMEBREW_NO_ANALYTICS="1",
               HOMEBREW_NO_INSTALL_CLEANUP="1", HOMEBREW_NO_ASK="1",
               HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK="1")
    # Use normal API dependency resolution instead of cloning all of homebrew/core.
    env.pop("HOMEBREW_NO_INSTALL_FROM_API", None)
    tap_name = "bibi-release/test"
    token = "bibi-release-test"
    installed = f"{tap_name}/{token}"
    if tap_name in command("brew", "tap", env=env).splitlines():
        raise RuntimeError(f"Test tap {tap_name} already exists")

    with tempfile.TemporaryDirectory(prefix="bibi-homebrew-") as directory:
        temp = Path(directory)
        env["HOMEBREW_CACHE"] = str(temp / "cache")
        # Reuse verified dependency metadata; keep downloads and writes disposable.
        if api_cache.is_dir():
            shutil.copytree(api_cache, temp / "cache/api")
        web = temp / "web"
        web.mkdir()
        for archive in dist.glob("*.tar.gz"):
            (web / archive.name).symlink_to(archive.resolve())
        handler = functools.partial(ArtifactHandler, directory=str(web))
        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
        base = f"http://127.0.0.1:{server.server_port}"
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        fixture = generated.replace('class Bibi < Formula', 'class BibiReleaseTest < Formula')
        # Serve dependencies from the cache already used to build the packages.
        # Go still verifies every module against the release's unchanged go.sum.
        fixture = fixture.replace('    ENV["CGO_ENABLED"]',
                                  f'    ENV["GOPROXY"] = {json.dumps(module_proxy)}\n    ENV["CGO_ENABLED"]')
        current = re.sub(r'https://github.com/[^/]+/[^/]+/releases/download/[^/]+/', base + "/", fixture)
        previous = re.sub(r'^  version "[^"]+"', '  version "0.0.0"', current, flags=re.M)

        tap = temp / "tap"
        (tap / "Formula").mkdir(parents=True)
        file = tap / f"Formula/{token}.rb"
        file.write_text(previous)
        command("git", "init", "-b", "main", str(tap))
        command("git", "-C", str(tap), "add", ".")
        command("git", "-C", str(tap), "-c", "user.name=Bibi package tests",
                "-c", "user.email=package-tests@example.invalid", "commit", "-m", "Previous package fixture")
        tapped = False
        try:
            command("brew", "tap", tap_name, tap.as_uri(), env=env)
            tapped = True
            command("brew", "install", "--formula", "--build-from-source", installed, env=env)
            assert command(str(binary), "--version").startswith("bibi v0.0.0"), "Previous version was not installed"
            # Update only this local tap; no public release or repository is changed.
            local_tap = Path(command("brew", "--repository", tap_name, env=env))
            (local_tap / f"Formula/{token}.rb").write_text(current)
            command("git", "-C", str(local_tap), "add", ".")
            command("git", "-C", str(local_tap), "-c", "user.name=Bibi package tests",
                    "-c", "user.email=package-tests@example.invalid", "commit", "-m", "Current package fixture")
            command("brew", "upgrade", "--formula", "--build-from-source", installed, env=env)
            assert command(str(binary), "--version").startswith(f"bibi v{version}"), "Upgrade did not install the new binary"
            assert command(str(binary), "abbr", "inventiones", "mathematicae") == "Invent. Math."
            for name in ("bash", "zsh", "fish"):
                assert "bibi" in command(str(binary), "completion", name)
            for completion in completions:
                assert completion.is_file() and "bibi" in completion.read_text(), f"Completion was not installed: {completion}"
            command("brew", "uninstall", "--formula", "--force", installed, env=env)
            assert not binary.exists() and not binary.is_symlink(), "Uninstall left a binary behind"
            for completion in completions:
                assert not completion.exists() and not completion.is_symlink(), f"Uninstall left a completion behind: {completion}"
        finally:
            try:
                if tapped:
                    try:
                        command("brew", "uninstall", "--formula", "--force", installed, env=env)
                    except RuntimeError:
                        pass  # Already removed by a successful lifecycle check.
                    command("brew", "untap", tap_name, env=env)
            finally:
                server.shutdown()
                server.server_close()
                thread.join()
    print("Homebrew install, upgrade and uninstall passed")


if __name__ == "__main__":
    test(Path(sys.argv[1]).resolve())
