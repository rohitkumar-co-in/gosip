#!/usr/bin/env python3
"""Remove known local build/cache outputs; preserve runtime data and secrets."""
from pathlib import Path
import shutil

root = Path(__file__).resolve().parent.parent
targets = ["bin", "build", "dist", "frontend/dist", "frontend/node_modules",
           "frontend/.vite", "frontend/vite.config.js", "frontend/vite.config.d.ts",
           "coverage.out", "coverage.html"]
targets += [str(path.relative_to(root)) for path in (root / "frontend").glob("*.tsbuildinfo")]
targets += [str(path.relative_to(root)) for folder in ("scripts", "deploy")
            for path in (root / folder).rglob("__pycache__")]
for name in targets:
    path = root / name
    if path.is_symlink() or not path.resolve().is_relative_to(root):
        raise SystemExit(f"Refusing unsafe cleanup target: {name}")
    if path.is_dir():
        shutil.rmtree(path)
    elif path.exists():
        path.unlink()
    else:
        continue
    print(f"Removed {name}")
