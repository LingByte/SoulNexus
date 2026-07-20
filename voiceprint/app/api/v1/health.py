from fastapi import APIRouter, HTTPException, Query
import time
from ...services.voiceprint_service import voiceprint_service
from ...core.logger import get_logger
from ...core.config import settings

logger = get_logger(__name__)

router = APIRouter()


@router.get(
    "/health",
    summary="健康检查",
    response_model=dict,
    description="检查服务运行状态，需要提供正确的密钥",
)
async def health_check(
    key: str = Query(..., description="访问密钥", example="your-secret-key")
):
    """
    健康检查接口

    Args:
        key: 访问密钥，必须与配置中的authorization密钥匹配

    Returns:
        dict: 服务状态信息

    Raises:
        HTTPException: 当密钥不正确时返回401错误
    """
    start_time = time.time()
    logger.start("健康检查请求")

    # 验证密钥
    key_check_start = time.time()
    if key != settings.api_token:
        logger.warning(f"健康检查接口收到无效密钥: {key}")
        raise HTTPException(status_code=401, detail="密钥验证失败")
    key_check_time = time.time() - key_check_start
    logger.info(f"密钥验证完成，耗时: {key_check_time:.3f}秒")

    try:
        count_start = time.time()
        logger.info("开始获取声纹统计信息...")
        count = voiceprint_service.get_voiceprint_count()
        count_time = time.time() - count_start
        logger.info(f"声纹统计信息获取完成，总数: {count}，耗时: {count_time:.3f}秒")

        total_time = time.time() - start_time
        logger.complete("健康检查请求", total_time)
        return {"total_voiceprints": count, "status": "healthy"}
    except Exception as e:
        total_time = time.time() - start_time
        logger.fail(f"获取统计信息异常，总耗时: {total_time:.3f}秒，错误: {e}")
        raise HTTPException(status_code=500, detail=f"获取统计信息失败: {str(e)}")
