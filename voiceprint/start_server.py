#!/usr/bin/env python3
"""
生产环境启动脚本
"""

# 导入统一日志模块（会自动执行早期日志设置）
from app.core.logger import setup_logging, get_logger

setup_logging()

import os
import sys
import socket
import signal
import uvicorn
from pathlib import Path

# 添加项目根目录到Python路径
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from app.core.config import settings

logger = get_logger(__name__)


def get_local_ip():
    """获取本机IP地址"""
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        local_ip = s.getsockname()[0]
        s.close()
        return local_ip
    except Exception:
        return "127.0.0.1"


def signal_handler(signum, frame):
    """信号处理器"""
    logger.info(f"收到信号 {signum}，正在优雅关闭服务...")
    sys.exit(0)


def main():
    """主函数"""
    # 注册信号处理器
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    try:
        logger.start(
            f"生产环境服务启动（Uvicorn），监听地址: {settings.host}:{settings.port}"
        )
        logger.info("=" * 60)
        local_ip = get_local_ip()
        logger.info(
            f"声纹接口地址: http://{local_ip}:{settings.port}/voiceprint/health?key="
            + settings.api_token
        )
        logger.info("=" * 60)

        # 使用Uvicorn启动，配置优化
        uvicorn.run(
            "app.application:app",
            host=settings.host,
            port=settings.port,
            reload=False,  # 生产环境关闭热重载
            workers=1,  # 单进程模式，避免模型重复加载
            access_log=False,  # 关闭uvicorn自带access日志
            log_level="info",
            timeout_keep_alive=30,  # keep-alive超时
            timeout_graceful_shutdown=300,  # 优雅关闭超时
            limit_concurrency=1000,  # 并发连接限制
            limit_max_requests=1000,  # 最大请求数限制
            backlog=2048,  # 连接队列大小
        )

    except KeyboardInterrupt:
        logger.info("收到中断信号，正在退出服务。")
    except Exception as e:
        logger.fail(f"服务启动失败: {e}")
        raise


if __name__ == "__main__":
    main()
