#!/bin/bash
# Script to fix remaining compilation errors

cd /workspace/project/OpenEAAP

# Fix errors.CodePermissionDenied -> errors.PermissionError
find internal -name "*.go" -exec sed -i 's/errors\.CodePermissionDenied/errors.PermissionError/g' {} \;
find internal -name "*.go" -exec sed -i 's/errors\.ErrUnauthorized/errors.UnauthorizedError/g' {} \;
find internal -name "*.go" -exec sed -i 's/errors\.CodeDeadlineExceeded/errors.DeadlineError/g' {} \;

# Fix MetricsCollector pointer issues - change *MetricsCollector to MetricsCollector in struct literals
find internal -name "*.go" -exec sed -i 's/\*metrics\.MetricsCollector/metrics.MetricsCollector/g' {} \;

# Fix vectorCacheEntry.Similarity - it should be Score
find internal -name "*.go" -exec sed -i 's/topResult\.Similarity/topResult.Score/g' {} \;
find internal -name "*.go" -exec sed -i 's/entry\.Similarity/entry.Score/g' {} \;

echo "Bulk fixes applied"
