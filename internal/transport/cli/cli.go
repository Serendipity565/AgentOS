package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"AgentOS/internal/agent"
	"AgentOS/pkg/schema"
)

type Options struct {
	Provider string
	Model    string
	Skills   int
	MCPs     int
}

const (
	// 重置所有样式（颜色、加粗等）
	ansiReset = "\033[0m"

	// 加粗文本（Bold）
	ansiBold = "\033[1m"

	// 变暗（Dim / Faint）
	ansiDim = "\033[2m"

	// 红色文本（Red）
	ansiRed = "\033[31m"

	// 灰色文本（Gray / Bright Black）
	ansiGray = "\033[90m"

	// 紫色文本（Purple / Magenta）
	ansiPurple = "\033[95m"

	// 蓝色文本（Blue）
	ansiBlue = "\033[34m"

	// 白色文本（White）
	ansiWhite = "\033[37m"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Run starts a styled REPL adapter that translates terminal input into agent
// service calls and streams the response back to stdout.
func Run(ctx context.Context, service *agent.Service, opts Options) error {
	commands := service.CommandNames()
	printWelcome(opts, commands)

	scanner := bufio.NewScanner(os.Stdin)
	history := make([]schema.Message, 0, 16)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		renderPrompt()
		if !scanner.Scan() {
			return scanner.Err()
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Println(colorize("Session closed.", ansiDim, ansiGray))
			return nil
		}

		history = append(history, schema.Message{
			Role:    schema.RoleUser,
			Content: line,
		})

		fmt.Println()
		fmt.Print(colorize("● ", ansiBold, ansiPurple))

		if service.IsLocalCommand(line) {
			resp, err := service.Handle(ctx, schema.ChatRequest{
				SessionID: "term",
				Messages:  history,
			})
			if err != nil {
				fmt.Println(colorize("error: "+err.Error(), ansiBold, ansiRed))
				history = history[:len(history)-1]
				fmt.Println()
				continue
			}

			fmt.Print(colorizeSlashCommands(resp.Message.Content, commands))
			fmt.Println()
			fmt.Println()

			history = append(history, resp.Message)
			continue
		}

		var started bool
		resp, err := service.HandleStream(ctx, schema.ChatRequest{
			SessionID: "term",
			Messages:  history,
		}, func(delta string) error {
			started = true
			fmt.Print(delta)
			return nil
		})
		if err != nil {
			fmt.Println(colorize("error: "+err.Error(), ansiBold, ansiRed))
			history = history[:len(history)-1]
			fmt.Println()
			continue
		}

		if !started {
			fmt.Print(resp.Message.Content)
		}
		fmt.Println()
		fmt.Println()

		history = append(history, resp.Message)
	}
}

func printWelcome(opts Options, commands []string) {
	cardLines := []string{
		renderHeroLine("AgentOS CLI", fallback(opts.Model, "local session"), ""),
		renderHeroLine("Describe a task to get started.", "", ""),
		"",
		"Tip: " + colorizeSlashCommands("/help View builtin commands, /skill and /mcp route tools.", commands),
		"AgentOS uses AI. Check important results before acting on them.",
	}

	lines := []string{
		box(cardLines, ansiPurple),
		renderStatus("Environment loaded", fmt.Sprintf("%d skills, %d MCP servers", opts.Skills, opts.MCPs)),
		renderStatus("Provider", fallback(opts.Provider, "mock")),
		renderStatus("Model", fallback(opts.Model, "not configured")),
	}

	fmt.Println(strings.Join(lines, "\n\n"))
	fmt.Println()
}

func renderStatus(label, value string) string {
	return colorize("● ", ansiBold, ansiBlue) +
		label + ": " +
		value
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

// colorize wraps the input text with ANSI escape codes for styling in the terminal.
func colorize(text string, codes ...string) string {
	var builder strings.Builder
	for _, code := range codes {
		builder.WriteString(code)
	}
	builder.WriteString(text)
	builder.WriteString(ansiReset)
	return builder.String()
}

func colorizeSlashCommands(text string, commands []string) string {
	replacements := make([]string, 0, len(commands)*2)
	for _, command := range commands {
		replacements = append(replacements, command, colorize(command, ansiBold, ansiBlue))
	}
	replacer := strings.NewReplacer(replacements...)
	return replacer.Replace(text)
}

func visibleWidth(text string) int {
	return len(ansiPattern.ReplaceAllString(text, ""))
}

func box(lines []string, borderColor string) string {
	width := 0
	for _, line := range lines {
		if visibleWidth(line) > width {
			width = visibleWidth(line)
		}
	}

	var out []string
	out = append(out, colorize("╭"+strings.Repeat("─", width+2)+"╮", borderColor))
	for _, line := range lines {
		padding := strings.Repeat(" ", width-visibleWidth(line))
		out = append(out, colorize("│ ", borderColor)+line+padding+colorize(" │", borderColor))
	}
	out = append(out, colorize("╰"+strings.Repeat("─", width+2)+"╯", borderColor))
	return strings.Join(out, "\n")
}

func renderHeroLine(icon, primary, secondary string) string {
	var parts []string
	if icon != "" {
		parts = append(parts, icon)
	}
	if primary != "" {
		parts = append(parts, primary)
	}
	if secondary != "" {
		parts = append(parts, secondary)
	}
	return strings.Join(parts, "  ")
}

func renderPrompt() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Print(colorize("❯ ", ansiBold, ansiWhite))
		return
	}

	home, _ := os.UserHomeDir()
	display := cwd
	if home != "" && strings.HasPrefix(cwd, home) {
		display = "~" + strings.TrimPrefix(cwd, home)
	}
	display = filepath.Clean(display)

	fmt.Println(colorize(display, ansiDim, ansiGray))
	fmt.Println(colorize(strings.Repeat("─", max(len(display), 40)), ansiGray))
	fmt.Print(colorize("❯ ", ansiBold, ansiWhite))
}
