#!/usr/bin/env python3
"""
Script to fix logger API mismatches in OpenEAAP codebase.
Converts logger.Info(ctx, "message", "key", value) to logger.WithContext(ctx).Info("message", logging.String("key", value))
"""

import re
import os
import sys
from pathlib import Path

def fix_logger_call(line, imports_needed):
    """Fix a single logger call line."""
    # Pattern: logger.Method(ctx, "message", "key1", val1, "key2", val2, ...)
    # Should become: logger.WithContext(ctx).Method("message", logging.String("key1", val1), logging.String("key2", val2))
    
    # Match logger method calls with ctx as first parameter
    pattern = r'(\w+\.(?:logger|Logger))\.(Debug|Info|Warn|Error)\(ctx,\s*"([^"]+)"(.*?)\)'
    match = re.search(pattern, line)
    
    if not match:
        # Try without quotes around message (variable message)
        pattern = r'(\w+\.(?:logger|Logger))\.(Debug|Info|Warn|Error)\(ctx,\s*(\w+)(.*?)\)'
        match = re.search(pattern, line)
    
    if match:
        logger_obj = match.group(1)
        method = match.group(2)
        message = match.group(3)
        rest = match.group(4).strip()
        
        # Parse the remaining arguments (key-value pairs)
        fields = []
        if rest:
            rest = rest.lstrip(',').strip()
            # Split by comma but not within parentheses or quotes
            args = split_args(rest.rstrip(')'))
            
            i = 0
            while i < len(args):
                key = args[i].strip().strip('"')
                if i + 1 < len(args):
                    value = args[i + 1].strip()
                    # Determine the field type
                    field = convert_to_field(key, value)
                    fields.append(field)
                    imports_needed.add('logging')
                    i += 2
                else:
                    i += 1
        
        # Build the new call
        if '"' in message or "'" in message:
            msg_str = message
        else:
            msg_str = f'"{message}"'
        
        if fields:
            fields_str = ', ' + ', '.join(fields)
        else:
            fields_str = ''
        
        new_call = f'{logger_obj}.WithContext(ctx).{method}({msg_str}{fields_str})'
        
        # Replace in line
        indent = len(line) - len(line.lstrip())
        new_line = ' ' * indent + new_call
        if line.rstrip().endswith(')'):
            new_line += ')'
        new_line += '\n'
        return new_line
    
    return line

def split_args(args_str):
    """Split arguments by comma, respecting quotes and parentheses."""
    args = []
    current = ''
    depth = 0
    in_quotes = False
    quote_char = None
    
    for char in args_str:
        if char in '"\'':
            if not in_quotes:
                in_quotes = True
                quote_char = char
            elif char == quote_char:
                in_quotes = False
        elif char in '({[':
            depth += 1
        elif char in ')}]':
            depth -= 1
        elif char == ',' and depth == 0 and not in_quotes:
            args.append(current.strip())
            current = ''
            continue
        current += char
    
    if current.strip():
        args.append(current.strip())
    
    return args

def convert_to_field(key, value):
    """Convert a key-value pair to the appropriate logging.Field call."""
    value = value.strip()
    
    # Check if it's a variable of type error
    if value == 'err' or value.endswith('Error') or value.endswith('Err'):
        return f'logging.Error({value})'
    
    # Check if it's a string literal
    if value.startswith('"') or value.startswith("'"):
        return f'logging.String("{key}", {value})'
    
    # Check if it looks like a number
    if value.isdigit() or (value.startswith('-') and value[1:].isdigit()):
        return f'logging.Int("{key}", {value})'
    
    # Check if it's a duration
    if 'time.Since' in value or 'Duration' in value or value.endswith('*time.Millisecond'):
        return f'logging.Duration("{key}", {value})'
    
    # Check if it's a boolean
    if value in ['true', 'false']:
        return f'logging.Bool("{key}", {value})'
    
    # Default: use Any for complex types
    return f'logging.String("{key}", fmt.Sprintf("%v", {value}))'

def fix_file(filepath):
    """Fix logger API calls in a single file."""
    print(f"Processing {filepath}...")
    
    with open(filepath, 'r') as f:
        lines = f.readlines()
    
    imports_needed = set()
    new_lines = []
    modified = False
    
    for line in lines:
        # Skip if line doesn't contain logger method call with ctx
        if 'logger.' in line.lower() and '(ctx,' in line:
            new_line = fix_logger_call(line, imports_needed)
            if new_line != line:
                modified = True
                new_lines.append(new_line)
            else:
                new_lines.append(line)
        else:
            new_lines.append(line)
    
    if modified:
        # Check if logging import exists
        has_logging_import = any('github.com/openeeap/openeeap/internal/observability/logging' in line for line in new_lines)
        
        if 'logging' in imports_needed and not has_logging_import:
            # Add import if needed
            for i, line in enumerate(new_lines):
                if line.strip().startswith('import ('):
                    # Add to existing import block
                    new_lines.insert(i + 1, '\t"github.com/openeeap/openeeap/internal/observability/logging"\n')
                    break
                elif line.strip() == 'import':
                    # Single import, convert to block
                    new_lines[i] = 'import (\n'
                    new_lines.insert(i + 1, '\t"github.com/openeeap/openeeap/internal/observability/logging"\n')
                    # Find the next line and add closing paren
                    for j in range(i + 2, len(new_lines)):
                        if new_lines[j].strip() and not new_lines[j].strip().startswith('//'):
                            new_lines.insert(j, ')\n')
                            break
                    break
        
        with open(filepath, 'w') as f:
            f.writelines(new_lines)
        
        print(f"  ✓ Fixed {filepath}")
        return True
    
    return False

def main():
    """Main function to fix all files."""
    # Get all Go files in internal and pkg directories
    base_dir = Path('.')
    go_files = []
    
    for pattern in ['internal/**/*.go', 'pkg/**/*.go']:
        go_files.extend(base_dir.glob(pattern))
    
    print(f"Found {len(go_files)} Go files")
    
    fixed_count = 0
    for filepath in go_files:
        if fix_file(filepath):
            fixed_count += 1
    
    print(f"\n✓ Fixed {fixed_count} files")

if __name__ == '__main__':
    main()
