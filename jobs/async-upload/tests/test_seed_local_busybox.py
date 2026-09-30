"""Checks the CI helper that makes BusyBox available to async-upload pods."""

import os
import subprocess
from pathlib import Path

import pytest


SCRIPT = Path(__file__).resolve().parents[3] / "scripts" / "seed_local_busybox.sh"
MAKE_DIR = Path(__file__).resolve().parents[1]


@pytest.fixture
def registry_commands(tmp_path):
    archive = tmp_path / "busybox-image.tar"
    archive.write_bytes(b"cached image archive")
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    docker = bin_dir / "docker"
    docker.write_text(
        "#!/bin/sh\n"
        'printf "%s\\n" "$*" >> "$COMMAND_LOG"\n'
        'if [ "$1" = push ]; then\n'
        '  [ "${FAIL_PUSH:-}" != 1 ] || exit 1\n'
        '  touch "$PUSHED_IMAGE"\n'
        'fi\n'
    )
    docker.chmod(0o755)
    curl = bin_dir / "curl"
    curl.write_text(
        "#!/bin/sh\n"
        'case "$*" in\n'
        '  */v2/busybox/manifests/latest*) [ -f "$PUSHED_IMAGE" ] ;;\n'
        '  */v2/*) exit 0 ;;\n'
        '  *) exit 1 ;;\n'
        'esac\n'
    )
    curl.chmod(0o755)
    command_log = tmp_path / "commands"
    env = {
        "PATH": f"{bin_dir}:{os.environ['PATH']}",
        "COMMAND_LOG": str(command_log),
        "PUSHED_IMAGE": str(tmp_path / "pushed-image"),
    }
    return archive, command_log, env


def test_seed_local_busybox_publishes_cached_image(registry_commands):
    archive, command_log, env = registry_commands

    result = subprocess.run(["bash", str(SCRIPT), str(archive)], env=env, capture_output=True, text=True)

    assert result.returncode == 0, result.stderr
    assert command_log.read_text().splitlines() == [
        f"load -i {archive}",
        "tag public.ecr.aws/docker/library/busybox:latest localhost:5001/busybox:latest",
        "push localhost:5001/busybox:latest",
    ]
    assert Path(env["PUSHED_IMAGE"]).exists()


def test_seed_local_busybox_fails_when_registry_push_fails(registry_commands):
    archive, command_log, env = registry_commands
    env["FAIL_PUSH"] = "1"

    result = subprocess.run(["bash", str(SCRIPT), str(archive)], env=env, capture_output=True, text=True)

    assert result.returncode != 0
    assert command_log.read_text().splitlines()[-1] == "push localhost:5001/busybox:latest"
    assert not Path(env["PUSHED_IMAGE"]).exists()


def test_make_seed_local_busybox_uses_ci_archive(registry_commands):
    archive, command_log, env = registry_commands
    env["BUSYBOX_IMAGE_ARCHIVE"] = str(archive)

    result = subprocess.run(
        ["make", "-C", str(MAKE_DIR), "seed-local-busybox"], env=env, capture_output=True, text=True
    )

    assert result.returncode == 0, result.stderr
    assert command_log.read_text().splitlines()[-1] == "push localhost:5001/busybox:latest"
