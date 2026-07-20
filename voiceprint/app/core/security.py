from fastapi import HTTPException, Header
from typing import Optional
from .config import settings
from .logger import get_logger

logger = get_logger(__name__)


def verify_token(authorization: str = Header(..., description="接口令牌")) -> bool:
    """
    验证API访问令牌

    Args:
        authorization: 请求头中的授权令牌

    Returns:
        bool: 验证是否通过

    Raises:
        HTTPException: 令牌无效时抛出401错误
    """
    expected_token = f"{settings.api_token}"

    if authorization != expected_token:
        logger.warning(f"无效的接口令牌: {authorization[:20]}...")
        raise HTTPException(status_code=401, detail="无效的接口令牌")

    return True


def get_token_dependency():
    """
    获取令牌验证依赖函数

    Returns:
        function: 令牌验证函数
    """
    return verify_token
