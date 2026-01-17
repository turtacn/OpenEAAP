#!/bin/bash
# Batch fix compilation errors
cd /workspace/project/OpenEAAP

# Fix logger calls with context.Background() as first argument  
echo "Fixing logger API calls..."
find internal/governance/policy -name "pdp.go" -exec sed -i 's/p\.logger\.Debug(context\.Background(), "\([^"]*\)")/p.logger.Debug("\1")/g' {} \;
find internal/platform/inference/cache -name "l1_local.go" -exec sed -i 's/c\.logger\.Debug(context\.Background(), "\([^"]*\)", "\([^"]*\)", \([^)]*\))/c.logger.Debug("\1", logging.String("\2", \3))/g' {} \;

# Fix metrics API - Increment -> IncrementCounter
find internal -name "*.go" -exec sed -i 's/\.metricsCollector\.Increment(/\.metricsCollector.IncrementCounter(/g' {} \;
# Fix metrics API - Histogram/RecordHistogram -> RecordDuration  
find internal -name "*.go" -exec sed -i 's/\.metricsCollector\.Histogram(/\.metricsCollector.RecordDuration(/g' {} \;
find internal -name "*.go" -exec sed -i 's/\.metricsCollector\.RecordHistogram(/\.metricsCollector.RecordDuration(/g' {} \;

echo "Done!"
