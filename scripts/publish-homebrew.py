#!/usr/bin/env python3
"""Publish an already validated stable formula with a tap-scoped SSH deploy key."""

import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import tempfile


def validate(path, tag, tap):
    if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+", tag):
        raise ValueError("Homebrew publication requires a stable version tag")
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", tap):
        raise ValueError("Expected an owner/repository tap name")
    formula = path.read_text()
    if f'  version "{tag[1:]}"' not in formula or 'class Bibi < Formula' not in formula:
        raise ValueError("Formula does not match the requested release")
    return formula


def check_update(previous, current, tag):
    if previous == current:
        return False
    old_version = re.search(r'^  version "(\d+)\.(\d+)\.(\d+)"$', previous, re.M)
    if old_version and tuple(map(int, old_version.groups())) > tuple(map(int, tag[1:].split("."))):
        raise ValueError("Refusing to downgrade the tap to an older release")
    return True


def write_deploy_key(path, value):
    # Clipboard copies may omit the final newline or use Windows line endings.
    # OpenSSH requires a complete key with Unix line endings.
    path.touch(mode=0o600)
    path.write_text(value.replace("\r\n", "\n").strip() + "\n")
    result = subprocess.run(["ssh-keygen", "-y", "-P", "", "-f", str(path)],
                            stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                            stderr=subprocess.PIPE, timeout=10)
    if result.returncode:
        raise ValueError("HOMEBREW_TAP_SSH_KEY must contain the complete, unencrypted private key")


def publish(path, tag, tap):
    formula = validate(path, tag, tap)
    private_key = os.environ.get("HOMEBREW_TAP_SSH_KEY", "")
    if not private_key:
        raise ValueError("Configure HOMEBREW_TAP_SSH_KEY with a write-enabled deploy key for the tap")
    with tempfile.TemporaryDirectory(prefix="bibi-publish-") as directory:
        temp = Path(directory)
        key = temp / "deploy-key"
        write_deploy_key(key, private_key)
        repository = temp / "tap"
        subprocess.run(["git", "clone", "--depth=1", f"https://github.com/{tap}.git", str(repository)], check=True)
        destination = repository / "Formula/bibi.rb"
        previous = destination.read_text() if destination.exists() else ""
        if not check_update(previous, formula, tag):
            print(f"{tap} already contains bibi {tag}")
            return
        branch = subprocess.check_output(["git", "-C", str(repository), "symbolic-ref", "--short", "HEAD"], text=True).strip()
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(formula)
        subprocess.run(["git", "-C", str(repository), "add", "Formula/bibi.rb"], check=True)
        subprocess.run(["git", "-C", str(repository), "-c", "user.name=github-actions[bot]",
                        "-c", "user.email=41898282+github-actions[bot]@users.noreply.github.com",
                        "commit", "-m", f"Update bibi to {tag}"], check=True)
        # Read GitHub's official SSH host keys over authenticated HTTPS.
        metadata = json.loads(subprocess.check_output(["gh", "api", "meta"], text=True))
        hosts = temp / "known_hosts"
        hosts.write_text("".join(f"github.com {host_key}\n" for host_key in metadata["ssh_keys"]))
        ssh = shlex.join(["ssh", "-i", str(key), "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=yes",
                          "-o", f"UserKnownHostsFile={hosts}"])
        # A regular fast-forward push rejects concurrent updates; never force it.
        subprocess.run(["git", "-C", str(repository), "-c", f"core.sshCommand={ssh}",
                        "push", f"git@github.com:{tap}.git", f"HEAD:{branch}"], check=True)
    print(f"Published bibi {tag} to {tap}")


if __name__ == "__main__":
    publish(Path(sys.argv[1]), sys.argv[2], sys.argv[3])
