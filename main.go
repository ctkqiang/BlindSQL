package main

import (
	"blind_sql/internal/utilities"
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	applicationName    = "BlindSQL"
	applicationTagline = "盲注 payload 加载器"
	ruleWidth          = 42
)

// 终端 ANSI 样式；颜色在 NO_COLOR 或 TERM=dumb 环境下会自动关闭。
const (
	styleReset   = "\033[0m"
	styleBold    = "\033[1m"
	styleDim     = "\033[2m"
	styleAccent  = "\033[36m"
	styleSuccess = "\033[32m"
	styleWarning = "\033[33m"
)

// standardInputReader 全进程复用同一个缓冲读取器。
// bufio.Reader 会预读比一行更多的数据，若每次调用都新建实例，
// 多次读取时前一次已缓冲的输入就会被丢弃。
var standardInputReader = bufio.NewReader(os.Stdin)

// useColor 决定是否输出 ANSI 颜色：FORCE_COLOR 可强制开启，
// 否则遵循 NO_COLOR 约定，并在 TERM=dumb 时退化，避免终端出现不可读的转义序列。
var useColor = os.Getenv("FORCE_COLOR") != "" ||
	(os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb")

func init() {
	// 日志属于诊断信息，统一写到 stderr，让 stdout 只承载 payload，
	// 这样把 stdout 重定向到文件时，文件里就只有 payload，不含日志与界面装饰。
	// 关闭 caller 显示是因为带 f 后缀的日志方法多一层跳栈，会打印出错误的文件行号。
	utilities.Default = utilities.New(utilities.Config{
		MinLevel:     utilities.LevelInfo,
		Output:       os.Stderr,
		EnableColor:  useColor,
		EnableCaller: false,
	})
}

func main() {
	flag.Usage = usage
	flag.Parse()

	fmt.Fprint(os.Stderr, banner())

	// 优先使用命令行传入的路径，未提供时回退到交互式输入。
	path := flag.Arg(0)

	if path == "" {
		enteredPath, err := readPath()
		if err != nil {
			utilities.Default.Error(err)
			os.Exit(1)
		}

		path = enteredPath
	}

	if path == "" {
		fmt.Fprintln(os.Stderr, "  "+paint(styleWarning, "未输入路径，已取消"))
		return
	}

	payload, err := loadPayload(path)
	if err != nil {
		utilities.Default.Error(err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stderr, report(path, payload))
	fmt.Print(payload)

	// 无论是否重定向，输出都以换行结束：换行由代码发出，
	// 而不是往 payload 字符串里拼，避免终端提示符贴在内容后面。
	if !strings.HasSuffix(payload, "\n") {
		fmt.Println()
	}
}

// usage 覆盖 flag 的默认帮助信息，说明位置参数的含义。
func usage() {
	output := flag.CommandLine.Output()

	fmt.Fprintf(output, "用法: %s [payload 文件路径]\n", filepath.Base(os.Args[0]))
	fmt.Fprintln(output, "  省略路径时，将在提示符下交互式输入。")
	fmt.Fprintln(output, "  示例: go run . payload.txt")
	fmt.Fprintln(output, "  环境变量: NO_COLOR=1 关闭彩色输出; FORCE_COLOR=1 强制开启。")
}

// banner 生成启动标题。
func banner() string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "\n  %s %s %s\n",
		paint(styleBold+styleAccent, applicationName),
		paint(styleDim, "·"),
		paint(styleDim, applicationTagline),
	)
	builder.WriteString(rule())
	builder.WriteString("\n")

	return builder.String()
}

// report 生成 payload 的加载结果摘要。
func report(path, payload string) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "\n  %s %s\n",
		paint(styleSuccess, "已加载"),
		paint(styleBold, path),
	)
	fmt.Fprintf(&builder, "  %s\n", paint(styleDim, fmt.Sprintf("%d 字节 · %d 行", len(payload), lineCount(payload))))
	builder.WriteString(rule())
	builder.WriteString("\n")

	return builder.String()
}

// readPath 在交互模式下从标准输入读取一行文件路径，并去除首尾空白。
func readPath() (string, error) {
	fmt.Fprintf(os.Stderr, "\n  %s %s", paint(styleAccent, "Payload"), paint(styleDim, "› "))

	input, err := standardInputReader.ReadString('\n')

	// 管道输入可能没有结尾换行符，此时 ReadString 会同时返回内容与 io.EOF，
	// 只要拿到了内容就不应视为错误。
	if err != nil && !(errors.Is(err, io.EOF) && input != "") {
		return "", fmt.Errorf("读取输入失败: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// loadPayload 读取指定文件的完整内容作为盲注 payload。
// 内容原样返回、不做裁剪，避免破坏 payload 的首尾空白或换行语义。
func loadPayload(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", path, err)
	}

	if len(content) == 0 {
		return "", fmt.Errorf("文件 %s 内容为空", path)
	}

	return string(content), nil
}

// lineCount 统计文本行数：以换行符分隔，末尾缺少换行符的最后一行同样计入。
func lineCount(text string) int {
	if text == "" {
		return 0
	}

	lines := strings.Count(text, "\n")

	if !strings.HasSuffix(text, "\n") {
		lines++
	}

	return lines
}

// rule 生成一条等宽分隔线。
func rule() string {
	return paint(styleDim, "  "+strings.Repeat("─", ruleWidth))
}

// paint 在启用颜色时用 ANSI 样式包裹文本，否则原样返回。
func paint(style, text string) string {
	if !useColor {
		return text
	}

	return style + text + styleReset
}
