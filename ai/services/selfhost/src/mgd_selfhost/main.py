"""Server entrypoint: uvicorn mgd_selfhost.main:app"""

from __future__ import annotations

import uvicorn

from mgd_selfhost.app import create_app
from mgd_selfhost.config import Settings

app = create_app()


def main() -> None:
    cfg = Settings.from_env()
    uvicorn.run(app, host=cfg.host, port=cfg.port)


if __name__ == "__main__":
    main()
