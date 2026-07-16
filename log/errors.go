package log

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// WithStack 给错误附加调用位置（文件名:行号）
// 使用: aiapiLog.WithStack(err)
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return err
	}
	return fmt.Errorf("%s:%d: %w", filepath.Base(file), line, err)
}
