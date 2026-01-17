#!/usr/bin/env python3
"""
Fix errors.New() calls throughout the codebase to use proper helper functions or full signature.
"""

import re
import os
from pathlib import Path

def fix_errors_new_calls(file_path):
    """Fix errors.New() calls in a single file."""
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original_content = content
    
    # Replace errors.New(errors.CodeXXX, "message") patterns with appropriate helpers
    replacements = [
        # CodeInvalidParameter/CodeInvalidArgument -> NewValidationError
        (r'errors\.New\(errors\.CodeInvalidParameter,\s*', 'errors.NewValidationError(errors.CodeInvalidParameter, '),
        (r'errors\.New\(errors\.CodeInvalidArgument,\s*', 'errors.NewValidationError(errors.CodeInvalidArgument, '),
        
        # CodeNotFound -> NewNotFoundError
        (r'errors\.New\(errors\.CodeNotFound,\s*', 'errors.NewNotFoundError(errors.CodeNotFound, '),
        
        # CodeUnauthorized -> NewAuthenticationError
        (r'errors\.New\(errors\.CodeUnauthorized,\s*', 'errors.NewAuthenticationError(errors.CodeUnauthorized, '),
        
        # CodeForbidden -> NewAuthorizationError
        (r'errors\.New\(errors\.CodeForbidden,\s*', 'errors.NewAuthorizationError(errors.CodeForbidden, '),
        
        # CodeDatabaseError -> NewDatabaseError
        (r'errors\.New\(errors\.CodeDatabaseError,\s*', 'errors.NewDatabaseError(errors.CodeDatabaseError, '),
        
        # CodeInternalError -> NewInternalError
        (r'errors\.New\(errors\.CodeInternalError,\s*', 'errors.NewInternalError(errors.CodeInternalError, '),
        
        # errors.New("CODE", "message") -> use appropriate helper based on code string
        (r'errors\.New\("VALIDATION_[^"]*",\s*', lambda m: m.group(0).replace('errors.New(', 'errors.NewValidationError(')),
        (r'errors\.New\("NOT_FOUND",\s*', 'errors.NewNotFoundError("NOT_FOUND", '),
        (r'errors\.New\("UNAUTHORIZED",\s*', 'errors.NewAuthenticationError("UNAUTHORIZED", '),
        (r'errors\.New\("FORBIDDEN",\s*', 'errors.NewAuthorizationError("FORBIDDEN", '),
        (r'errors\.New\("DATABASE_[^"]*",\s*', lambda m: m.group(0).replace('errors.New(', 'errors.NewDatabaseError(')),
        (r'errors\.New\("INTERNAL_[^"]*",\s*', lambda m: m.group(0).replace('errors.New(', 'errors.NewInternalError(')),
        (r'errors\.New\("TIMEOUT[^"]*",\s*', lambda m: m.group(0).replace('errors.New(', 'errors.NewTimeoutError(')),
        
        # Generic errors.New("CODE", "message") -> NewInternalError as fallback
        (r'errors\.New\("([A-Z_]+)",\s*(["\w])', r'errors.NewInternalError("\1", \2'),
    ]
    
    for pattern, replacement in replacements:
        if callable(replacement):
            content = re.sub(pattern, replacement, content)
        else:
            content = re.sub(pattern, replacement, content)
    
    if content != original_content:
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        return True
    return False

def main():
    """Main function to fix all Go files."""
    base_dir = Path('/workspace/project/OpenEAAP')
    
    # Find all Go files
    go_files = list(base_dir.glob('**/*.go'))
    
    fixed_count = 0
    for go_file in go_files:
        if 'vendor' in str(go_file) or '.git' in str(go_file):
            continue
        
        try:
            if fix_errors_new_calls(go_file):
                print(f"Fixed: {go_file}")
                fixed_count += 1
        except Exception as e:
            print(f"Error fixing {go_file}: {e}")
    
    print(f"\nTotal files fixed: {fixed_count}")

if __name__ == '__main__':
    main()
