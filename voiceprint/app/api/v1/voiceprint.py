from fastapi import APIRouter, File, UploadFile, Form, HTTPException, Depends
from fastapi.security import HTTPBearer
from typing import Optional
import time
from ...models.voiceprint import (
    VoiceprintRegisterResponse,
    VoiceprintIdentifyResponse,
    VoiceprintBindResponse,
    VoiceprintDeleteResponse,
)
from ...services.voiceprint_service import voiceprint_service
from ...api.dependencies import AuthorizationToken
from ...core.logger import get_logger

security = HTTPBearer(description="接口令牌")
logger = get_logger(__name__)
router = APIRouter()


def _parse_optional_uint(value: Optional[str]) -> Optional[int]:
    raw = (value or "").strip()
    if not raw:
        return None
    try:
        parsed = int(raw)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail="invalid assistant_id") from exc
    if parsed <= 0:
        raise HTTPException(status_code=400, detail="invalid assistant_id")
    return parsed


def _resolve_tenant_id(tenant_id: Optional[str], agent_id: Optional[str]) -> int:
    raw = (tenant_id or agent_id or "").strip()
    if not raw:
        raise HTTPException(status_code=400, detail="tenant_id is required")
    try:
        parsed = int(raw)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail="invalid tenant_id") from exc
    if parsed <= 0:
        raise HTTPException(status_code=400, detail="invalid tenant_id")
    return parsed


def _resolve_feature_id(feature_id: Optional[str], speaker_id: Optional[str]) -> str:
    value = (feature_id or speaker_id or "").strip()
    if not value:
        raise HTTPException(status_code=400, detail="feature_id is required")
    return value


def _resolve_feature_ids(feature_ids: Optional[str], speaker_ids: Optional[str]) -> list[str]:
    raw = (feature_ids or speaker_ids or "").strip()
    if not raw:
        raise HTTPException(status_code=400, detail="feature_ids is required")
    ids = [x.strip() for x in raw.split(",") if x.strip()]
    if not ids:
        raise HTTPException(status_code=400, detail="feature_ids is required")
    return ids


@router.post(
    "/register",
    summary="声纹注册",
    response_model=VoiceprintRegisterResponse,
    description="在租户下注册声纹；字段与 LingEchoX VoiceprintProfile 一致",
    dependencies=[Depends(security)],
)
async def register_voiceprint(
    token: AuthorizationToken,
    file: UploadFile = File(..., description="WAV音频文件"),
    feature_id: Optional[str] = Form(None, description="特征ID（对应平台 featureId）"),
    speaker_id: Optional[str] = Form(None, description="兼容旧字段，等同 feature_id"),
    tenant_id: Optional[str] = Form(None, description="租户ID"),
    agent_id: Optional[str] = Form(None, description="兼容旧字段，等同 tenant_id"),
    assistant_id: Optional[str] = Form(None, description="助手ID，可选"),
    profile_id: Optional[str] = Form(None, description="LingEchoX 已创建的行 ID，仅更新 feature_vector"),
    name: Optional[str] = Form(None, description="显示名称"),
    provider: Optional[str] = Form("http", description="提供方 slug"),
    status: Optional[str] = Form("active", description="状态 active/failed"),
    description: Optional[str] = Form(None, description="备注"),
):
    try:
        tid = _resolve_tenant_id(tenant_id, agent_id)
        fid = _resolve_feature_id(feature_id, speaker_id)
        if not file.filename.lower().endswith(".wav"):
            raise HTTPException(status_code=400, detail="只支持WAV格式音频文件")

        audio_bytes = await file.read()
        aid = _parse_optional_uint(assistant_id)
        pid = _parse_optional_uint(profile_id)
        display_name = (name or "").strip() or fid
        prov = (provider or "http").strip() or "http"
        st = (status or "active").strip() or "active"
        desc = (description or "").strip() or None

        success = voiceprint_service.register_voiceprint(
            tid,
            fid,
            audio_bytes,
            profile_id=pid,
            assistant_id=aid,
            name=display_name,
            provider=prov,
            status=st,
            description=desc,
        )

        if success:
            msg = f"已登记: {fid} (tenant: {tid})"
            if aid:
                msg += f" assistant: {aid}"
            return VoiceprintRegisterResponse(success=True, msg=msg, feature_id=fid)
        raise HTTPException(status_code=500, detail="声纹注册失败")
    except HTTPException:
        raise
    except Exception as e:
        logger.fail(f"声纹注册异常: {e}")
        raise HTTPException(status_code=500, detail=f"声纹注册失败: {str(e)}")


