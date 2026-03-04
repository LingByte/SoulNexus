package utils

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"time"
)

// CalculateFileHash 计算文件的SHA1哈希
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GenerateID 生成唯一的ID
func GenerateID() string {
	// 使用时间戳和随机数生成ID
	timestamp := time.Now().UnixNano()
	hash := sha1.Sum([]byte(fmt.Sprintf("%d", timestamp)))
	return fmt.Sprintf("%x", hash[:8])
}

// IsDirectory 检查路径是否为目录
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile 检查路径是否为文件
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileSize 获取文件大小
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("复制文件失败: %v", err)
	}

	return nil
}

// EnsureDirectory 确保目录存在
func EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// RemoveFile 删除文件
func RemoveFile(path string) error {
	return os.Remove(path)
}

// RemoveDirectory 删除目录
func RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}
