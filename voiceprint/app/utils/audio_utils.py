import os
import tempfile
import soundfile as sf
import librosa
import numpy as np
import time
from typing import Tuple
from ..core.config import settings
from ..core.logger import get_logger

logger = get_logger(__name__)


class AudioProcessor:
    """音频处理工具类"""

    def __init__(self):
        self.target_sample_rate = settings.target_sample_rate
        self.tmp_dir = settings.tmp_dir
        # 确保临时目录存在
        os.makedirs(self.tmp_dir, exist_ok=True)

    def ensure_16k_wav(self, audio_bytes: bytes) -> str:
        """
        将任意采样率的wav bytes转为16kHz wav临时文件

        Args:
            audio_bytes: 音频字节数据

        Returns:
            str: 临时文件路径
        """
        start_time = time.time()
        logger.debug(f"开始音频处理，输入大小: {len(audio_bytes)}字节")

        with tempfile.NamedTemporaryFile(
            delete=False, suffix=".wav", dir=self.tmp_dir
        ) as tmpf:
            tmpf.write(audio_bytes)
            tmp_path = tmpf.name

        try:
            # 读取原采样率
            read_start = time.time()
            data, sr = sf.read(tmp_path)
            read_time = time.time() - read_start
            logger.debug(
                f"音频文件读取完成，采样率: {sr}Hz，时长: {len(data)/sr:.2f}秒，耗时: {read_time:.3f}秒"
            )

            if sr != self.target_sample_rate:
                # librosa重采样，支持多通道
                resample_start = time.time()
                logger.debug(f"开始音频重采样: {sr}Hz -> {self.target_sample_rate}Hz")

                if data.ndim == 1:
                    data_rs = librosa.resample(
                        data, orig_sr=sr, target_sr=self.target_sample_rate
                    )
                else:
                    data_rs = np.vstack(
                        [
                            librosa.resample(
                                data[:, ch],
                                orig_sr=sr,
                                target_sr=self.target_sample_rate,
                            )
                            for ch in range(data.shape[1])
                        ]
                    ).T

                resample_time = time.time() - resample_start
                logger.debug(f"音频重采样完成，耗时: {resample_time:.3f}秒")

                # 写入重采样后的音频
                write_start = time.time()
                sf.write(tmp_path, data_rs, self.target_sample_rate)
                write_time = time.time() - write_start
                logger.debug(f"重采样音频写入完成，耗时: {write_time:.3f}秒")

            total_time = time.time() - start_time
            logger.debug(f"音频处理完成，总耗时: {total_time:.3f}秒")
            return tmp_path

        except Exception as e:
            # 清理临时文件
            if os.path.exists(tmp_path):
                os.remove(tmp_path)
            total_time = time.time() - start_time
            logger.error(f"音频处理失败，总耗时: {total_time:.3f}秒，错误: {e}")
            raise

    def validate_audio_file(self, audio_bytes: bytes) -> bool:
        """
        验证音频文件格式是否有效（简化版本）

        Args:
            audio_bytes: 音频字节数据

        Returns:
            bool: 音频文件是否有效
        """
        start_time = time.time()
        logger.debug(f"开始音频文件验证，输入大小: {len(audio_bytes)}字节")

        try:
            with tempfile.NamedTemporaryFile(
                delete=False, suffix=".wav", dir=self.tmp_dir
            ) as tmpf:
                tmpf.write(audio_bytes)
                tmp_path = tmpf.name

            # 尝试读取音频文件
            read_start = time.time()
            data, sr = sf.read(tmp_path)
            read_time = time.time() - read_start
            logger.debug(
                f"音频文件读取完成，采样率: {sr}Hz，数据长度: {len(data)}，耗时: {read_time:.3f}秒"
            )

            # 检查音频数据
            if len(data) == 0:
                logger.warning("音频文件为空")
                return False

            # 检查采样率
            if sr < 8000:  # 最低采样率要求
                logger.warning(f"采样率过低: {sr}Hz")
                return False

            # 检查音频时长（至少0.5秒，最多30秒）
            duration = len(data) / sr
            if duration < 0.5:
                logger.warning(f"音频时长过短: {duration:.2f}秒")
                return False
            elif duration > 30:
                logger.warning(f"音频时长过长: {duration:.2f}秒")
                return False

            total_time = time.time() - start_time
            logger.debug(
                f"音频验证通过: {duration:.2f}秒, {sr}Hz，总耗时: {total_time:.3f}秒"
            )
            return True

        except Exception as e:
            total_time = time.time() - start_time
            logger.error(f"音频文件验证失败，总耗时: {total_time:.3f}秒，错误: {e}")
            return False
        finally:
            # 清理临时文件
            if "tmp_path" in locals() and os.path.exists(tmp_path):
                os.remove(tmp_path)

    def cleanup_temp_file(self, file_path: str) -> None:
        """
        清理临时文件

        Args:
            file_path: 临时文件路径
        """
        try:
            if file_path and os.path.exists(file_path):
                os.remove(file_path)
                logger.debug(f"临时文件已清理: {file_path}")
        except Exception as e:
            logger.debug(f"清理临时文件失败 {file_path}: {e}")


# 全局音频处理器实例
audio_processor = AudioProcessor()
