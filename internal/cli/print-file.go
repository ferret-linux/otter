package cli

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/ferret-linux/otter/pkg/ui"
)

//go:embed show-file
var printFS embed.FS

//nolint:gochecknoglobals // package-level color slot table is effectively a constant
var colorSlots = []string{
	"\033[32m", // {0} green   — command names
	"\033[96m", // {1} teal   — tagline/messages
	"\033[36m", // {2} cyan    — box borders
	"\033[33m", // {3} yellow  — headers (▸ Commands:)
	"\033[34m", // {4} blue    — flags/extras
	"\033[37m", // {5} dim     — descriptions/info text
}

//nolint:gochecknoinits // required to hook urfave/cli help printer before command execution
func init() {
	//nolint:reassign // intentional: hooks urfave/cli HelpPrinter for custom help rendering
	ucli.HelpPrinter = func(_ io.Writer, _ string, data any) {
		cmd, ok := data.(*ucli.Command)
		if !ok {
			return
		}
		printFile(strings.ReplaceAll(cmd.FullName(), " ", "_"))
	}
}

func renderColors(s string) string {
	if ui.NoColor() {
		for i := range colorSlots {
			s = strings.ReplaceAll(s, fmt.Sprintf("{%d}", i), "")
		}
		return strings.ReplaceAll(s, "{R}", "")
	}
	for i, code := range colorSlots {
		s = strings.ReplaceAll(s, fmt.Sprintf("{%d}", i), code)
	}
	return strings.ReplaceAll(s, "{R}", "\033[0m")
}

func contentWidth(raw string) int {
	width := 0
	for _, line := range strings.Split(raw, "\n") {
		clean := line
		for i := range colorSlots {
			clean = strings.ReplaceAll(clean, fmt.Sprintf("{%d}", i), "")
		}
		clean = strings.ReplaceAll(clean, "{R}", "")
		if runes := len([]rune(clean)); runes > width {
			width = runes
		}
	}
	return width
}

func printFile(name string) {
	b, err := printFS.ReadFile("show-file/" + name + ".help")
	if err != nil {
		fmt.Fprintf(os.Stderr, "File not found: %s\n", name)
		return
	}

	termW, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		needW := contentWidth(string(b))
		if termW < needW {
			ui.DefaultLogger.Warn("terminal window is too small to display help")
			return
		}
	}

	fmt.Fprint(os.Stdout, renderColors(string(b)))
}
