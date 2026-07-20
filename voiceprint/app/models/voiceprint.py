from pydantic import BaseModel
from typing import Optional


class VoiceprintProfile(BaseModel):
    """与 LingEchoX VoiceprintProfile 一致的业务模型（向量由服务端写入）。"""

    tenant_id: int
    assistant_id: Optional[int] = None
    name: str
    provider: str = "http"
    feature_id: str
    status: str = "active"
    description: Optional[str] = None


class VoiceprintRegisterRequest(BaseModel):
    feature_id: str
    tenant_id: int
    assistant_id: Optional[int] = None
    name: str
    provider: str = "http"
    status: str = "active"
    description: Optional[str] = None


class VoiceprintRegisterResponse(BaseModel):
    success: bool
    msg: str
    feature_id: Optional[str] = None


class VoiceprintIdentifyRequest(BaseModel):
    feature_ids: str
    tenant_id: int
    assistant_id: Optional[int] = None


class VoiceprintIdentifyResponse(BaseModel):
    feature_id: str
    score: float
    speaker_id: str  # deprecated alias of feature_id


class VoiceprintBindResponse(BaseModel):
    success: bool
    msg: str


class VoiceprintDeleteResponse(BaseModel):
    success: bool
    msg: str
