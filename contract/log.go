package contract

type Logger interface {
	// Debug Adds a log record at the DEBUG level.
	Debug(v string, args ...interface{})
	// Info Adds a log record at the INFO level.
	Info(v string, args ...interface{})
	// Notice Adds a log record at the NOTICE level.
	Notice(v string, args ...interface{})
	// Warn Adds a log record at the WARNING level.
	Warn(v string, args ...interface{})
	// Error Adds a log record at the ERROR level.
	Error(v string, args ...interface{})
	// Crit Adds a log record at the CRITICAL level.
	Crit(v string, args ...interface{})
	// Alert Adds a log record at the ALERT level.
	Alert(v string, args ...interface{})
	// Emerg Adds a log record at the EMERGENCY level.
	Emerg(v string, args ...interface{})
}
