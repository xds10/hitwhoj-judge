package minio

import (
	"context"
	"fmt"
	"hitwh-judge/internal/dao"
	"io"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
)

// DownloadFileByMD5 根据 bucket 和 md5 下载文件
// bucket: MinIO 存储桶名称
// md5: 文件的 MD5 哈希值（用于构建文件路径或对象名称）
// savePath: 可选参数，指定本地保存路径；若为空则返回文件内容字节数组
// 返回值: 文件内容字节数组（当 savePath 为空时）、错误信息
func DownloadFileByMD5(bucket, md5, savePath string) ([]byte, error) {
	// 1. 参数校验
	if bucket == "" || md5 == "" {
		return nil, fmt.Errorf("bucket and md5 cannot be empty")
	}

	// 2. 构建对象名称（根据项目约定，这里假设使用 md5 作为对象名称或路径）
	// 注意：实际项目中可能需要根据具体存储策略构建对象名称
	// 例如：md5 的前两位作为目录，后 30 位作为文件名: "ab/cdef1234567890..."
	objectName := md5 // 这里简化处理，直接使用 md5 作为对象名称

	// 3. 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 4. 获取对象
	object, err := dao.MinIOClient.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object fail: %w", err)
	}
	defer object.Close()

	// 5. 读取对象内容
	var content []byte
	if savePath != "" {
		// 保存到本地文件
		file, err := os.Create(savePath)
		if err != nil {
			return nil, fmt.Errorf("create local file fail: %w", err)
		}
		defer file.Close()

		// 复制内容到文件
		_, err = io.Copy(file, object)
		if err != nil {
			return nil, fmt.Errorf("save file fail: %w", err)
		}

		fmt.Printf("file download success! save to: %s\n", savePath)
		return nil, nil // 保存到本地时返回 nil 内容
	} else {
		// 返回字节数组
		content, err = io.ReadAll(object)
		if err != nil {
			return nil, fmt.Errorf("read object content fail: %w", err)
		}

		fmt.Printf("file download success! size: %d bytes\n", len(content))
		return content, nil
	}
}

// DownloadFileByMD5AsString 根据 bucket 和 md5 下载文件并返回 string 形式
// bucket: MinIO 存储桶名称
// md5: 文件的 MD5 哈希值（用于构建文件路径或对象名称）
// 返回值: 文件内容字符串、错误信息
func DownloadFileByMD5AsString(bucket, md5 string) (string, error) {
	// 1. 参数校验
	if bucket == "" || md5 == "" {
		return "", fmt.Errorf("bucket and md5 cannot be empty")
	}

	// 2. 构建对象名称
	objectName := md5 // 与 DownloadFileByMD5 保持一致的对象名称构建方式

	// 3. 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 4. 获取对象
	object, err := dao.MinIOClient.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("get object fail: %w", err)
	}
	defer object.Close()

	// 5. 读取对象内容并转换为字符串
	contentBytes, err := io.ReadAll(object)
	if err != nil {
		return "", fmt.Errorf("read object content fail: %w", err)
	}

	contentStr := string(contentBytes)
	fmt.Printf("file download success as string! size: %d bytes\n", len(contentStr))
	return contentStr, nil
}
