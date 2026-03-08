package logger

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Errorf(format string, args ...any)
}

type Factory interface {
	Get(category string) Logger
}

type Config struct {
	Dir           string
	Level         string
	Format        string // "text" or "json"
	ConsoleFormat string // "text" or "json"
	EnableConsole bool
}
