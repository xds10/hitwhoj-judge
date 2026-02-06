package errors

import (
	"fmt"
)

// ErrorCode 错误码类型
type ErrorCode int

const (
	// 系统错误 (1000-1999)
	ErrCodeSystem ErrorCode = 1000 + iota
	ErrCodeInternal
	ErrCodeTimeout
	ErrCodeResourceExhausted
	ErrCodeNotFound
	ErrCodeAlreadyExists

	// 参数错误 (2000-2999)
	ErrCodeInvalidParam ErrorCode = 2000 + iota
	ErrCodeMissingParam
	ErrCodeInvalidLanguage
	ErrCodeInvalidTimeLimit
	ErrCodeInvalidMemoryLimit
	ErrCodeInvalidTestCase

	// 编译错误 (3000-3999)
	ErrCodeCompile ErrorCode = 3000 + iota
	ErrCodeCompilerNotFound
	ErrCodeCompileTimeout

	// 运行错误 (4000-4999)
	ErrCodeRuntime ErrorCode = 4000 + iota
	ErrCodeSandboxNotFound
	ErrCodeSandboxFailed
	ErrCodeExecutionTimeout

	// 存储错误 (5000-5999)
	ErrCodeStorage ErrorCode = 5000 + iota
	ErrCodeFileNotFound
	ErrCodeFileDownloadFailed
	ErrCodeCacheFailed
)

// JudgeError 评测系统错误
type JudgeError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error 实现 error 接口
func (e *JudgeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持错误链
func (e *JudgeError) Unwrap() error {
	return e.Err
}

// New 创建新的评测错误
func New(code ErrorCode, message string) *JudgeError {
	return &JudgeError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装已有错误
func Wrap(code ErrorCode, message string, err error) *JudgeError {
	return &JudgeError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 预定义的错误创建函数

// NewInvalidParamError 创建参数错误
func NewInvalidParamError(param string, reason string) *JudgeError {
	return New(ErrCodeInvalidParam, fmt.Sprintf("参数 %s 无效: %s", param, reason))
}

// NewCompileError 创建编译错误
func NewCompileError(message string, err error) *JudgeError {
	return Wrap(ErrCodeCompile, message, err)
}

// NewSandboxError 创建沙箱错误
func NewSandboxError(message string, err error) *JudgeError {
	return Wrap(ErrCodeSandboxFailed, message, err)
}

// NewStorageError 创建存储错误
func NewStorageError(message string, err error) *JudgeError {
	return Wrap(ErrCodeStorage, message, err)
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(operation string) *JudgeError {
	return New(ErrCodeTimeout, fmt.Sprintf("操作超时: %s", operation))
}

// NewResourceExhaustedError 创建资源耗尽错误
func NewResourceExhaustedError(resource string) *JudgeError {
	return New(ErrCodeResourceExhausted, fmt.Sprintf("资源耗尽: %s", resource))
}

// IsErrorCode 判断错误是否为指定错误码
func IsErrorCode(err error, code ErrorCode) bool {
	if judgeErr, ok := err.(*JudgeError); ok {
		return judgeErr.Code == code
	}
	return false
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	if judgeErr, ok := err.(*JudgeError); ok {
		return judgeErr.Code
	}
	return ErrCodeInternal
}
