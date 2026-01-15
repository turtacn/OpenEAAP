#!/bin/bash
# OpenEAAP Build Script
# Compiles Go applications and generates binary files
# Supports cross-compilation for Linux, macOS, and Windows

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project metadata
PROJECT_NAME="openeeap"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(go version | awk '{print $3}')

# Directories
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
BUILD_DIR="${ROOT_DIR}/build"
BIN_DIR="${BUILD_DIR}/bin"
DIST_DIR="${BUILD_DIR}/dist"

# Build targets
TARGETS=(
   "cmd/server"
   "cmd/cli"
   "cmd/worker"
)

# Platform configurations for cross-compilation
PLATFORMS=(
   "linux/amd64"
   "linux/arm64"
   "darwin/amd64"
   "darwin/arm64"
   "windows/amd64"
)

# Build flags
LDFLAGS="-s -w \
   -X github.com/${PROJECT_NAME}/${PROJECT_NAME}/pkg/version.Version=${VERSION} \
   -X github.com/${PROJECT_NAME}/${PROJECT_NAME}/pkg/version.BuildTime=${BUILD_TIME} \
   -X github.com/${PROJECT_NAME}/${PROJECT_NAME}/pkg/version.GitCommit=${GIT_COMMIT} \
   -X github.com/${PROJECT_NAME}/${PROJECT_NAME}/pkg/version.GoVersion=${GO_VERSION}"

# Build tags
BUILD_TAGS="netgo"

# CGO settings
export CGO_ENABLED=0

# Functions
log_info() {
   echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
   echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
   echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
   echo -e "${RED}[ERROR]${NC} $1"
}

# Clean build artifacts
clean() {
   log_info "Cleaning build artifacts..."
   rm -rf "${BUILD_DIR}"
   log_success "Build artifacts cleaned"
}

# Setup build directories
setup_dirs() {
   log_info "Setting up build directories..."
   mkdir -p "${BIN_DIR}"
   mkdir -p "${DIST_DIR}"
   log_success "Build directories created"
}

# Install dependencies
install_deps() {
   log_info "Installing Go dependencies..."
   cd "${ROOT_DIR}"
   go mod download
   go mod verify
   log_success "Dependencies installed and verified"
}

# Generate code (protobuf, mocks, etc.)
generate() {
   log_info "Generating code..."
   cd "${ROOT_DIR}"

   # Generate protobuf code
   if [ -d "api/proto" ]; then
       log_info "Generating protobuf code..."
       bash scripts/generate.sh protobuf
   fi

   # Generate mocks
   if command -v mockgen &> /dev/null; then
       log_info "Generating mocks..."
       bash scripts/generate.sh mocks
   fi

   log_success "Code generation completed"
}

# Run tests before build
test() {
   log_info "Running tests..."
   cd "${ROOT_DIR}"
   go test -v -race -coverprofile=coverage.out ./...

   if [ $? -eq 0 ]; then
       log_success "All tests passed"
   else
       log_error "Tests failed"
       exit 1
   fi
}

# Build for current platform
build_local() {
   log_info "Building for current platform..."
   cd "${ROOT_DIR}"

   for target in "${TARGETS[@]}"; do
       app_name=$(basename "${target}")
       log_info "Building ${app_name}..."

       go build \
           -tags="${BUILD_TAGS}" \
           -ldflags="${LDFLAGS}" \
           -o "${BIN_DIR}/${app_name}" \
           "./${target}"

       if [ $? -eq 0 ]; then
           log_success "${app_name} built successfully"

           # Display binary info
           binary="${BIN_DIR}/${app_name}"
           size=$(du -h "${binary}" | cut -f1)
           log_info "  Location: ${binary}"
           log_info "  Size: ${size}"
       else
           log_error "Failed to build ${app_name}"
           exit 1
       fi
   done
}

# Build for all platforms (cross-compilation)
build_all() {
   log_info "Building for all platforms..."
   cd "${ROOT_DIR}"

   for target in "${TARGETS[@]}"; do
       app_name=$(basename "${target}")

       for platform in "${PLATFORMS[@]}"; do
           IFS='/' read -r os arch <<< "${platform}"

           output_name="${app_name}-${os}-${arch}"
           if [ "${os}" = "windows" ]; then
               output_name="${output_name}.exe"
           fi

           log_info "Building ${app_name} for ${os}/${arch}..."

           GOOS="${os}" GOARCH="${arch}" go build \
               -tags="${BUILD_TAGS}" \
               -ldflags="${LDFLAGS}" \
               -o "${DIST_DIR}/${output_name}" \
               "./${target}"

           if [ $? -eq 0 ]; then
               size=$(du -h "${DIST_DIR}/${output_name}" | cut -f1)
               log_success "  ${output_name} (${size})"
           else
               log_error "Failed to build ${app_name} for ${os}/${arch}"
               exit 1
           fi
       done
   done
}

