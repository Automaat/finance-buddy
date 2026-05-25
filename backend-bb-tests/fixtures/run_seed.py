"""Dev seeder entrypoint.

Applies the bb-tests fixture seed against the dev database, but only when the
accounts table is empty. Lets `mise dev` boot a populated app on a fresh
volume without clobbering data added through the UI on subsequent restarts.
"""

from __future__ import annotations

import os
import sys

import psycopg2

from seed import seed


def main() -> int:
    dsn = os.environ["DATABASE_URL"]
    with psycopg2.connect(dsn) as conn, conn.cursor() as cur:
        cur.execute("SELECT 1 FROM accounts LIMIT 1")
        if cur.fetchone() is not None:
            print("dev seed: accounts present, skipping")
            return 0
    print("dev seed: applying fixture data")
    seed(dsn)
    print("dev seed: done")
    return 0


if __name__ == "__main__":
    sys.exit(main())
