import time
from typing import Dict, List, Optional

import numpy as np

from .connection import get_db
from ..core.logger import get_logger

logger = get_logger(__name__)

TABLE_NAME = "voiceprints"


def _blob_param(driver: str, raw: bytes):
    """Dialect-safe binary bind parameter."""
    if driver == "postgres":
        from psycopg2.extensions import Binary

        return Binary(raw)
    return raw


def _row_bytes(value) -> bytes:
    if value is None:
        return b""
    if isinstance(value, memoryview):
        return value.tobytes()
    if isinstance(value, bytearray):
        return bytes(value)
    if isinstance(value, bytes):
        return value
    return bytes(value)


class VoiceprintDB:
    """与 LingEchoX 共用 voiceprints 表；Go 写元数据，本服务写 feature_vector。"""

    def update_feature_vector(
        self,
        profile_id: int,
        tenant_id: int,
        feature_id: str,
        emb: np.ndarray,
    ) -> bool:
        db = get_db()
        try:
            with db.get_cursor() as cursor:
                sql = f"""
                UPDATE {TABLE_NAME}
                SET feature_vector = %s, status = 'active'
                WHERE id = %s AND tenant_id = %s AND feature_id = %s
                  AND deleted_at IS NULL
                """
                cursor.execute(
                    sql,
                    (
                        _blob_param(db.driver, emb.tobytes()),
                        profile_id,
                        tenant_id,
                        feature_id,
                    ),
                )
                if cursor.rowcount > 0:
                    logger.success(
                        f"声纹向量已写入: profile={profile_id} feature={feature_id} tenant={tenant_id}"
                    )
                    return True
                logger.warning(
                    f"未找到可更新的声纹记录: profile={profile_id} feature={feature_id} tenant={tenant_id}"
                )
                return False
        except Exception as e:
            logger.fail(
                f"写入声纹向量失败 profile={profile_id} feature={feature_id} tenant={tenant_id}: {e}"
            )
            return False

    def save_voiceprint(
        self,
        tenant_id: int,
        feature_id: str,
        emb: np.ndarray,
        *,
        profile_id: Optional[int] = None,
        assistant_id: Optional[int] = None,
        name: str = "",
        provider: str = "http",
        status: str = "active",
        description: Optional[str] = None,
    ) -> bool:
        if profile_id:
            return self.update_feature_vector(profile_id, tenant_id, feature_id, emb)

        db = get_db()
        blob = _blob_param(db.driver, emb.tobytes())
        row_id = int(time.time() * 1000)
        display_name = name or feature_id
        prov = provider or "http"
        st = status or "active"

        try:
            with db.get_cursor() as cursor:
                if db.driver == "mysql":
                    sql = f"""
                    INSERT INTO {TABLE_NAME} (
                        id, tenant_id, assistant_id, name, provider, feature_id, status, description, feature_vector
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                    ON DUPLICATE KEY UPDATE
                        assistant_id = VALUES(assistant_id),
                        name = VALUES(name),
                        provider = VALUES(provider),
                        status = VALUES(status),
                        description = VALUES(description),
                        feature_vector = VALUES(feature_vector)
                    """
                else:
                    # postgres + sqlite share ON CONFLICT syntax
                    sql = f"""
                    INSERT INTO {TABLE_NAME} (
                        id, tenant_id, assistant_id, name, provider, feature_id, status, description, feature_vector
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                    ON CONFLICT (tenant_id, feature_id) DO UPDATE SET
                        assistant_id = excluded.assistant_id,
                        name = excluded.name,
                        provider = excluded.provider,
                        status = excluded.status,
                        description = excluded.description,
                        feature_vector = excluded.feature_vector
                    """
                cursor.execute(
                    sql,
                    (
                        row_id,
                        tenant_id,
                        assistant_id,
                        display_name,
                        prov,
                        feature_id,
                        st,
                        description,
                        blob,
                    ),
                )
                logger.success(f"声纹已登记: {feature_id} (tenant={tenant_id})")
                return True
        except Exception as e:
            logger.fail(f"保存声纹失败 {feature_id} (tenant={tenant_id}): {e}")
            return False

    def bind_assistant(
        self, tenant_id: int, feature_id: str, assistant_id: Optional[int]
    ) -> bool:
        db = get_db()
        try:
            with db.get_cursor() as cursor:
                sql = f"""
                UPDATE {TABLE_NAME}
                SET assistant_id = %s
                WHERE tenant_id = %s AND feature_id = %s AND deleted_at IS NULL
                """
                cursor.execute(sql, (assistant_id, tenant_id, feature_id))
                return cursor.rowcount > 0
        except Exception as e:
            logger.error(f"绑定助手失败 {feature_id} tenant={tenant_id}: {e}")
            return False

    def get_voiceprints(
        self,
        tenant_id: int,
        feature_ids: Optional[List[str]] = None,
        assistant_id: Optional[int] = None,
    ) -> Dict[str, np.ndarray]:
        start_time = time.time()
        db = get_db()
        try:
            with db.get_cursor() as cursor:
                params: list = [tenant_id]
                sql = (
                    f"SELECT feature_id, feature_vector FROM {TABLE_NAME} "
                    "WHERE tenant_id = %s AND status = 'active' AND deleted_at IS NULL "
                    "AND feature_vector IS NOT NULL"
                )

                if assistant_id:
                    sql += " AND (assistant_id IS NULL OR assistant_id = %s)"
                    params.append(assistant_id)

                if feature_ids:
                    format_strings = ",".join(["%s"] * len(feature_ids))
                    sql += f" AND feature_id IN ({format_strings})"
                    params.extend(feature_ids)

                cursor.execute(sql, tuple(params))
                results = cursor.fetchall()
                voiceprints = {}
                for row in results:
                    fid = row[0]
                    raw = _row_bytes(row[1])
                    if not raw:
                        continue
                    voiceprints[fid] = np.frombuffer(raw, dtype=np.float32)
                logger.info(
                    f"获取到 {len(voiceprints)} 个声纹向量，耗时: {time.time() - start_time:.3f}秒"
                )
                return voiceprints
        except Exception as e:
            logger.error(f"获取声纹向量失败: {e}")
            return {}

    def delete_voiceprint(
        self,
        tenant_id: int,
        feature_id: str,
        assistant_id: Optional[int] = None,
    ) -> bool:
        db = get_db()
        try:
            with db.get_cursor() as cursor:
                if assistant_id:
                    sql = f"""
                    DELETE FROM {TABLE_NAME}
                    WHERE tenant_id = %s AND feature_id = %s AND assistant_id = %s
                    """
                    cursor.execute(sql, (tenant_id, feature_id, assistant_id))
                else:
                    sql = f"DELETE FROM {TABLE_NAME} WHERE tenant_id = %s AND feature_id = %s"
                    cursor.execute(sql, (tenant_id, feature_id))
                return cursor.rowcount > 0
        except Exception as e:
            logger.error(f"删除声纹失败 {feature_id} tenant={tenant_id}: {e}")
            return False

    def count_voiceprints(self, tenant_id: Optional[int] = None) -> int:
        db = get_db()
        try:
            with db.get_cursor() as cursor:
                if tenant_id:
                    sql = (
                        f"SELECT COUNT(*) FROM {TABLE_NAME} "
                        "WHERE tenant_id = %s AND deleted_at IS NULL AND feature_vector IS NOT NULL"
                    )
                    cursor.execute(sql, (tenant_id,))
                else:
                    sql = (
                        f"SELECT COUNT(*) FROM {TABLE_NAME} "
                        "WHERE deleted_at IS NULL AND feature_vector IS NOT NULL"
                    )
                    cursor.execute(sql)
                result = cursor.fetchone()
                return int(result[0]) if result else 0
        except Exception as e:
            logger.error(f"获取声纹总数失败: {e}")
            return 0


voiceprint_db = VoiceprintDB()
