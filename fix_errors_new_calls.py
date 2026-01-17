#!/usr/bin/env python3
"""Fix errors.New() calls with missing arguments."""

import re
import os

def fix_file(filepath):
    """Fix errors.New() calls in a file."""
    with open(filepath, 'r') as f:
        content = f.read()
    
    original = content
    
    # Pattern 1: errors.New(errors.CodeXXX, "message") -> appropriate helper
    # CodeInvalidArgument -> ValidationError
    content = re.sub(
        r'errors\.New\(errors\.CodeInvalidArgument,\s*([^)]+)\)',
        r'errors.ValidationError(\1)',
        content
    )
    
    # Pattern 2: errors.New(errors.CodeInternal, msg) -> InternalError
    content = re.sub(
        r'errors\.New\(errors\.CodeInternal,\s*([^)]+)\)',
        r'errors.InternalError(\1)',
        content
    )
    
    # Pattern 3: errors.New(errors.CodeUnimplemented, msg) -> InternalError with message
    content = re.sub(
        r'errors\.New\(errors\.CodeUnimplemented,\s*([^)]+)\)',
        r'errors.InternalError(\1)',
        content
    )
    
    # Pattern 4: errors.New("message") or errors.New(variable) -> InternalError
    # This is trickier - need to match single-argument calls
    content = re.sub(
        r'errors\.New\(([^,)]+)\)(?!\s*\.)',
        r'errors.InternalError(\1)',
        content
    )
    
    if content != original:
        with open(filepath, 'w') as f:
            f.write(content)
        return True
    return False

# List of files to fix
files_to_fix = [
    'internal/platform/rag/reranker.go',
    'internal/platform/rag/retriever.go',
    'internal/platform/inference/cache/cache_manager.go',
    'internal/platform/inference/cache/l1_local.go',
    'internal/platform/inference/cache/l3_vector.go',
    'internal/platform/training/training_service.go',
    'internal/governance/audit/audit_query.go',
    'internal/governance/policy/pep.go',
    'internal/platform/runtime/langchain/langchain_adapter.go',
]

base_dir = '/workspace/project/OpenEAAP'
fixed_count = 0

for filepath in files_to_fix:
    full_path = os.path.join(base_dir, filepath)
    if os.path.exists(full_path):
        if fix_file(full_path):
            print(f"Fixed: {filepath}")
            fixed_count += 1
    else:
        print(f"Not found: {filepath}")

print(f"\nTotal files fixed: {fixed_count}")
