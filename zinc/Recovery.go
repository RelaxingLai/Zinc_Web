package zinc

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// trace 方法获取堆栈跟踪信息用以debug
func trace(message string) string {
	var pcs [32]uintptr

	// 跳过前三个 Caller
	// 第 0 个 Caller 是 Callers 本身，第 1 个是上一层 trace，第 2 个是再上一层的 defer func。
	n := runtime.Callers(3, pcs[:])

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

// 错误处理中间件
func Recovery() HandlerFunc {
	return func(c *Context) {
		// panic 发生时立即调用被defer延迟的函数
		defer func() {
			// 捕获 panic
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				// 将堆栈信息打印在日志中
				// trace 获取触发 panic 的堆栈信息
				log.Printf("%s\n\n", trace(message))
				// 向用户返回 Internal Server Error
				c.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		// 执行后面的中间件或Handler
		c.Next()
	}
}
