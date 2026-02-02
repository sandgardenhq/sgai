---
name: Cross-Platform Detection
description: Detect OS (darwin/linux) and architecture (amd64/arm64) with fallbacks
when_to_use: When creating shell scripts that need to run on multiple platforms
---

# Detect operating system
detect_os() {
    local os
    os="$(uname -s)"
    
    case "${os}" in
        Darwin)
            echo "darwin"
            ;;
        Linux)
            echo "linux"
            ;;
        *)
            echo "Unsupported operating system: ${os}" >&2
            echo "Script currently supports macOS and Linux only" >&2
            exit 1
            ;;
    esac
}

# Detect system architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    
    case "${arch}" in
        x86_64)
            echo "amd64"
            ;;
        amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l)
            echo "arm64"
            ;;
        *)
            echo "Unsupported architecture: ${arch}" >&2
            exit 1
            ;;
    esac
}