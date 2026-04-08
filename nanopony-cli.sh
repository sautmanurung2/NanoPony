#!/bin/bash
# nanopony - Proto file compiler CLI tool
# This script compiles .proto files into .pb.go files

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Execute the nanopony CLI
exec "${SCRIPT_DIR}/nanopony" "$@"
