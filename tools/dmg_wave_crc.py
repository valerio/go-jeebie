#!/usr/bin/env python3
"""Run the DMG wave trigger ROM headless and compute CRCs of logged wave RAM dumps."""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
import zlib


DEFAULT_ROM = "test-roms/game-boy-test-roms/blargg/dmg_sound/rom_singles/10-wave trigger while on.gb"
DEFAULT_BINARY = "./bin/jeebie"
DEFAULT_FRAMES = 400
WAVE_ADDR_START = 0xFF30
WAVE_ADDR_END = 0xFF3F
WAVE_READ_RE = re.compile(
    r"apu\.ch3 wave read.*addr=0x([0-9A-Fa-f]{4}).*result=0x([0-9A-Fa-f]{2})"
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Run the emulator in headless mode, capture CH3 wave RAM reads from the log, "
            "and report CRC32 values for each complete dump."
        )
    )
    parser.add_argument(
        "rom",
        nargs="?",
        default=DEFAULT_ROM,
        help=(
            "ROM to execute (defaults to blargg dmg_sound test 10 relative to the repo root)."
        ),
    )
    parser.add_argument(
        "--binary",
        default=DEFAULT_BINARY,
        help="Path to the jeebie executable (defaults to ./bin/jeebie).",
    )
    parser.add_argument(
        "--frames",
        type=int,
        default=DEFAULT_FRAMES,
        help=f"Number of frames to run in headless mode (default: {DEFAULT_FRAMES}).",
    )
    parser.add_argument(
        "--keep-log",
        metavar="PATH",
        help="Optional path to write the full emulator log output.",
    )
    return parser.parse_args()


def run_emulator(binary: str, rom: str, frames: int) -> str:
    if not os.path.exists(binary):
        raise FileNotFoundError(f"emulator binary not found: {binary}")
    if not os.path.exists(rom):
        raise FileNotFoundError(f"ROM not found: {rom}")

    cmd = [binary, "--headless", f"--frames={frames}", "--debug", rom]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    output = (proc.stdout or "") + (proc.stderr or "")
    if proc.returncode != 0:
        raise RuntimeError(
            f"emulator exited with status {proc.returncode}\n\n{output}".rstrip()
        )
    return output


def collect_wave_dumps(log_data: str) -> list[bytes]:
    dumps: list[bytes] = []
    current: list[int] = []
    expected_addr = WAVE_ADDR_START

    for line in log_data.splitlines():
        match = WAVE_READ_RE.search(line)
        if not match:
            continue
        addr = int(match.group(1), 16)
        if not (WAVE_ADDR_START <= addr <= WAVE_ADDR_END):
            continue
        value = int(match.group(2), 16)

        if addr == WAVE_ADDR_START and current:
            if len(current) == 16:
                dumps.append(bytes(current))
            current = []
            expected_addr = WAVE_ADDR_START

        if addr != expected_addr:
            current = []
            expected_addr = WAVE_ADDR_START
            if addr != expected_addr:
                continue

        current.append(value)
        expected_addr += 1
        if expected_addr > WAVE_ADDR_END:
            expected_addr = WAVE_ADDR_START

        if len(current) == 16:
            dumps.append(bytes(current))
            current = []
            expected_addr = WAVE_ADDR_START

    if current and len(current) == 16:
        dumps.append(bytes(current))

    return dumps


def format_dump(data: bytes) -> str:
    return " ".join(f"{b:02X}" for b in data)


def main() -> None:
    args = parse_args()

    # Always build the binary before running the helper.
    print("Rebuilding binary via `make build`...", file=sys.stderr)
    build_proc = subprocess.run(["make", "build"], capture_output=True, text=True)
    if build_proc.returncode != 0:
        print(
            "`make build` failed:\n" + (build_proc.stdout or "") + (build_proc.stderr or ""),
            file=sys.stderr,
        )
        sys.exit(1)

    log_data = run_emulator(args.binary, args.rom, args.frames)

    if args.keep_log:
        with open(args.keep_log, "w", encoding="utf-8") as fh:
            fh.write(log_data)

    dumps = collect_wave_dumps(log_data)
    if not dumps:
        print("No complete wave RAM dumps found in log output.", file=sys.stderr)
        sys.exit(1)

    stream = b"".join(dumps)
    stream_crc = zlib.crc32(stream) & 0xFFFFFFFF
    print(
        f"Captured {len(dumps)} wave RAM dumps from {args.rom} "
        f"({len(stream)} bytes total)."
    )
    for dump in dumps:
        print(format_dump(dump))

    print(f"Final CRC32: 0x{stream_crc:08X} ({len(stream)} bytes)")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(130)
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(1)
