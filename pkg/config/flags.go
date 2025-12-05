// Package config
package config

// LogLevelFlag set the log level. Overrides env var LOG_LEVEL
// Flag: 				LogLevel (string)
// default: 		""
var LogLevelFlag string

// WorkDirFlag set the path to the root of target application
var WorkDirFlag string

// ProcfileFlag set the path to Procfile
var ProcfileFlag string

var ForegroundFlag bool
