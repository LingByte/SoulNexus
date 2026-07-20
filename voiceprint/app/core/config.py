import os
import yaml
import uuid
import logging
from typing import Dict, Any
from pathlib import Path

logger = logging.getLogger(__name__)


class Settings:
    """应用配置管理类"""

    def __init__(self):
        self._config = self._load_config()

    def _load_config(self) -> Dict[str, Any]:
        """
        加载配置文件，优先读取环境变量（适合Docker部署），否则读取本地yaml。
        如果authorization不足32位或为空，自动生成UUID并更新配置文件。
        """
        config_path = Path("data/.voiceprint.yaml")
        if not config_path.exists():
            logger.error("配置文件 data/.voiceprint.yaml 未找到，请先配置。")
            raise RuntimeError("请先配置 data/.voiceprint.yaml")

        with open(config_path, "r", encoding="utf-8") as f:
            config = yaml.safe_load(f)

        # 检查authorization字段
        if "server" not in config:
            config["server"] = {}

        authorization = config["server"].get("authorization", "")

        # 如果authorization为空或长度不足32位，生成新的UUID
        if not authorization or len(str(authorization)) < 32:
            new_authorization = str(uuid.uuid4())
            config["server"]["authorization"] = new_authorization

            # 更新配置文件
            with open(config_path, "w", encoding="utf-8") as f:
                yaml.dump(config, f, default_flow_style=False, allow_unicode=True)

            logger.info(f"已自动生成新的authorization密钥: {new_authorization}")
            logger.info("配置文件已更新，请妥善保管此密钥")

        return config

    @property
    def server(self) -> Dict[str, Any]:
        """服务器配置"""
        return self._config.get("server", {})

    @property
    def mysql(self) -> Dict[str, Any]:
        """Legacy MySQL block (still supported). Prefer `database` or env DB_DRIVER/DSN."""
        return self._config.get("mysql", {})

    @property
    def database(self) -> Dict[str, Any]:
        """Unified DB config: {driver, dsn} — mirrors LingEchoX DB_DRIVER/DSN."""
        return self._config.get("database", {})

    @property
    def voiceprint(self) -> Dict[str, Any]:
        """声纹识别配置"""
        return self._config.get("voiceprint", {})

    @property
    def logging(self) -> Dict[str, Any]:
        """日志配置"""
        return self._config.get("logging", {})

    @property
    def api_token(self) -> str:
        """API访问令牌"""
        return self.server.get("authorization", "")

    @property
    def host(self) -> str:
        """服务器监听地址"""
        return self.server.get("host", "0.0.0.0")

    @property
    def port(self) -> int:
        """服务器监听端口"""
        return self.server.get("port", 8005)

    @property
    def similarity_threshold(self) -> float:
        """声纹相似度阈值"""
        return self.voiceprint.get("similarity_threshold", 0.2)

    @property
    def target_sample_rate(self) -> int:
        """目标音频采样率"""
        return self.voiceprint.get("target_sample_rate", 16000)

    @property
    def tmp_dir(self) -> str:
        """临时文件目录"""
        return self.voiceprint.get("tmp_dir", "tmp")


# 全局配置实例
settings = Settings()
