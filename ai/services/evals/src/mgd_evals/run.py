"""CLI 入口（python -m mgd_evals.run）。"""

from __future__ import annotations

import sys

from .runner import main

if __name__ == "__main__":
    sys.exit(main())
