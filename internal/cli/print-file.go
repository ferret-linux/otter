package cli

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"

	ucli "github.com/urfave/cli/v3"
)

//go:embed show-file
var helpFS embed.FS

var colorSlots = []string{
	"\033[32m", // {0} green   — command names
	"\033[96m", // {1} teal   — tagline/messages
	"\033[36m", // {2} cyan    — box borders
	"\033[33m", // {3} yellow  — headers (▸ Commands:)
	"\033[34m", // {4} blue    — flags/extras
	"\033[37m", // {5} dim     — descriptions/info text
}

func init() {
	ucli.HelpPrinter = func(_ io.Writer, _ string, data any) {
		cmd, ok := data.(*ucli.Command)
		if !ok {
			return
		}
		printHelp("otter_" + cmd.Name)
	}
}

func renderColors(s string) string {
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	if noColor {
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

func printHelp(name string) {
	b, err := helpFS.ReadFile("show-file/" + name + ".help")
	if err != nil {
		fmt.Fprintf(os.Stderr, "help not found: %s\n", name)
		return
	}
	fmt.Fprint(os.Stdout, renderColors(string(b)))
}
