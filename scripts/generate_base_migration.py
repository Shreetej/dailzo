#!/usr/bin/env python3
"""Regenerate migrations/000_base_schema.sql from the pgAdmin ERD export.

The ERD script is the source of truth for the core schema, but it is not
usable as a migration unchanged: it opens its own transaction (cmd/migrate
already provides one, and the script's END would commit it early) and its
ADD CONSTRAINT statements fail on a database that already has them.

Run from the repo root:  python3 scripts/generate_base_migration.py
"""
import re
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent.parent
SOURCE = ROOT / "DatabaseScripts&ERD" / "DailzoScript.sql"
TARGET = ROOT / "migrations" / "000_base_schema.sql"

HEADER = """-- Migration: 000_base_schema.sql
-- Description: The core schema every later migration builds on — users,
--              restaurants, food_products, orders, payments and friends.
--
-- This lived only in DatabaseScripts&ERD/DailzoScript.sql and was applied by
-- hand to each developer's database, so a brand-new database (a managed
-- Postgres on first deploy) had no tables at all and 001_new_tables.sql died
-- on its first foreign key: relation "public.users" does not exist.
--
-- Generated from that script, with two changes so it is safe as a migration:
--   * no outer BEGIN/END — cmd/migrate already runs each file in one
--     transaction, and the script's own END would have committed it early;
--   * every ADD CONSTRAINT ignores "already exists", so this applies cleanly
--     to a database that was set up by hand before migrations existed.
--
-- Regenerate with scripts/generate_base_migration.py if the ERD script changes.

"""


def wrap_constraint(match: re.Match) -> str:
    """Make one ALTER TABLE ... ADD CONSTRAINT tolerant of already existing."""
    statement = match.group(1)
    indented = "\n".join(
        "    " + line if line.strip() else line for line in statement.splitlines()
    )
    return (
        "DO $$ BEGIN\n"
        f"{indented}\n"
        "EXCEPTION WHEN duplicate_object THEN NULL;\n"
        "END $$;"
    )


def main() -> None:
    body = SOURCE.read_text(encoding="utf-8")

    # The ERD tool's own preamble; HEADER explains the file instead.
    body = re.sub(r"\A(--[^\n]*\n)+\n?", "", body)
    # The outer transaction belongs to cmd/migrate.
    body = re.sub(r"(?m)^BEGIN;\s*$", "", body, count=1)
    body = re.sub(r"(?s)\nEND;\s*\Z", "\n", body)

    body, count = re.subn(
        r"(?ms)^(ALTER TABLE IF EXISTS\s+.*?;)\s*$", wrap_constraint, body
    )

    TARGET.write_text(HEADER + body.strip() + "\n", encoding="utf-8")
    print(f"wrote {TARGET.relative_to(ROOT)} ({count} constraints made idempotent)")


if __name__ == "__main__":
    main()
