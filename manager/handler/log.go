package handler

import (
	"context"
	"log/slog"
	"os"

	"github.com/lijcoder/aiapi/constant"
	"github.com/lijcoder/aiapi/manager/base"
)

// LogsResp 日志响应
type LogsResp struct {
	Lines []string `json:"lines"`
	Total int      `json:"total"`
}

// Logs 读取最近 100 条日志（从文件尾倒序读取）
func Logs(ctx context.Context) (*LogsResp, *base.BizError) {
	logFile := constant.LogFilePath()

	f, err := os.Open(logFile)
	if err != nil {
		slog.Error("[LogsLast100] open log file failed", "err", err)
		return nil, base.ErrInternal
	}
	defer f.Close()

	const maxLines = 100
	lines := readLastLines(f, maxLines)

	// 翻转：最新的在前
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	return &LogsResp{Lines: lines, Total: len(lines)}, nil
}

// readLastLines 从文件末尾向前读取最多 n 行
func readLastLines(f *os.File, n int) []string {
	// 获取文件大小
	info, err := f.Stat()
	if err != nil {
		return nil
	}
	size := info.Size()
	if size == 0 {
		return nil
	}

	const chunkSize = 8192
	var buf []byte
	offset := size
	lines := make([]string, 0, n)

	// 从文件尾部向前读块
	for offset > 0 && len(lines) < n {
		readSize := chunkSize
		if offset < int64(readSize) {
			readSize = int(offset)
		}
		offset -= int64(readSize)

		chunk := make([]byte, readSize)
		_, err := f.ReadAt(chunk, offset)
		if err != nil {
			break
		}

		// 新块拼到前面
		buf = append(chunk, buf...)

		// 现在 buf 是从 offset 到文件尾的完整内容
		// 从尾部开始切行
		end := len(buf)
		for len(lines) < n {
			// 从 end-1 向前找 \n
			pos := -1
			for i := end - 1; i >= 0; i-- {
				if buf[i] == '\n' {
					pos = i
					break
				}
			}
			if pos < 0 {
				break
			}
			lines = append(lines, string(buf[pos+1:end]))
			// 去掉 \n 和后面的部分
			buf = buf[:pos]
			end = pos
		}
		// 如果已经够了，buf 保留剩余的前面部分
		// 但 buf 只是当前已读范围内的内容，不需要再处理
	}

	// 处理剩余的第一行（文件头）
	if len(buf) > 0 && len(lines) < n {
		lines = append(lines, string(buf))
	}

	return lines
}
