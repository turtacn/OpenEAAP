#!/usr/bin/env python3
"""Fix remaining compilation errors in OpenEAAP project."""

import re
import os

def fix_file(filepath, fixes):
    """Apply fixes to a file."""
    with open(filepath, 'r') as f:
        content = f.read()
    
    original = content
    for pattern, replacement in fixes:
        content = re.sub(pattern, replacement, content, flags=re.MULTILINE | re.DOTALL)
    
    if content != original:
        with open(filepath, 'w') as f:
            f.write(content)
        print(f"Fixed: {filepath}")
        return True
    return False

# Fix l1_local.go
l1_local_fixes = [
    # Fix logger call with context.Background()
    (r'c\.logger\.Debug\(context\.Background\(\), "L1 cache cleanup completed"\s*\n\s*"expired_count", len\(expiredKeys\),\s*\n\s*"remaining_count", len\(c\.entries\)\s*\n\s*\)',
     'c.logger.Debug("L1 cache cleanup completed",\n\t\tlogging.Int("expired_count", len(expiredKeys)),\n\t\tlogging.Int("remaining_count", len(c.entries)))'),
]

# Fix l2_redis.go
l2_redis_fixes = [
    # Find and fix incomplete logger calls
    (r'(\w+\.logger\.(?:Debug|Info|Warn|Error))\(([^)]+)\n([^)]*$)', r'\1(\2)\n\3'),
]

# Fix l3_vector.go  
l3_vector_fixes = [
    # Fix unexpected ) 
]

# Fix pii_detector.go - wrong logger API usage
pii_detector_fixes = [
    (r'd\.logger\.Info\(context\.Background\(\), "ML model loading not implemented", "path", modelPath\)',
     'd.logger.Info("ML model loading not implemented", logging.String("path", modelPath))'),
]

# Fix audit_logger.go - wrong logger API usage
audit_logger_fixes = [
    (r'a\.logger\.Info\(context\.Background\(\), "Closing audit logger"\)',
     'a.logger.Info("Closing audit logger")'),
    (r'a\.logger\.Error\(context\.Background\(\), "Failed to flush buffer on close", "error", err\)',
     'a.logger.Error("Failed to flush buffer on close", logging.Error(err))'),
    (r'a\.logger\.Info\(context\.Background\(\), "Audit logger closed"\)',
     'a.logger.Info("Audit logger closed")'),
    (r'a\.logger\.Error\(context\.Background\(\), "Auto flush failed"',
     'a.logger.Error("Auto flush failed"'),
]

# Fix vllm_client.go
vllm_fixes = [
    # Fix incomplete logger call
]

# Fix generator.go
generator_fixes = [
    # Fix syntax error with "Error" name
]

# Apply fixes
files_to_fix = [
    ('internal/platform/inference/cache/l1_local.go', l1_local_fixes),
    ('internal/platform/inference/privacy/pii_detector.go', pii_detector_fixes),
    ('internal/governance/audit/audit_logger.go', audit_logger_fixes),
]

base_dir = '/workspace/project/OpenEAAP'
for filepath, fixes in files_to_fix:
    full_path = os.path.join(base_dir, filepath)
    if os.path.exists(full_path):
        fix_file(full_path, fixes)

print("Done!")
