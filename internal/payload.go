package internal

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultPayloadPath 是默认 payload 文件相对项目根目录的路径。
const DefaultPayloadPath = "payload.txt"

// defaultPayloadContent 是随二进制一起分发的内置 payload 列表，每行一个 payload。
// 项目根目录缺少 payload.txt 时用它生成默认文件。
//
//go:embed default_payload.txt
var defaultPayloadContent string

// LoadDefaultPayload 读取项目根目录下的默认 payload 文件，按行返回其中的 payload。
// 文件不存在时，先用内置列表创建它，再读取。
func LoadDefaultPayload() ([]string, error) {
	rootDirectory, err := projectRootDirectory()
	if err != nil {
		return nil, err
	}

	payloadPath := filepath.Join(rootDirectory, DefaultPayloadPath)
	if err := ensurePayloadFile(payloadPath); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(payloadPath)
	if err != nil {
		return nil, fmt.Errorf("读取 payload 文件 %s 失败: %w", payloadPath, err)
	}

	return splitPayloadLines(string(content)), nil
}

// ensurePayloadFile 保证 payloadPath 存在：文件缺失时写入内置的默认 payload。
func ensurePayloadFile(payloadPath string) error {
	_, err := os.Stat(payloadPath)
	if err == nil {
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("检查 payload 文件 %s 失败: %w", payloadPath, err)
	}

	if err := os.WriteFile(payloadPath, []byte(defaultPayloadContent), 0o644); err != nil {
		return fmt.Errorf("创建 payload 文件 %s 失败: %w", payloadPath, err)
	}

	return nil
}

// projectRootDirectory 从当前工作目录向上查找包含 go.mod 的目录；
// 一直找不到时退回当前工作目录。
func projectRootDirectory() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前工作目录失败: %w", err)
	}

	currentDirectory := workingDirectory

	for {
		if _, err := os.Stat(filepath.Join(currentDirectory, "go.mod")); err == nil {
			return currentDirectory, nil
		}

		parentDirectory := filepath.Dir(currentDirectory)
		if parentDirectory == currentDirectory {
			return workingDirectory, nil
		}

		currentDirectory = parentDirectory
	}
}

// splitPayloadLines 按行切分 payload 列表：去掉行尾的 \r，并跳过空行。
func splitPayloadLines(content string) []string {
	var payloads []string

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSuffix(line, "\r")

		if line == "" {
			continue
		}

		payloads = append(payloads, line)
	}

	return payloads
}
