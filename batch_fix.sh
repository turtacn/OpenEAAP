#!/bin/bash
# Batch fix common compilation errors

cd /workspace/project/OpenEAAP

echo "Fixing common error patterns..."

# Fix errors.New() calls with 2 arguments - change to appropriate helper functions
# Pattern: errors.New("ERR_INTERNAL", "message") → errors.NewInternalError("ERR_INTERNAL", "message")
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/errors\.New("ERR_INTERNAL",/errors.NewInternalError("ERR_INTERNAL",/g' {} \;
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/errors\.New("VALIDATION_/errors.NewValidationError("VALIDATION_/g' {} \;
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/errors\.New("DATABASE_/errors.NewDatabaseError("DATABASE_/g' {} \;
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/errors\.New("NOT_FOUND",/errors.NewNotFoundError("NOT_FOUND",/g' {} \;
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/errors\.New("TIMEOUT/errors.NewTimeoutError("TIMEOUT/g' {} \;

# Fix logger calls - remove context parameter
# Pattern: logger.Error(ctx, "message", ...) → logger.Error("message", ...)
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/\.logger\.Error(context\.Background(), /\.logger\.Error(/g' {} \;
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/\.logger\.Warn(context\.Background(), /\.logger\.Warn(/g' {} \;
find . -name "*.go" -not -path "./vendor/*" -exec sed -i 's/\.logger\.Info(context\.Background(), /\.logger\.Info(/g' {} \;

# Fix logger field format
# Pattern: "key", value → logging.String("key", value) or logging.Any("key", value)
# This is complex and needs manual intervention, skip for now

echo "Basic fixes complete. Re-running compilation..."
export PATH=/workspace/go/bin:$PATH
go build ./... 2>&1 | grep -E "^(#|.*\.go:[0-9]+:)" | head -50
