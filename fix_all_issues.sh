#!/bin/bash

# Script to fix common compilation issues

echo "Fixing logger API issues..."

# Find all files with logger calls that need fixing
FILES=$(find internal pkg -name "*.go" -type f)

for file in $FILES; do
    # Skip if file doesn't exist or is not readable
    [ ! -f "$file" ] && continue
    
    # Check if file has logger issues
    if grep -q 'logger\.\(Info\|Warn\|Debug\|Error\)(ctx,' "$file" 2>/dev/null; then
        echo "Processing $file..."
        
        # Simple patterns - single line logger calls without extra args
        sed -i 's/logger\.Info(ctx, \("\([^"]*\)"\))/logger.WithContext(ctx).Info(\1)/g' "$file"
        sed -i 's/logger\.Warn(ctx, \("\([^"]*\)"\))/logger.WithContext(ctx).Warn(\1)/g' "$file"
        sed -i 's/logger\.Debug(ctx, \("\([^"]*\)"\))/logger.WithContext(ctx).Debug(\1)/g' "$file"
        sed -i 's/logger\.Error(ctx, \("\([^"]*\)"\))/logger.WithContext(ctx).Error(\1)/g' "$file"
        
        # Add logging import if WithContext is used
        if grep -q '\.WithContext(ctx)\.' "$file"; then
            if ! grep -q '"github.com/openeeap/openeeap/internal/observability/logging"' "$file"; then
                # Add import if needed
                sed -i '/^import (/a\"github.com/openeeap/openeeap/internal/observability/logging"' "$file"
            fi
        fi
    fi
done

echo "Logger API fixes completed."
