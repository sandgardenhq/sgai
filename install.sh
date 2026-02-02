#!/usr/bin/env bash
#
# sgai installer - curl-pipe-bash installation script
# Usage: curl -fsSL https://raw.githubusercontent.com/sandgardenhq/sgai/main/install.sh | bash
#

set -euo pipefail

readonly GITHUB_REPO="sandgardenhq/sgai"
readonly BINARY_NAME="sgai"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TEMP_DIR=""

cleanup() {
    if [[ -n "${TEMP_DIR}" && -d "${TEMP_DIR}" ]]; then
        rm -rf "${TEMP_DIR}"
    fi
}

trap cleanup EXIT

print_success() {
    printf "${GREEN}✓${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}✗${NC} %s\n" "$1" >&2
}

print_warning() {
    printf "${YELLOW}⚠${NC} %s\n" "$1"
}

print_info() {
    printf "${BLUE}→${NC} %s\n" "$1"
}

show_help() {
    cat << EOF
sgai installer

Usage: ./install.sh [OPTIONS]

Options:
  --version <tag>       Install specific version (default: latest)
  --install-dir <path>  Install to specific directory (skips prompt)
  --uninstall           Uninstall sgai from system
  --help                Show this help message

Examples:
  ./install.sh                              # Install latest version interactively
  ./install.sh --version v1.0.0             # Install specific version
  ./install.sh --install-dir ~/.local/bin   # Install to specific directory
  curl -fsSL URL | bash -s -- --install-dir ~/.local/bin  # Scripted installation

EOF
}

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
            print_error "Unsupported operating system: ${os}"
            print_info "sgai currently supports macOS and Linux only"
            exit 1
            ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    
    case "${arch}" in
        x86_64|amd64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            print_error "Unsupported architecture: ${arch}"
            print_info "sgai currently supports amd64 and arm64 only"
            exit 1
            ;;
    esac
}

