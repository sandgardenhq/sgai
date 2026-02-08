---
name: Safe Cleanup Trap
description: Automatic cleanup of temporary files on script exit
when_to_use: When creating scripts that create temporary files or directories
---

# Global variable for cleanup
TEMP_DIR=""

# Cleanup function that runs on exit
cleanup() {
    if [[ -n "${TEMP_DIR}" && -d "${TEMP_DIR}" ]]; then
        rm -rf "${TEMP_DIR}"
    fi
}

# Set up trap to call cleanup on exit
trap cleanup EXIT

# Example usage:
# TEMP_DIR="$(mktemp -d)"
# Files created in ${TEMP_DIR} will be automatically cleaned up