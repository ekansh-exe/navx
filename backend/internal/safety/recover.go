package safety

import (
	"log/slog"
	"runtime/debug"
)

func Recover(job string) {
	if r := recover(); r != nil {
		slog.Error("PANIC_RECOVERED", "job", job, "panic", r, "stack", string(debug.Stack()))
	}
}
