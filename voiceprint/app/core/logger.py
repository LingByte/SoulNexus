"""
日志管理模块 - 统一的日志配置和记录器
"""

import os
import sys
import logging
import warnings
from typing import Optional
from loguru import logger
from .config import settings
from .version import VERSION

# 移除默认的loguru处理器
logger.remove()


class LoggingHandler(logging.Handler):
    """拦截logging日志并转发到loguru"""

    def emit(self, record):
        # 获取对应的loguru level
        try:
            level = logger.level(record.levelname).name
        except ValueError:
            level = record.levelno

        # 获取logger名称
        logger_name = record.name
        if logger_name.startswith("uvicorn"):
            logger_name = "uvicorn"
        elif logger_name.startswith("fastapi"):
            logger_name = "fastapi"
        elif logger_name.startswith("modelscope"):
            logger_name = "modelscope"
        elif logger_name.startswith("torch"):
            logger_name = "torch"
        elif logger_name.startswith("pydantic"):
            logger_name = "pydantic"
        elif logger_name.startswith("app."):
            logger_name = record.name

        # 转发到loguru
        logger.opt(exception=record.exc_info).bind(
            name=logger_name, version=VERSION
        ).log(level, record.getMessage())


class WarningHandler:
    """拦截warnings并转发到loguru"""

    def __init__(self):
        self.original_showwarning = warnings.showwarning

    def showwarning(self, message, category, filename, lineno, file=None, line=None):
        # 转发到loguru
        logger.bind(name="warnings", version=VERSION).warning(
            f"{category.__name__}: {message}"
        )


class StderrHandler:
    """拦截stderr输出"""

    def __init__(self):
        self.original_stderr = sys.stderr

    def write(self, text):
        if text.strip():  # 忽略空行
            # 尝试解析uvicorn格式的日志
            if (
                text.startswith("INFO:")
                or text.startswith("WARNING:")
                or text.startswith("ERROR:")
            ):
                # 这是uvicorn格式，转换为我们的格式
                parts = text.strip().split(":", 1)
                if len(parts) == 2:
                    level = parts[0].strip()
                    message = parts[1].strip()
                    logger.bind(name="uvicorn", version=VERSION).info(message)
            else:
                # 其他stderr输出
                logger.bind(name="stderr", version=VERSION).warning(text.strip())

    def flush(self):
        self.original_stderr.flush()


def setup_logging(level: Optional[str] = None) -> None:
    """
    设置应用日志配置，使用loguru实现优雅的分段颜色显示

    格式: 时间[青色] 版本号[蓝色] 模块[灰色]-级别[彩色]-消息[绿色]
    示例: 250705 13:33:23[0.6.2][core.utils.modules_initialize]-INFO-初始化组件: intent成功

    Args:
        level: 日志级别
    """
    # 获取配置
    log_level = level or settings.logging.get("level", "INFO")

    # 确保日志目录存在
    log_dir = "logs"
    os.makedirs(log_dir, exist_ok=True)

    # 控制台输出格式 - 分段颜色显示
    console_format = (
        "<cyan>{time:YYMMDD HH:mm:ss}</cyan>"
        "<blue>[{extra[version]}]</blue>"
        "<light-black>[{name}]</light-black>-"
        "<level>{level}</level>-"
        "<green>{message}</green>"
    )

    # 文件输出格式 - 无颜色，保持相同格式
    file_format = (
        "{time:YYMMDD HH:mm:ss}" "[{extra[version]}]" "[{name}]-" "{level}-" "{message}"
    )

    # 添加控制台处理器
    logger.add(
        sys.stdout,
        format=console_format,
        level=log_level,
        colorize=True,
        backtrace=True,
        diagnose=True,
        enqueue=True,
    )

    # 添加文件处理器
    logger.add(
        os.path.join(log_dir, "voiceprint_api.log"),
        format=file_format,
        level=log_level,
        rotation="10 MB",
        retention="7 days",
        compression="gz",
        encoding="utf-8",
        backtrace=True,
        diagnose=True,
        enqueue=True,
    )

    # 拦截所有logging日志
    # 1. 移除root logger的所有handler
    for handler in logging.root.handlers[:]:
        logging.root.removeHandler(handler)

    # 2. 设置root logger只使用我们的handler
    logging.basicConfig(handlers=[LoggingHandler()], level=0, force=True)

    # 3. 强制替换所有已存在的logger的handler
    intercept_handler = LoggingHandler()
    for name in logging.root.manager.loggerDict:
        log = logging.getLogger(name)
        # 移除所有现有handler
        for handler in log.handlers[:]:
            log.removeHandler(handler)
        # 添加我们的handler
        log.addHandler(intercept_handler)
        # 设置propagate为False，防止重复输出
        log.propagate = False

    # 设置第三方库的日志级别
    logger.bind(version=VERSION).info(f"日志系统初始化完成，级别: {log_level}")


