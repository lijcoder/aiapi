package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
)

// Formatter 输出格式化的日志
// 格式: 2026-07-16 12:46:57.341 [INFO]  store/store.go:45 > msg  - key=value
type Formatter struct {
	w     io.Writer
	level slog.Level
}

func NewFormatter(w io.Writer, level slog.Level) *Formatter {
	return &Formatter{w: w, level: level}
}

func (h *Formatter) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *Formatter) Handle(_ context.Context, r slog.Record) error {
	time := r.Time.Format("2006-01-02 15:04:05.000")
	level := r.Level.String()

	// 从 PC 提取调用源文件行号
	source := ""
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		// 只取文件名，不要全路径
		if idx := strings.LastIndex(f.File, "/"); idx >= 0 {
			source = " " + f.File[idx+1:] + fmt.Sprintf(":%d", f.Line)
		} else {
			source = " " + f.File + fmt.Sprintf(":%d", f.Line)
		}
	}

	_, err := fmt.Fprintf(h.w, "%s [%s]%s %s", time, level, source, r.Message)
	if err != nil {
		return err
	}

	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(h.w, "  - %s=%v", a.Key, a.Value.Any())
		return true
	})

	_, err = fmt.Fprintln(h.w)
	return err
}

func (h *Formatter) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *Formatter) WithGroup(_ string) slog.Handler      { return h }
