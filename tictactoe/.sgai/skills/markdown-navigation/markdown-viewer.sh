#!/bin/bash

# markdown-viewer.sh - Tool for navigating large markdown files
# Usage: ./markdown-viewer.sh <file.md> [section name]

if [ $# -lt 1 ]; then
    echo "Usage: $0 <file.md> [section name]"
    exit 1
fi

file="$1"
section="$2"

if [ ! -f "$file" ]; then
    echo "File not found: $file"
    exit 1
fi

if [ -z "$section" ]; then
    # Outline mode: print hierarchical list of headers
    grep '^#' "$file" | while read -r line; do
        level=$(echo "$line" | sed 's/^\(#*\).*/\1/' | wc -c)
        level=$((level - 1))
        title=$(echo "$line" | sed 's/^#* *//')
        indent=$(printf '%*s' $(((level - 1) * 2)) '')
        echo "${indent}- ${title}"
    done
else
    # Section mode: extract body of specific section
    # Find the line number of the section header
    section_line=$(grep -n "^##* $section$" "$file" | head -1 | cut -d: -f1)
    if [ -z "$section_line" ]; then
        echo "Section '$section' not found in $file"
        exit 1
    fi
    
    # Find the next header at the same level or higher
    header_level=$(sed -n "${section_line}p" "$file" | sed 's/^\(#*\).*/\1/' | wc -c)
    header_level=$((header_level - 1))
    
    next_header_line=$(tail -n +$((section_line + 1)) "$file" | grep -E -n "^#{1,$header_level} " | head -1 | cut -d: -f1)
    
    if [ -z "$next_header_line" ]; then
        # No next header, print to end
        sed -n "$((section_line + 1)),$ p" "$file"
    else
        # Print from after section header to before next header
        end_line=$((section_line + next_header_line - 1))
        sed -n "$((section_line + 1)),${end_line} p" "$file"
    fi
fi
