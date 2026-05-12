package cli

import (
	"embed"
	"fmt"
	"io"
	"os"

	ucli "github.com/urfave/cli/v3"
)

//go:embed show-help
var helpFS embed.FS

func init() {
	ucli.HelpPrinter = func(_ io.Writer, _ string, data any) {
		cmd, ok := data.(*ucli.Command)
		if !ok {
			return
		}
		printHelp("otter_" + cmd.Name)
	}
}

func printHelp(name string) {
	b, err := helpFS.ReadFile("show-help/" + name + ".help")
	if err != nil {
		fmt.Fprintf(os.Stderr, "help not found: %s\n", name)
		return
	}
	fmt.Fprint(os.Stdout, string(b))
}