class Logger:
    """优雅的日志记录器 - 基于loguru"""

    def __init__(self, name: str):
        self._name = name
        # 直接使用loguru的logger，绑定模块名和版本
        self._logger = logger.bind(name=name, version=VERSION)

    def debug(self, message: str, *args, **kwargs):
        """调试日志"""
        self._logger.debug(message, *args, **kwargs)

    def info(self, message: str, *args, **kwargs):
        """信息日志"""
        self._logger.info(message, *args, **kwargs)

    def warning(self, message: str, *args, **kwargs):
        """警告日志"""
        self._logger.warning(message, *args, **kwargs)

    def error(self, message: str, *args, **kwargs):
        """错误日志"""
        self._logger.error(message, *args, **kwargs)

    def critical(self, message: str, *args, **kwargs):
        """严重错误日志"""
        self._logger.critical(message, *args, **kwargs)

    def success(self, message: str, *args, **kwargs):
        """成功日志（使用INFO级别但语义更清晰）"""
        self._logger.info(f"✅ {message}", *args, **kwargs)

    def fail(self, message: str, *args, **kwargs):
        """失败日志（使用ERROR级别但语义更清晰）"""
        self._logger.error(f"❌ {message}", *args, **kwargs)

    def start(self, operation: str, *args, **kwargs):
        """开始操作日志"""
        self._logger.info(f"🚀 开始: {operation}", *args, **kwargs)

    def complete(
        self, operation: str, duration: Optional[float] = None, *args, **kwargs
    ):
        """完成操作日志"""
        if duration is not None:
            self._logger.info(
                f"✅ 完成: {operation} (耗时: {duration:.3f}秒)", *args, **kwargs
            )
        else:
            self._logger.info(f"✅ 完成: {operation}", *args, **kwargs)

    def init_component(
        self, component_name: str, status: str = "成功", *args, **kwargs
    ):
        """组件初始化日志"""
        if status.lower() in ["成功", "success", "ok"]:
            self._logger.info(
                f"🔧 初始化组件: {component_name} {status}", *args, **kwargs
            )
        else:
            self._logger.error(
                f"🔧 初始化组件: {component_name} {status}", *args, **kwargs
            )


def get_logger(name: str) -> Logger:
    """
    获取优雅的日志记录器

    Args:
        name: 日志记录器名称

    Returns:
        Logger: 日志记录器实例
    """
    return Logger(name)


# 便捷函数
def log_success(message: str, logger_name: str = __name__):
    """记录成功日志"""
    get_logger(logger_name).success(message)


def log_fail(message: str, logger_name: str = __name__):
    """记录失败日志"""
    get_logger(logger_name).fail(message)


def log_start(operation: str, logger_name: str = __name__):
    """记录开始操作"""
    get_logger(logger_name).start(operation)


def log_complete(
    operation: str, duration: Optional[float] = None, logger_name: str = __name__
):
    """记录完成操作"""
    get_logger(logger_name).complete(operation, duration)


def log_init_component(
    component_name: str, status: str = "成功", logger_name: str = __name__
):
    """记录组件初始化"""
    get_logger(logger_name).init_component(component_name, status)
