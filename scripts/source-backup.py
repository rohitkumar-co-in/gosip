#!/usr/bin/env python3
"""Archive the committed Git source; omit untracked runtime/build files."""
import argparse
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath
import subprocess
import zipfile

root = Path(__file__).resolve().parent.parent
def git(*args):
    return subprocess.check_output(["git", "-C", str(root), *args])

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--output", type=Path, help="ZIP path outside this checkout")
args = parser.parse_args()
if git("status", "--porcelain", "--untracked-files=normal").strip():
    raise SystemExit("Commit or preserve pending changes before exporting the source archive.")
commit = git("rev-parse", "HEAD").decode().strip()
stamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
output = (args.output or root.parent / f"leadomi-sip-source-{stamp}-{commit[:8]}.zip").resolve()
if output.is_relative_to(root) or output.exists():
    raise SystemExit("Choose a new output path outside the checkout.")
tracked = git("ls-files", "-z").decode().split("\0")
for name in filter(None, tracked):
    path = PurePosixPath(name)
    if (any(part in {"node_modules", "dist", "build", "bin", "tmp", "data", "backups", "__pycache__", "secrets"} for part in path.parts)
            or path.suffix in {".pem", ".key", ".db", ".exe", ".pyc", ".tsbuildinfo"}
            or (path.name.startswith(".env") and path.name != ".env.example")):
        raise SystemExit(f"Refusing private/generated tracked file: {name}")
output.parent.mkdir(parents=True, exist_ok=True)
try:
    subprocess.run(["git", "-C", str(root), "archive", "--format=zip",
                    "--prefix=leadomi-sip/", "--output", str(output), commit], check=True)
    with zipfile.ZipFile(output) as archive:
        bad = archive.testzip()
        if bad:
            raise RuntimeError(f"Invalid archive member: {bad}")
        files = sum(not member.is_dir() for member in archive.infolist())
except BaseException:
    output.unlink(missing_ok=True)
    raise
print(f"Source revision: {commit}\nFiles: {files}\nArchive: {output}")
