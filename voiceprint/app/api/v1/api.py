from fastapi import APIRouter
from . import voiceprint, health

# 创建API路由器
api_router = APIRouter()

# 注册各个模块的路由
api_router.include_router(health.router, tags=["健康检查"])
api_router.include_router(voiceprint.router, tags=["声纹识别"])
