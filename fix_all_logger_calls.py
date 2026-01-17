#!/usr/bin/env python3
"""
Comprehensive script to fix all logger API calls in the codebase.
Converts logger.Method(ctx, "message", "key", value, ...) to 
logger.WithContext(ctx).Method("message", logging.Type("key", value), ...)
"""

import re
import sys
from pathlib import Path
from typing import List, Tuple

def infer_field_type(value: str) -> str:
    """Infer the appropriate logging field type for a value."""
    value = value.strip()
    
    # Error type
    if value in ['err', 'error'] or value.endswith('Err') or value.endswith('Error'):
        return 'Error'
    
    # Boolean literals
    if value in ['true', 'false']:
        return 'Bool'
    
    # Numeric literals
    if value.isdigit() or (value.startswith('-') and value[1:].isdigit()):
        return 'Int'
    
    # Float literals
    if '.' in value and value.replace('.', '').replace('-', '').isdigit():
        return 'Float64'
    
    # Duration expressions
    if 'time.Since' in value or 'Duration' in value or 'time.Millisecond' in value:
        return 'Duration'
    
    # String literals
    if value.startswith('"') or value.startswith("'"):
        return 'String'
    
    # Default: use String with fmt.Sprintf for safety
    return 'Any'

def parse_logger_args(args_str: str) -> List[Tuple[str, str]]:
    """Parse key-value pairs from logger arguments."""
    pairs = []
    tokens = []
    current = ''
    depth = 0
    in_quotes = False
    quote_char = None
    
    # Tokenize by comma, respecting nesting
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
            tokens.append(current.strip())
            current = ''
            continue
        current += char
    
    if current.strip():
        tokens.append(current.strip())
    
    # Group into key-value pairs
    i = 0
    while i < len(tokens):
        if i + 1 < len(tokens):
            key = tokens[i].strip('"\'')
            value = tokens[i + 1].strip()
            pairs.append((key, value))
            i += 2
        else:
            i += 1
    
    return pairs

def convert_logger_call(match, line_num: int) -> str:
    """Convert a single logger call to the new API."""
    logger_obj = match.group(1)
    method = match.group(2)
    message = match.group(3)
    rest = match.group(4).strip() if match.group(4) else ''
    
    # Handle message - could be quoted string or variable
    if message.startswith('"') or message.startswith("'"):
        msg_str = message
    else:
        # Variable name, keep as is
        msg_str = message
    
    # Parse arguments if any
    fields = []
    if rest:
        rest = rest.lstrip(',').strip().rstrip(')')
        if rest:
            pairs = parse_logger_args(rest)
            for key, value in pairs:
                field_type = infer_field_type(value)
                
                if field_type == 'Error':
                    fields.append(f'logging.Error({value})')
                elif field_type == 'Bool':
                    fields.append(f'logging.Bool("{key}", {value})')
                elif field_type == 'Int':
                    fields.append(f'logging.Int("{key}", {value})')
                elif field_type == 'Float64':
                    fields.append(f'logging.Float64("{key}", {value})')
                elif field_type == 'Duration':
                    fields.append(f'logging.Duration("{key}", {value})')
                elif field_type == 'String':
                    fields.append(f'logging.String("{key}", {value})')
                else:
                    # Use Any for complex expressions
                    fields.append(f'logging.Any("{key}", {value})')
    
    # Build the new call
    if fields:
        fields_str = ', ' + ', '.join(fields)
    else:
        fields_str = ''
    
    new_call = f'{logger_obj}.WithContext(ctx).{method}({msg_str}{fields_str})'
    return new_call

