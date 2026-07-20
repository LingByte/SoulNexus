"""Multi-dialect DB connection for the voiceprint service.

Supports the same drivers as LingEchoX:
  - sqlite   (default for local ./lingecho.db)
  - postgres / pgsql / postgresql
  - mysql

Config resolution order:
  1. Env DB_DRIVER + DSN (same as LingEchoX)
  2. YAML `database.driver` + `database.dsn`
  3. Legacy YAML `mysql:` block → mysql DSN
"""

from __future__ import annotations

import os
import re
import sqlite3
from contextlib import contextmanager
from typing import Any, Iterator, Optional, Tuple
from urllib.parse import quote_plus, urlparse

from ..core.config import settings
from ..core.logger import get_logger

logger = get_logger(__name__)

_PLACEHOLDER_RE = re.compile(r"%s")


def _normalize_driver(raw: str) -> str:
    d = (raw or "").strip().lower()
    if d in ("", "sqlite", "sqlite3"):
        return "sqlite"
    if d in ("postgres", "postgresql", "pgsql", "pg"):
        return "postgres"
    if d in ("mysql", "mariadb"):
        return "mysql"
    raise ValueError(f"unsupported DB driver: {raw!r} (want sqlite|postgres|mysql)")


def resolve_database_config() -> Tuple[str, str]:
    """Return (driver, dsn) for the shared LingEchoX database."""
    env_driver = os.getenv("DB_DRIVER", "").strip()
    env_dsn = os.getenv("DSN", "").strip()
    if env_driver or env_dsn:
        driver = _normalize_driver(env_driver or "sqlite")
        dsn = env_dsn or "./lingecho.db"
        return driver, dsn

    db_cfg = settings.database or {}
    if db_cfg.get("driver") or db_cfg.get("dsn"):
        driver = _normalize_driver(str(db_cfg.get("driver") or "sqlite"))
        dsn = str(db_cfg.get("dsn") or "./lingecho.db").strip()
        return driver, dsn

    # Legacy mysql: block
    mysql = settings.mysql or {}
    if mysql.get("host") or mysql.get("database"):
        user = quote_plus(str(mysql.get("user") or ""))
        password = quote_plus(str(mysql.get("password") or ""))
        host = str(mysql.get("host") or "127.0.0.1")
        port = int(mysql.get("port") or 3306)
        database = str(mysql.get("database") or "")
        auth = f"{user}:{password}@" if user or password else ""
        return "mysql", f"{auth}{host}:{port}/{database}?charset=utf8mb4"

    # Sensible local default matching LingEchoX sqlite
    return "sqlite", "./lingecho.db"


class DatabaseConnection:
    """Dialect-aware connection with %s placeholders for all drivers."""

    def __init__(self) -> None:
        self.driver, self.dsn = resolve_database_config()
        self._conn: Any = None
        self._connect()

    def _connect(self) -> None:
        try:
            if self.driver == "sqlite":
                path = self.dsn
                if path.startswith("file:"):
                    # glebarez/sqlite style: file:./foo.db?cache=shared
                    path = path[len("file:") :].split("?", 1)[0]
                # Ensure parent dir exists for file DSN
                parent = os.path.dirname(os.path.abspath(path))
                if parent and parent != os.path.abspath(path):
                    os.makedirs(parent, exist_ok=True)
                self._conn = sqlite3.connect(path, check_same_thread=False, timeout=30)
                self._conn.row_factory = sqlite3.Row
                self._conn.execute("PRAGMA journal_mode=WAL;")
                self._conn.execute("PRAGMA foreign_keys=ON;")
            elif self.driver == "postgres":
                import psycopg2

                self._conn = psycopg2.connect(self.dsn)
                self._conn.autocommit = True
            else:
                import pymysql

                # Accept either URL, user:pass@host:port/db, or key=value is not used for mysql
                dsn = self.dsn
                if "://" in dsn:
                    u = urlparse(dsn)
                    self._conn = pymysql.connect(
                        host=u.hostname or "127.0.0.1",
                        port=u.port or 3306,
                        user=u.username or "",
                        password=u.password or "",
                        database=(u.path or "/").lstrip("/"),
                        charset="utf8mb4",
                        autocommit=True,
                        connect_timeout=10,
                        read_timeout=30,
                        write_timeout=30,
                    )
                elif "@" in dsn or "/" in dsn:
                    # user:pass@host:port/db?charset=utf8mb4
                    auth_host, _, db_part = dsn.partition("/")
                    database, _, _query = db_part.partition("?")
                    if "@" in auth_host:
                        auth, _, hostport = auth_host.rpartition("@")
                        user, _, password = auth.partition(":")
                    else:
                        user, password, hostport = "", "", auth_host
                    host, _, port_s = hostport.partition(":")
                    port = int(port_s) if port_s else 3306
                    self._conn = pymysql.connect(
                        host=host or "127.0.0.1",
                        port=port,
                        user=user,
                        password=password,
                        database=database,
                        charset="utf8mb4",
                        autocommit=True,
                        connect_timeout=10,
                        read_timeout=30,
                        write_timeout=30,
                    )
                else:
                    raise ValueError(f"invalid mysql DSN: {dsn!r}")

            logger.success(f"数据库连接成功 driver={self.driver}")
        except Exception as e:
            logger.fail(f"数据库连接失败 driver={self.driver}: {e}")
            raise

    def _alive(self) -> bool:
        if self._conn is None:
            return False
        if self.driver == "sqlite":
            try:
                self._conn.execute("SELECT 1")
                return True
            except Exception:
                return False
        if self.driver == "postgres":
            return getattr(self._conn, "closed", 1) == 0
        return bool(getattr(self._conn, "open", False))

    def adapt_sql(self, sql: str) -> str:
        """Convert %s placeholders to dialect form (sqlite uses ?)."""
        if self.driver == "sqlite":
            return _PLACEHOLDER_RE.sub("?", sql)
        return sql

    @contextmanager
    def get_cursor(self) -> Iterator[Any]:
        if not self._alive():
            self._connect()

        cursor = None
        try:
            cursor = self._conn.cursor()
            yield _CursorAdapter(self, cursor)
            if self.driver == "sqlite":
                self._conn.commit()
        except Exception as e:
            logger.fail(f"数据库操作失败: {e}")
            try:
                if self._conn is not None:
                    self._conn.rollback()
            except Exception:
                pass
            raise
        finally:
            if cursor is not None:
                try:
                    cursor.close()
                except Exception:
                    pass

    def close(self) -> None:
        if self._conn is None:
            return
        try:
            self._conn.close()
            logger.info("数据库连接已关闭")
        except Exception:
            pass
        self._conn = None

    def __del__(self) -> None:
        try:
            self.close()
        except Exception:
            pass


class _CursorAdapter:
    """Wraps a DB-API cursor so callers can always use %s placeholders."""

    def __init__(self, db: DatabaseConnection, cursor: Any) -> None:
        self._db = db
        self._cursor = cursor

    @property
    def rowcount(self) -> int:
        return int(getattr(self._cursor, "rowcount", 0) or 0)

    def execute(self, sql: str, params: Any = ()) -> Any:
        return self._cursor.execute(self._db.adapt_sql(sql), params)

    def fetchone(self) -> Any:
        return self._cursor.fetchone()

    def fetchall(self) -> Any:
        return self._cursor.fetchall()


# Lazy singleton — imported modules may construct Settings first.
db_connection: Optional[DatabaseConnection] = None


def get_db() -> DatabaseConnection:
    global db_connection
    if db_connection is None:
        db_connection = DatabaseConnection()
    return db_connection
