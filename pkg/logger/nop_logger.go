package logger

// NopLogger 是一个空实现，避免 nil logger 引发 panic
// 主要用于测试或临时占位。
type NopLogger struct{}

func (NopLogger) Debug(string, ...any)  {}
func (NopLogger) Info(string, ...any)   {}
func (NopLogger) Warn(string, ...any)   {}
func (NopLogger) Error(string, ...any)  {}
func (NopLogger) Errorf(string, ...any) {}

func NewNopLogger() Logger { return NopLogger{} }
