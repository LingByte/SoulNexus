from fastapi import FastAPI, Request
from fastapi.security import HTTPBearer
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import RedirectResponse
from fastapi.openapi.docs import get_swagger_ui_html, get_redoc_html
from fastapi.openapi.utils import get_openapi

from .api.v1.api import api_router
from loguru import logger
from .core.version import VERSION
import time


def create_app() -> FastAPI:
    """创建FastAPI应用实例"""

    # 创建安全模式
    security = HTTPBearer(description="接口令牌")

    # 创建应用
    app = FastAPI(
        title="3D-Speaker 声纹识别API",
        description="基于3D-Speaker的声纹注册与识别服务",
        version=VERSION,
        docs_url=None,  # 禁用默认的docs路径
        redoc_url=None,  # 禁用默认的redoc路径
    )

    # 添加CORS中间件
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],  # 生产环境中应该限制具体域名
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # 注册API路由
    app.include_router(api_router, prefix="/voiceprint")

    # 自定义OpenAPI文档路径
    @app.get("/voiceprint/openapi.json", include_in_schema=False)
    async def custom_openapi():
        """自定义OpenAPI JSON路径"""
        if app.openapi_schema:
            return app.openapi_schema
        app.openapi_schema = get_openapi(
            title=app.title,
            version=app.version,
            description=app.description,
            routes=app.routes,
        )
        return app.openapi_schema

    # 自定义Swagger UI文档
    @app.get("/voiceprint/docs", include_in_schema=False)
    async def custom_swagger_ui_html():
        """自定义Swagger UI文档页面"""
        return get_swagger_ui_html(
            openapi_url="/voiceprint/openapi.json",
            title=app.title + " - Swagger UI",
            swagger_js_url="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui-bundle.js",
            swagger_css_url="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui.css",
        )

    # 自定义ReDoc文档
    @app.get("/voiceprint/redoc", include_in_schema=False)
    async def custom_redoc_html():
        """自定义ReDoc文档页面"""
        return get_redoc_html(
            openapi_url="/voiceprint/openapi.json",
            title=app.title + " - ReDoc",
            redoc_js_url="https://cdn.jsdelivr.net/npm/redoc@2.1.3/bundles/redoc.standalone.js",
        )

    # 根路径
    @app.get("/", include_in_schema=False)
    def root():
        """根路径，自动跳转到API文档"""
        return RedirectResponse(url="/voiceprint/docs")

    # /voiceprint/ 跳转到 /voiceprint/docs
    @app.get("/voiceprint/", include_in_schema=False)
    def voiceprint_root():
        return RedirectResponse(url="/voiceprint/docs")

    return app


# 创建应用实例
app = create_app()
