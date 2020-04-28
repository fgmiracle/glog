package glog

import "strings"

type Color int

const (
	NoColor Color = iota
	Black
	Red
	Green
	Yellow
	Blue
	Purple
	DarkGreen
	White
)

var logColorPrefix = []string{
	"",
	"\x1b[030m",
	"\x1b[031m",
	"\x1b[032m",
	"\x1b[033m",
	"\x1b[034m",
	"\x1b[035m",
	"\x1b[036m",
	"\x1b[037m",
}

type colorData struct {
	name string
	c    Color
}

var colorByName = []colorData{
	{"none", NoColor},
	{"black", Black},
	{"red", Red},
	{"green", Green},
	{"yellow", Yellow},
	{"blue", Blue},
	{"purple", Purple},
	{"darkgreen", DarkGreen},
	{"white", White},
}

func matchColor(name string) Color {

	lower := strings.ToLower(name)

	for _, d := range colorByName {

		if d.name == lower {
			return d.c
		}
	}

	return NoColor
}

func colorFromLevel(l Level) Color {
	switch l {
	case Level_Warn:
		return Yellow
	case Level_Error:
		return Red
	}

	return NoColor
}

var logColorSuffix = "\x1b[0m"

func logPart_colorbegin(b *[]byte, tpl *tplInfo) {
	if tpl.tpl.logRule {
		s := logColorPrefix[colorFromLevel(tpl.currLog.log.level)]
		*b = append(*b, s...)
	}

}

func logPart_colorend(b *[]byte, tpl *tplInfo) {
	if tpl.tpl.logRule {
		*b = append(*b, logColorSuffix...)
	}
}
