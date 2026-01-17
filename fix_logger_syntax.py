#!/usr/bin/env python3
"""Fix missing closing parentheses in logger calls."""

import re
import sys

def fix_logger_calls(filename):
    """Fix incomplete logger calls in a file."""
    try:
        with open(filename, 'r') as f:
            lines = f.readlines()
        
        modified = False
        for i, line in enumerate(lines):
            # Check if line has a logger call that's missing closing parenthesis
            # Pattern: ends with ) but not enough closing parens for the call
            if 'logger' in line and ('Info(' in line or 'Warn(' in line or 'Error(' in line or 'Debug(' in line):
                # Count opening and closing parentheses
                open_count = line.count('(')
                close_count = line.count(')')
                
                # If there are more opening than closing parens, add closing parens
                if open_count > close_count:
                    missing = open_count - close_count
                    # Add the missing closing parentheses at the end
                    lines[i] = line.rstrip() + ')' * missing + '\n'
                    modified = True
                    print(f"Fixed line {i+1} in {filename}: added {missing} closing paren(s)")
        
        if modified:
            with open(filename, 'w') as f:
                f.writelines(lines)
            return True
        return False
    except Exception as e:
        print(f"Error processing {filename}: {e}", file=sys.stderr)
        return False

if __name__ == "__main__":
    files_to_fix = [
        "internal/platform/rag/generator.go",
        "internal/platform/rag/rag_engine.go",
        "internal/platform/rag/reranker.go",
        "internal/platform/rag/retriever.go",
        "internal/platform/inference/cache/cache_manager.go",
        "internal/platform/inference/cache/l1_local.go",
        "internal/platform/inference/cache/l2_redis.go",
        "internal/governance/audit/audit_logger.go",
        "internal/governance/policy/pdp.go",
        "internal/governance/policy/pep.go",
        "internal/governance/policy/policy_loader.go",
    ]
    
    for filename in files_to_fix:
        print(f"Processing {filename}...")
        fix_logger_calls(filename)