get_latest_version() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    local version
    
    print_info "Fetching latest version..."
    
    if command -v curl &> /dev/null; then
        version=$(curl -fsSL "${api_url}" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    elif command -v wget &> /dev/null; then
        version=$(wget -qO- "${api_url}" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi
    
    if [[ -z "${version}" ]]; then
        print_error "Failed to determine latest version"
        exit 1
    fi
    
    echo "${version}"
}

download_file() {
    local url="$1"
    local dest="$2"
    
    if command -v curl &> /dev/null; then
        curl -fsSL -o "${dest}" "${url}"
    elif command -v wget &> /dev/null; then
        wget -q -O "${dest}" "${url}"
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi
}

verify_checksum() {
    local binary_path="$1"
    local checksums_path="$2"
    local binary_filename="$3"
    
    print_info "Verifying checksum..."
    
    local expected_hash
    expected_hash=$(grep -E "^[a-f0-9]{64}[[:space:]]+(\\*)?${binary_filename}\$" "${checksums_path}" | head -1 | awk '{print $1}')
    
    if [[ -z "${expected_hash}" ]]; then
        expected_hash=$(grep "${binary_filename}" "${checksums_path}" | head -1 | awk '{print $1}')
    fi
    
    if [[ -z "${expected_hash}" || ${#expected_hash} -ne 64 ]]; then
        print_error "Valid checksum for ${binary_filename} not found in checksums.txt"
        exit 1
    fi
    
    local actual_hash
    if command -v sha256sum &> /dev/null; then
        actual_hash=$(sha256sum "${binary_path}" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        actual_hash=$(shasum -a 256 "${binary_path}" | awk '{print $1}')
    else
        print_error "Neither sha256sum nor shasum is available"
        exit 1
    fi
    
    if [[ "${expected_hash}" != "${actual_hash}" ]]; then
        print_error "Checksum verification failed!"
        print_error "Expected: ${expected_hash}"
        print_error "Actual:   ${actual_hash}"
        exit 1
    fi
    
    print_success "Checksum verified"
}

prompt_install_dir() {
    echo ""
    echo "Where would you like to install sgai?"
    echo ""
    echo "  1) /usr/local/bin (may require sudo)"
    echo "  2) ~/.local/bin (user-local, no sudo needed)"
    echo ""
    
    if [[ ! -t 0 ]]; then
        print_warning "Non-interactive mode detected, using default: ~/.local/bin"
        echo "${HOME}/.local/bin"
        return
    fi
    
    read -rp "Enter choice [1-2, default: 2]: " choice
    choice="${choice:-2}"
    
    case "${choice}" in
        1)
            echo "/usr/local/bin"
            ;;
        2|*)
            echo "${HOME}/.local/bin"
            ;;
    esac
}

check_existing_installation() {
    local install_dir="$1"
    local binary_path="${install_dir}/${BINARY_NAME}"
    
    if [[ -f "${binary_path}" ]]; then
        print_warning "sgai is already installed at ${binary_path}"
        
        if [[ ! -t 0 ]]; then
            print_info "Non-interactive mode: overwriting existing installation"
            return 0
        fi
        
        read -rp "Do you want to overwrite it? [y/N]: " response
        case "${response}" in
            [yY][eE][sS]|[yY])
                return 0
                ;;
            *)
                print_info "Installation cancelled"
                exit 0
                ;;
        esac
    fi
}

install_binary() {
    local version="$1"
    local install_dir="$2"
    local os="$3"
    local arch="$4"
    
    check_existing_installation "${install_dir}"
    
    local binary_filename="${BINARY_NAME}-${os}-${arch}"
    local binary_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${binary_filename}"
    local checksums_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/checksums.txt"
    
    TEMP_DIR=$(mktemp -d)
    local temp_binary="${TEMP_DIR}/${binary_filename}"
    local temp_checksums="${TEMP_DIR}/checksums.txt"
    
    print_info "Downloading sgai ${version} for ${os}/${arch}..."
    download_file "${checksums_url}" "${temp_checksums}"
    download_file "${binary_url}" "${temp_binary}"
    
    verify_checksum "${temp_binary}" "${temp_checksums}" "${binary_filename}"
    
    if [[ ! -d "${install_dir}" ]]; then
        print_info "Creating directory ${install_dir}..."
        mkdir -p "${install_dir}"
    fi
    
    local final_path="${install_dir}/${BINARY_NAME}"
    
    print_info "Installing to ${final_path}..."
    
    if [[ ! -w "${install_dir}" ]]; then
        print_info "Directory not writable, using sudo..."
        sudo cp "${temp_binary}" "${final_path}"
        sudo chmod +x "${final_path}"
    else
        cp "${temp_binary}" "${final_path}"
        chmod +x "${final_path}"
    fi
    
    echo ""
    print_success "sgai ${version} installed successfully!"
    echo ""
    
    if [[ ":${PATH}:" != *":${install_dir}:"* ]]; then
        print_warning "${install_dir} is not in your PATH"
        echo ""
        echo "Add it to your PATH by adding this line to your shell profile:"
        echo ""
        echo "  export PATH=\"${install_dir}:\$PATH\""
        echo ""
    fi
    
    echo "Run 'sgai --help' to get started."
}

uninstall_sgai() {
    local found=false
    local locations=("/usr/local/bin/${BINARY_NAME}" "${HOME}/.local/bin/${BINARY_NAME}")
    
    echo ""
    print_info "Searching for sgai installations..."
    echo ""
    
    for location in "${locations[@]}"; do
        if [[ -f "${location}" ]]; then
            found=true
            print_info "Found: ${location}"
            
            if [[ ! -t 0 ]]; then
                print_warning "Non-interactive mode: skipping ${location}"
                continue
            fi
            
            read -rp "Remove ${location}? [y/N]: " response
            case "${response}" in
                [yY][eE][sS]|[yY])
                    local parent_dir
                    parent_dir=$(dirname "${location}")
                    if [[ ! -w "${parent_dir}" ]]; then
                        sudo rm -f "${location}"
                    else
                        rm -f "${location}"
                    fi
                    print_success "Removed ${location}"
                    ;;
                *)
                    print_info "Skipped ${location}"
                    ;;
            esac
        fi
    done
    
    if [[ "${found}" == false ]]; then
        print_warning "No sgai installation found"
    else
        echo ""
        print_success "Uninstall complete"
    fi
}

main() {
    local version=""
    local install_dir=""
    local uninstall=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)
                if [[ -z "${2:-}" ]]; then
                    print_error "--version requires a value"
                    exit 1
                fi
                version="$2"
                shift 2
                ;;
            --install-dir)
                if [[ -z "${2:-}" ]]; then
                    print_error "--install-dir requires a value"
                    exit 1
                fi
                install_dir="$2"
                shift 2
                ;;
            --uninstall)
                uninstall=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo ""
                show_help
                exit 1
                ;;
        esac
    done
    
    echo ""
    echo "┌─────────────────────────────────────────┐"
    echo "│          sgai installer                 │"
    echo "└─────────────────────────────────────────┘"
    echo ""
    
    if [[ "${uninstall}" == true ]]; then
        uninstall_sgai
        exit 0
    fi
    
    local os
    local arch
    os=$(detect_os)
    arch=$(detect_arch)
    
    print_success "Detected platform: ${os}/${arch}"
    
    if [[ -z "${version}" ]]; then
        version=$(get_latest_version)
    fi
    
    print_success "Version: ${version}"
    
    if [[ -z "${install_dir}" ]]; then
        install_dir=$(prompt_install_dir)
    fi
    
    install_dir="${install_dir/#\~/$HOME}"
    
    print_success "Install directory: ${install_dir}"
    
    install_binary "${version}" "${install_dir}" "${os}" "${arch}"
}

main "$@"
