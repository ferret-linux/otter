package ui

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorDim    = "\033[37m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

func Red(text string) string {
	return colorRed + text + colorReset
}

func Green(text string) string {
	return colorGreen + text + colorReset
}

func Yellow(text string) string {
	return colorYellow + text + colorReset
}

func Dim(text string) string {
	return colorDim + text + colorReset
}

func Bold(text string) string {
	return colorBold + text + colorReset
}
