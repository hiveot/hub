package utils

// Console ASCII commands to control colors
// See also: https://www.codequoi.com/en/coloring-terminal-text-tput-and-ansi-escape-sequences/
const (
	COReset  = "\033[0m"
	CORed    = "\033[31m"
	COGreen  = "\033[32m"
	COYellow = "\033[33m"
	COBlue   = "\033[34m"
	COPurple = "\033[35m"
	COCyan   = "\033[36m"
	COGray   = "\033[37m"
	COWhite  = "\033[97m"
)

const (
	CBBlack  = "\033[40m"
	CBRed    = "\033[41m"
	CBGreen  = "\033[42m"
	CBYellow = "\033[43m"
	CBBlue   = "\033[44m"
	CBGray   = "\033[47m"
	CBWhite  = "\033[41m"
)

const (
	WrapOff = "\033[?7l"
	WrapOn  = "\033[?7h"
)