# Create distribution packages
package() {
   log_info "Creating distribution packages..."
   cd "${DIST_DIR}"

   for target in "${TARGETS[@]}"; do
       app_name=$(basename "${target}")

       for platform in "${PLATFORMS[@]}"; do
           IFS='/' read -r os arch <<< "${platform}"

           binary_name="${app_name}-${os}-${arch}"
           if [ "${os}" = "windows" ]; then
               binary_name="${binary_name}.exe"
           fi

           if [ -f "${binary_name}" ]; then
               package_name="${PROJECT_NAME}-${app_name}-${VERSION}-${os}-${arch}"

               log_info "Packaging ${package_name}..."

               # Create package directory
               mkdir -p "${package_name}"
               cp "${binary_name}" "${package_name}/"

               # Copy additional files
               if [ -f "${ROOT_DIR}/README.md" ]; then
                   cp "${ROOT_DIR}/README.md" "${package_name}/"
               fi

               if [ -f "${ROOT_DIR}/LICENSE" ]; then
                   cp "${ROOT_DIR}/LICENSE" "${package_name}/"
               fi

               # Copy default config
               if [ -f "${ROOT_DIR}/configs/default.yaml" ]; then
                   mkdir -p "${package_name}/configs"
                   cp "${ROOT_DIR}/configs/default.yaml" "${package_name}/configs/"
               fi

               # Create archive
               if [ "${os}" = "windows" ]; then
                   zip -q -r "${package_name}.zip" "${package_name}"
                   log_success "  ${package_name}.zip created"
               else
                   tar czf "${package_name}.tar.gz" "${package_name}"
                   log_success "  ${package_name}.tar.gz created"
               fi

               # Clean up package directory
               rm -rf "${package_name}"
           fi
       done
   done
}

# Generate checksums
checksum() {
   log_info "Generating checksums..."
   cd "${DIST_DIR}"

   if command -v sha256sum &> /dev/null; then
       sha256sum * > checksums.txt
       log_success "Checksums saved to checksums.txt"
   elif command -v shasum &> /dev/null; then
       shasum -a 256 * > checksums.txt
       log_success "Checksums saved to checksums.txt"
   else
       log_warn "sha256sum/shasum not found, skipping checksum generation"
   fi
}

# Build Docker images
docker_build() {
   log_info "Building Docker images..."
   cd "${ROOT_DIR}"

   for target in "${TARGETS[@]}"; do
       app_name=$(basename "${target}")
       image_name="${PROJECT_NAME}-${app_name}:${VERSION}"

       log_info "Building Docker image: ${image_name}..."

       docker build \
           --build-arg APP_NAME="${app_name}" \
           --build-arg VERSION="${VERSION}" \
           --build-arg BUILD_TIME="${BUILD_TIME}" \
           --build-arg GIT_COMMIT="${GIT_COMMIT}" \
           -t "${image_name}" \
           -f "deployments/docker/Dockerfile" \
           .

       if [ $? -eq 0 ]; then
           log_success "Docker image ${image_name} built successfully"

           # Tag as latest
           docker tag "${image_name}" "${PROJECT_NAME}-${app_name}:latest"
           log_success "Tagged as ${PROJECT_NAME}-${app_name}:latest"
       else
           log_error "Failed to build Docker image for ${app_name}"
           exit 1
       fi
   done
}

# Display build information
info() {
   log_info "Build Information:"
   echo "  Project: ${PROJECT_NAME}"
   echo "  Version: ${VERSION}"
   echo "  Build Time: ${BUILD_TIME}"
   echo "  Git Commit: ${GIT_COMMIT}"
   echo "  Go Version: ${GO_VERSION}"
   echo "  Build Dir: ${BUILD_DIR}"
}

# Show usage
usage() {
   cat << EOF
OpenEAAP Build Script

Usage: $0 [COMMAND]

Commands:
   clean       Clean build artifacts
   deps        Install dependencies
   generate    Generate code (protobuf, mocks)
   test        Run tests
   build       Build for current platform (default)
   all         Build for all platforms
   package     Create distribution packages
   checksum    Generate checksums
   docker      Build Docker images
   info        Display build information
   help        Show this help message

Examples:
   $0              # Build for current platform
   $0 all          # Build for all platforms
   $0 clean build  # Clean and build
   $0 test all     # Run tests and build all

EOF
}

# Main execution
main() {
   cd "${ROOT_DIR}"

   # Default to build if no arguments
   if [ $# -eq 0 ]; then
       setup_dirs
       install_deps
       build_local
       log_success "Build completed successfully!"
       exit 0
   fi

   # Process commands
   for cmd in "$@"; do
       case "${cmd}" in
           clean)
               clean
               ;;
           deps|dependencies)
               install_deps
               ;;
           generate|gen)
               generate
               ;;
           test)
               test
               ;;
           build)
               setup_dirs
               build_local
               ;;
           all|cross)
               setup_dirs
               build_all
               ;;
           package|pkg)
               package
               ;;
           checksum|sum)
               checksum
               ;;
           docker)
               docker_build
               ;;
           info)
               info
               ;;
           help|--help|-h)
               usage
               exit 0
               ;;
           *)
               log_error "Unknown command: ${cmd}"
               usage
               exit 1
               ;;
       esac
   done

   log_success "All operations completed successfully!"
}

# Run main function
main "$@"

# Personal.AI order the ending