def fix_logger_calls_in_file(filepath: Path) -> Tuple[bool, int]:
    """Fix all logger calls in a file. Returns (modified, count)."""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading {filepath}: {e}")
        return False, 0
    
    original = content
    lines = content.split('\n')
    fixed_lines = []
    fixes_count = 0
    i = 0
    
    while i < len(lines):
        line = lines[i]
        
        # Match single-line logger calls
        pattern = r'(\w+\.logger)\.(Debug|Info|Warn|Error)\(ctx,\s*(["\'].*?["\']|\w+)(.*?)\)(?:\s*$)'
        match = re.search(pattern, line)
        
        if match and 'WithContext' not in line:
            try:
                new_line = re.sub(pattern, lambda m: convert_logger_call(m, i+1), line)
                fixed_lines.append(new_line)
                fixes_count += 1
            except Exception as e:
                # If conversion fails, keep original
                fixed_lines.append(line)
        else:
            # Check for multi-line logger calls
            if re.search(r'(\w+\.logger)\.(Debug|Info|Warn|Error)\(ctx,', line) and 'WithContext' not in line:
                # Multi-line call - collect all lines
                call_lines = [line]
                j = i + 1
                paren_count = line.count('(') - line.count(')')
                
                while j < len(lines) and paren_count > 0:
                    call_lines.append(lines[j])
                    paren_count += lines[j].count('(') - lines[j].count(')')
                    j += 1
                
                # Join and try to fix
                full_call = ' '.join(call_lines)
                match = re.search(r'(\w+\.logger)\.(Debug|Info|Warn|Error)\(ctx,\s*(["\'].*?["\']|\w+)(.*?)\)', full_call)
                
                if match:
                    try:
                        new_call = convert_logger_call(match, i+1)
                        # Add to fixed_lines as single line with proper indentation
                        indent = len(line) - len(line.lstrip())
                        fixed_lines.append(' ' * indent + new_call)
                        fixes_count += 1
                        i = j - 1  # Skip the lines we processed
                    except Exception as e:
                        # Keep original if conversion fails
                        fixed_lines.extend(call_lines)
                        i = j - 1
                else:
                    fixed_lines.append(line)
            else:
                fixed_lines.append(line)
        
        i += 1
    
    content = '\n'.join(fixed_lines)
    
    # Ensure logging import exists if we made changes
    if fixes_count > 0:
        if 'github.com/openeeap/openeeap/internal/observability/logging' not in content:
            # Add import
            import_pattern = r'(import\s+\()'
            if re.search(import_pattern, content):
                content = re.sub(
                    import_pattern,
                    r'\1\n\t"github.com/openeeap/openeeap/internal/observability/logging"',
                    content,
                    count=1
                )
    
    if content != original:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        return True, fixes_count
    
    return False, 0

def main():
    """Main function to fix all files."""
    target_files = [
        'internal/api/http/handler/agent_handler.go',
        'internal/api/http/handler/model_handler.go',
        'internal/api/http/handler/workflow_handler.go',
        'internal/api/http/middleware/auth.go',
        'internal/api/http/middleware/ratelimit.go',
        'internal/app/service/agent_service.go',
        'internal/governance/audit/audit_logger.go',
        'internal/governance/audit/audit_query.go',
        'internal/governance/compliance/compliance_checker.go',
        'internal/governance/policy/pdp.go',
        'internal/governance/policy/pep.go',
        'internal/governance/policy/policy_loader.go',
        'internal/platform/inference/cache/cache_manager.go',
        'internal/platform/inference/cache/l1_local.go',
        'internal/platform/inference/cache/l2_redis.go',
        'internal/platform/inference/cache/l3_vector.go',
        'internal/platform/inference/gateway.go',
        'internal/platform/inference/privacy/privacy_gateway.go',
        'internal/platform/inference/router.go',
        'internal/platform/inference/vllm/vllm_client.go',
        'internal/platform/learning/feedback_collector.go',
        'internal/platform/learning/learning_engine.go',
        'internal/platform/learning/optimizer.go',
        'internal/platform/rag/generator.go',
        'internal/platform/rag/rag_engine.go',
        'internal/platform/rag/reranker.go',
        'internal/platform/rag/retriever.go',
        'internal/platform/training/dpo/dpo_trainer.go',
        'internal/platform/training/rlhf/rlhf_trainer.go',
        'internal/platform/training/training_service.go',
    ]
    
    total_files = 0
    total_fixes = 0
    
    for filepath in target_files:
        path = Path(filepath)
        if path.exists():
            modified, count = fix_logger_calls_in_file(path)
            if modified:
                total_files += 1
                total_fixes += count
                print(f"✓ Fixed {filepath} ({count} calls)")
            else:
                print(f"  Skipped {filepath} (no changes needed)")
        else:
            print(f"✗ Not found: {filepath}")
    
    print(f"\n{'='*60}")
    print(f"Fixed {total_fixes} logger calls in {total_files} files")
    print(f"{'='*60}")

if __name__ == '__main__':
    main()
