#!/usr/bin/env python3
"""Create a macOS-friendly zip that preserves Unix permissions.

macOS Archive Utility restores a zip entry's permissions when the entry's
"version made by" system is Unix (create_system == 3) and external_attr carries
the Unix mode in its high 16 bits. We set those explicitly, so the .app comes out
with the executable bit already set -- no `chmod` step for the user.

Usage: python make_zip.py <app_dir> <out_zip>
"""
import os
import sys
import time
import zipfile

# Files that must be executable inside the bundle (relative to the .app root).
EXECUTABLES = {
    "Contents/MacOS/VLessBar",
    "Contents/Resources/xray",
}


def mode_for(relpath, is_dir):
    if is_dir:
        return 0o40755
    if relpath.replace("\\", "/") in EXECUTABLES:
        return 0o100755
    return 0o100644


def add_entry(zf, fs_path, arcname, is_dir):
    # arcname is prefixed with the .app folder; EXECUTABLES uses paths inside it.
    rel_in_bundle = arcname.split("/", 1)[1] if "/" in arcname else arcname
    zi = zipfile.ZipInfo(arcname + ("/" if is_dir else ""))
    zi.create_system = 3  # Unix
    zi.external_attr = (mode_for(rel_in_bundle, is_dir) & 0xFFFF) << 16
    if is_dir:
        zi.external_attr |= 0x10  # MS-DOS directory flag (belt and braces)
    zi.compress_type = zipfile.ZIP_DEFLATED
    try:
        zi.date_time = time.localtime(os.stat(fs_path).st_mtime)[:6]
    except OSError:
        pass
    if is_dir:
        zf.writestr(zi, b"")
    else:
        with open(fs_path, "rb") as f:
            zf.writestr(zi, f.read())


def main():
    if len(sys.argv) != 3:
        print("usage: make_zip.py <app_dir> <out_zip>", file=sys.stderr)
        return 2
    app_dir = sys.argv[1].rstrip("\\/")
    out_zip = sys.argv[2]
    base = os.path.dirname(app_dir)

    with zipfile.ZipFile(out_zip, "w", zipfile.ZIP_DEFLATED) as zf:
        for root, _dirs, files in os.walk(app_dir):
            rel = os.path.relpath(root, base).replace("\\", "/")
            add_entry(zf, root, rel, True)
            for fn in files:
                add_entry(zf, os.path.join(root, fn), rel + "/" + fn, False)

    print("wrote", out_zip)
    return 0


if __name__ == "__main__":
    sys.exit(main())