@router.post(
    "/identify",
    summary="声纹识别",
    response_model=VoiceprintIdentifyResponse,
    description="在租户范围内识别说话人；assistant_id 可选用于限定助手绑定范围",
    dependencies=[Depends(security)],
)
async def identify_voiceprint(
    token: AuthorizationToken,
    file: UploadFile = File(..., description="WAV音频文件"),
    feature_ids: Optional[str] = Form(None, description="候选 featureId，逗号分隔"),
    speaker_ids: Optional[str] = Form(None, description="兼容旧字段，等同 feature_ids"),
    tenant_id: Optional[str] = Form(None, description="租户ID"),
    agent_id: Optional[str] = Form(None, description="兼容旧字段，等同 tenant_id"),
    assistant_id: Optional[str] = Form(None, description="助手ID，可选"),
):
    start_time = time.time()
    try:
        tid = _resolve_tenant_id(tenant_id, agent_id)
        if not file.filename.lower().endswith(".wav"):
            raise HTTPException(status_code=400, detail="只支持WAV格式音频文件")

        candidate_ids = _resolve_feature_ids(feature_ids, speaker_ids)
        audio_bytes = await file.read()
        aid = _parse_optional_uint(assistant_id)
        match_id, match_score = voiceprint_service.identify_voiceprint(
            tid, candidate_ids, audio_bytes, aid
        )
        logger.info(
            f"声纹识别完成，耗时 {time.time() - start_time:.3f}s，结果 {match_id} score={match_score:.4f}"
        )
        return VoiceprintIdentifyResponse(
            feature_id=match_id,
            score=match_score,
            speaker_id=match_id,
        )
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"声纹识别异常: {e}")
        raise HTTPException(status_code=500, detail=f"声纹识别失败: {str(e)}")


@router.put(
    "/{feature_id}/assistant",
    summary="绑定助手",
    description="为已注册声纹绑定或解绑助手（assistant_id 为空表示租户级通用声纹）",
    dependencies=[Depends(security)],
)
async def bind_voiceprint_assistant(
    token: AuthorizationToken,
    feature_id: str,
    tenant_id: Optional[str] = Form(None, description="租户ID"),
    agent_id: Optional[str] = Form(None, description="兼容旧字段，等同 tenant_id"),
    assistant_id: Optional[str] = Form(None, description="助手ID，留空表示解绑"),
):
    try:
        tid = _resolve_tenant_id(tenant_id, agent_id)
        fid = feature_id.strip()
        if not fid:
            raise HTTPException(status_code=400, detail="feature_id is required")
        aid = _parse_optional_uint(assistant_id)
        success = voiceprint_service.bind_assistant(tid, fid, aid)
        if success:
            msg = f"已绑定: {fid} -> assistant {aid or '(none)'}"
            return VoiceprintBindResponse(success=True, msg=msg)
        raise HTTPException(status_code=404, detail=f"未找到声纹: {fid} tenant={tid}")
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"绑定助手异常 {feature_id}: {e}")
        raise HTTPException(status_code=500, detail=f"绑定助手失败: {str(e)}")


@router.delete(
    "/{feature_id}",
    summary="删除声纹",
    description="删除租户下指定 featureId 的声纹特征",
    dependencies=[Depends(security)],
)
async def delete_voiceprint(
    token: AuthorizationToken,
    feature_id: str,
    tenant_id: Optional[str] = Form(None, description="租户ID"),
    agent_id: Optional[str] = Form(None, description="兼容旧字段，等同 tenant_id"),
    assistant_id: Optional[str] = Form(None, description="助手ID，可选，仅删除该助手绑定记录"),
):
    try:
        tid = _resolve_tenant_id(tenant_id, agent_id)
        fid = feature_id.strip()
        if not fid:
            raise HTTPException(status_code=400, detail="feature_id is required")
        aid = _parse_optional_uint(assistant_id)
        success = voiceprint_service.delete_voiceprint(tid, fid, aid)

        if success:
            msg = f"已删除: {fid} (tenant: {tid})"
            if aid:
                msg += f" assistant: {aid}"
            return VoiceprintDeleteResponse(success=True, msg=msg)
        raise HTTPException(status_code=404, detail=f"未找到声纹: {fid} tenant={tid}")
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"删除声纹异常 {feature_id}: {e}")
        raise HTTPException(status_code=500, detail=f"删除声纹失败: {str(e)}")
