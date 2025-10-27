#!/bin/bash

# Script: collect_text.sh
# Description: Recursively collects all text from files with a specified extension
# and outputs them grouped under filenames

# Default values
EXTENSION=""
DIRECTORY="."
OUTPUT_FILE=""
SEPARATOR="================================"

# Help function
show_help() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Recursively collects text from files with a specified extension and groups them under filenames.

OPTIONS:
    -e, --extension EXT     File extension to search for (required, e.g., txt, md, log)
    -d, --directory DIR     Directory to search in (default: current directory)
    -o, --output FILE       Output file (default: prints to stdout)
    -s, --separator SEP     Custom separator between files (default: $SEPARATOR)
    -h, --help              Show this help message

EXAMPLES:
    $(basename "$0") -e txt
    $(basename "$0") -e md -d /path/to/docs
    $(basename "$0") -e log -o combined.txt
EOF
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--extension)
            EXTENSION="$2"
            shift 2
            ;;
        -d|--directory)
            DIRECTORY="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -s|--separator)
            SEPARATOR="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            ;;
        *)
            echo "Error: Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Validate required arguments
if [[ -z "$EXTENSION" ]]; then
    echo "Error: Extension is required (-e or --extension)"
    echo "Use -h or --help for usage information"
    exit 1
fi

# Remove leading dot from extension if present
EXTENSION="${EXTENSION#.}"

# Check if directory exists
if [[ ! -d "$DIRECTORY" ]]; then
    echo "Error: Directory '$DIRECTORY' does not exist"
    exit 1
fi

# Function to process files
process_files() {
    # Find all files recursively and sort them
    local files
    files=$(find "$DIRECTORY" -type f -name "*.$EXTENSION" | sort)
    
    if [[ -z "$files" ]]; then
        echo "No files with extension '.$EXTENSION' found in '$DIRECTORY' (searched recursively)"
        exit 0
    fi
    
    local file_count=0
    
    # Process each file
    while IFS= read -r file; do
        ((file_count++))
        
        # Print separator (except for first file)
        if [[ $file_count -gt 1 ]]; then
            echo ""
            echo "$SEPARATOR"
            echo ""
        fi
        
        # Print filename header
        echo "FILE: $file"
        echo "$SEPARATOR"
        echo ""
        
        # Print file contents
        if [[ -r "$file" ]]; then
            cat "$file"
        else
            echo "[Error: Cannot read file]"
        fi
        
        echo ""
        
    done <<< "$files"
    
    # Print summary
    echo ""
    echo "$SEPARATOR"
    echo "Total files processed: $file_count"
}

# Execute and handle output
if [[ -n "$OUTPUT_FILE" ]]; then
    process_files > "$OUTPUT_FILE"
    echo "Output written to: $OUTPUT_FILE"
else
    process_files
fi

