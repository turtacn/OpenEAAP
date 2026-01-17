#!/bin/bash
# Fix errors.New calls with proper helpers

file="internal/platform/runtime/native/native_runtime.go"

# Replace patterns
sed -i 's/errors\.New(errors\.CodeInvalidParameter, \(.*\))/errors.NewValidationError(errors.CodeInvalidParameter, \1)/g' "$file"
sed -i 's/errors\.New(errors\.CodeInternalError, \(.*\))/errors.NewInternalError(errors.CodeInternalError, \1)/g' "$file"
sed -i 's/errors\.New(errors\.CodeDeadlineExceeded, \(.*\))/errors.NewTimeoutError("DEADLINE_EXCEEDED", \1)/g' "$file"

echo "Fixed $file"
