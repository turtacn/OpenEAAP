package logging

import "context"

// LogWithCtx is a compatibility wrapper for logger calls with context
// TODO: Replace with proper WithContext().Method() calls
func LogWithCtx(logger Logger, ctx context.Context, level string, msg string, args ...interface{}) {
// Ignore args for now, just log the message
switch level {
case "debug":
logger.WithContext(ctx).Debug(msg)
case "info":
logger.WithContext(ctx).Info(msg)
case "warn":
logger.WithContext(ctx).Warn(msg)
case "error":
logger.WithContext(ctx).Error(msg)
}
}
