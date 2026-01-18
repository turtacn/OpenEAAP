#!/bin/bash
# Fix logger calls to use logging.String() for fields

files=(
    "internal/platform/runtime/native/native_runtime.go"
    "internal/platform/runtime/langchain/langchain_adapter.go"
    "internal/infrastructure/message/kafka/kafka_client.go"
)

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "Processing $file..."
        # Replace logger calls with proper Field usage
        # Pattern: "key", value -> logging.String("key", value)
        sed -i -E 's/\.logger\.(Info|Debug|Warn|Error|Fatal)\("([^"]+)", "([^"]+)", ([^,)]+)/\.logger.\1("\2", logging.String("\3", \4)/g' "$file"
    fi
done

echo "Done!"
