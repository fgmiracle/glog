package glog

type Level uint8

const (
	Level_Debug Level = iota
	Level_Info
	Level_Warn
	Level_Error
)

var levelString = [...]string{
	"[DEBU]",
	"[INFO]",
	"[WARN]",
	"[ERRO]",
}

func str2loglevel(level string) Level {
	switch level {
	case "debug":
		return Level_Debug
	case "info":
		return Level_Info
	case "warn":
		return Level_Warn
	case "error", "err":
		return Level_Error
	}

	return Level_Debug
}

func logPart_level(b *[]byte, tpl *tplInfo) {
	if tpl.tpl.logRule {
		s := levelString[tpl.currLog.log.level]
		*b = append(*b, s...)
	}

}
