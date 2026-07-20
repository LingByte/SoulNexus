from fastapi import Depends, Header, HTTPException
from fastapi.security import HTTPBearer
from typing import Annotated
from ..core.security import verify_token
from ..core.logger import get_logger

logger = get_logger(__name__)

# 创建HTTPBearer实例
security = HTTPBearer(description="接口令牌")


def get_authorization_token(
    credentials: Annotated[HTTPBearer, Depends(security)],
) -> str:
    """
    获取并验证授权令牌

    Args:
        credentials: HTTPBearer凭证

    Returns:
        str: 验证通过的令牌

    Raises:
        HTTPException: 令牌无效时抛出401错误
    """
    verify_token(credentials.credentials)
    return credentials.credentials


# 类型别名，用于依赖注入
AuthorizationToken = Annotated[str, Depends(get_authorization_token)]
