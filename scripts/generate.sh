#!/bin/bash

################################################################################
# OpenEAAP Code Generation Script
#
# This script generates various code artifacts including:
# - gRPC code from proto definitions
# - OpenAPI client SDKs
# - Mock implementations for testing
#
# Usage: ./scripts/generate.sh [options]
# Options:
#   --proto        Generate gRPC code only
#   --openapi      Generate OpenAPI clients only
#   --mocks        Generate mocks only
#   --all          Generate all (default)
#   --clean        Clean generated files before generation
################################################################################

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Directories
PROTO_DIR="api/proto"
OPENAPI_DIR="api/openapi"
GENERATED_DIR="internal/generated"
MOCK_DIR="internal/mocks"

# Tools versions
PROTOC_VERSION="24.4"
PROTOC_GEN_GO_VERSION="v1.31.0"
PROTOC_GEN_GO_GRPC_VERSION="v1.3.0"
OPENAPI_GENERATOR_VERSION="7.1.0"
MOCKGEN_VERSION="v0.4.0"

################################################################################
# Utility Functions
################################################################################

log_info() {
   echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
   echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
   echo -e "${RED}[ERROR]${NC} $1"
}

check_command() {
   if ! command -v "$1" &> /dev/null; then
       log_error "$1 is not installed. Please install it first."
       return 1
   fi
   return 0
}

################################################################################
# Tool Installation
################################################################################

install_tools() {
   log_info "Checking and installing required tools..."

   # Check Go installation
   if ! check_command go; then
       log_error "Go is not installed. Please install Go 1.24+ first."
       exit 1
   fi

   # Install protoc
   if ! check_command protoc; then
       log_warn "protoc not found. Installing protoc ${PROTOC_VERSION}..."

       OS="$(uname -s)"
       ARCH="$(uname -m)"

       case "$OS" in
           Linux*)
               PROTOC_OS="linux"
               ;;
           Darwin*)
               PROTOC_OS="osx"
               ;;
           *)
               log_error "Unsupported OS: $OS"
               exit 1
               ;;
       esac

       case "$ARCH" in
           x86_64)
               PROTOC_ARCH="x86_64"
               ;;
           arm64|aarch64)
               PROTOC_ARCH="aarch_64"
               ;;
           *)
               log_error "Unsupported architecture: $ARCH"
               exit 1
               ;;
       esac

       PROTOC_ZIP="protoc-${PROTOC_VERSION}-${PROTOC_OS}-${PROTOC_ARCH}.zip"
       PROTOC_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}"

       curl -LO "$PROTOC_URL"
       unzip -o "$PROTOC_ZIP" -d /usr/local bin/protoc
       unzip -o "$PROTOC_ZIP" -d /usr/local 'include/*'
       rm -f "$PROTOC_ZIP"

       log_info "protoc installed successfully"
   fi

   # Install protoc-gen-go
   if ! check_command protoc-gen-go; then
       log_warn "protoc-gen-go not found. Installing..."
       go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}
   fi

   # Install protoc-gen-go-grpc
   if ! check_command protoc-gen-go-grpc; then
       log_warn "protoc-gen-go-grpc not found. Installing..."
       go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}
   fi

   # Install mockgen
   if ! check_command mockgen; then
       log_warn "mockgen not found. Installing..."
       go install go.uber.org/mock/mockgen@${MOCKGEN_VERSION}
   fi

   # Check openapi-generator-cli
   if ! check_command openapi-generator-cli; then
       log_warn "openapi-generator-cli not found. Installing via npm..."
       if check_command npm; then
           npm install -g @openapitools/openapi-generator-cli@${OPENAPI_GENERATOR_VERSION}
       else
           log_warn "npm not found. Skipping OpenAPI client generation."
       fi
   fi

   log_info "All required tools are ready"
}

################################################################################
# gRPC Code Generation
################################################################################

generate_grpc() {
   log_info "Generating gRPC code from proto definitions..."

   # Create output directory
   mkdir -p "${GENERATED_DIR}/proto"

   # Find all proto files
   PROTO_FILES=$(find "${PROTO_DIR}" -name "*.proto")

   if [ -z "$PROTO_FILES" ]; then
       log_warn "No proto files found in ${PROTO_DIR}"
       return
   fi

   # Generate Go code for each proto file
   for proto_file in $PROTO_FILES; do
       log_info "Processing ${proto_file}..."

       protoc \
           --proto_path="${PROTO_DIR}" \
           --go_out="${GENERATED_DIR}/proto" \
           --go_opt=paths=source_relative \
           --go-grpc_out="${GENERATED_DIR}/proto" \
           --go-grpc_opt=paths=source_relative \
           "${proto_file}"

       if [ $? -eq 0 ]; then
           log_info "✓ Generated code for ${proto_file}"
       else
           log_error "✗ Failed to generate code for ${proto_file}"
           exit 1
       fi
   done

   log_info "gRPC code generation completed successfully"
}

################################################################################
# OpenAPI Client Generation
################################################################################

generate_openapi_clients() {
   log_info "Generating OpenAPI clients..."

   # Check if openapi-generator-cli is available
   if ! check_command openapi-generator-cli; then
       log_warn "openapi-generator-cli not found. Skipping OpenAPI client generation."
       return
   fi

   OPENAPI_SPEC="${OPENAPI_DIR}/openapi.yaml"

   if [ ! -f "$OPENAPI_SPEC" ]; then
       log_warn "OpenAPI spec not found at ${OPENAPI_SPEC}"
       return
   fi

   # Generate Go client
   log_info "Generating Go client..."
   mkdir -p "${GENERATED_DIR}/openapi/go"

   openapi-generator-cli generate \
       -i "${OPENAPI_SPEC}" \
       -g go \
       -o "${GENERATED_DIR}/openapi/go" \
       --package-name openeeap \
       --git-user-id openeeap \
       --git-repo-id openeeap-go-client

   # Generate Python client
   log_info "Generating Python client..."
   mkdir -p "${GENERATED_DIR}/openapi/python"

   openapi-generator-cli generate \
       -i "${OPENAPI_SPEC}" \
       -g python \
       -o "${GENERATED_DIR}/openapi/python" \
       --package-name openeeap

   # Generate TypeScript client
   log_info "Generating TypeScript client..."
   mkdir -p "${GENERATED_DIR}/openapi/typescript"

   openapi-generator-cli generate \
       -i "${OPENAPI_SPEC}" \
       -g typescript-axios \
       -o "${GENERATED_DIR}/openapi/typescript"

   log_info "OpenAPI client generation completed successfully"
}

################################################################################
# Mock Generation
################################################################################

generate_mocks() {
   log_info "Generating mock implementations..."

   mkdir -p "${MOCK_DIR}"

   # Define interfaces to mock
   declare -A MOCK_TARGETS=(
       ["internal/domain/agent/repository.go"]="AgentRepository"
       ["internal/domain/workflow/repository.go"]="WorkflowRepository"
       ["internal/domain/model/repository.go"]="ModelRepository"
       ["internal/domain/knowledge/repository.go"]="DocumentRepository,ChunkRepository"
       ["internal/infrastructure/vector/interface.go"]="VectorStore"
       ["internal/infrastructure/storage/interface.go"]="ObjectStore"
       ["internal/infrastructure/message/interface.go"]="MessageQueue"
       ["internal/platform/orchestrator/orchestrator.go"]="Orchestrator"
       ["internal/platform/runtime/interface.go"]="PluginRuntime"
       ["internal/platform/inference/gateway.go"]="InferenceGateway"
       ["internal/platform/inference/router.go"]="ModelRouter"
       ["internal/platform/rag/rag_engine.go"]="RAGEngine"
       ["internal/platform/learning/learning_engine.go"]="OnlineLearningEngine"
       ["internal/platform/training/training_service.go"]="TrainingService"
       ["internal/governance/policy/pdp.go"]="PolicyDecisionPoint"
       ["internal/governance/audit/audit_logger.go"]="AuditLogger"
   )

   for source_file in "${!MOCK_TARGETS[@]}"; do
       if [ ! -f "$source_file" ]; then
           log_warn "Source file not found: ${source_file}, skipping..."
           continue
       fi

       interfaces="${MOCK_TARGETS[$source_file]}"
       package_name=$(basename "$(dirname "$source_file")")
       output_file="${MOCK_DIR}/mock_${package_name}.go"

       log_info "Generating mocks for ${source_file}..."

       mockgen \
           -source="${source_file}" \
           -destination="${output_file}" \
           -package=mocks \
           ${interfaces}

       if [ $? -eq 0 ]; then
           log_info "✓ Generated mocks in ${output_file}"
       else
           log_error "✗ Failed to generate mocks for ${source_file}"
       fi
   done

   log_info "Mock generation completed successfully"
}

################################################################################
# Clean Generated Files
################################################################################

clean_generated() {
   log_info "Cleaning generated files..."

   if [ -d "$GENERATED_DIR" ]; then
       rm -rf "$GENERATED_DIR"
       log_info "Removed ${GENERATED_DIR}"
   fi

   if [ -d "$MOCK_DIR" ]; then
       rm -rf "$MOCK_DIR"
       log_info "Removed ${MOCK_DIR}"
   fi

   log_info "Cleanup completed"
}

################################################################################
# Main
################################################################################

main() {
   log_info "OpenEAAP Code Generation Script"
   log_info "================================"

   # Parse arguments
   GENERATE_PROTO=false
   GENERATE_OPENAPI=false
   GENERATE_MOCKS=false
   CLEAN=false

   if [ $# -eq 0 ]; then
       GENERATE_PROTO=true
       GENERATE_OPENAPI=true
       GENERATE_MOCKS=true
   fi

   while [[ $# -gt 0 ]]; do
       case $1 in
           --proto)
               GENERATE_PROTO=true
               shift
               ;;
           --openapi)
               GENERATE_OPENAPI=true
               shift
               ;;
           --mocks)
               GENERATE_MOCKS=true
               shift
               ;;
           --all)
               GENERATE_PROTO=true
               GENERATE_OPENAPI=true
               GENERATE_MOCKS=true
               shift
               ;;
           --clean)
               CLEAN=true
               shift
               ;;
           *)
               log_error "Unknown option: $1"
               echo "Usage: $0 [--proto] [--openapi] [--mocks] [--all] [--clean]"
               exit 1
               ;;
       esac
   done

   # Clean if requested
   if [ "$CLEAN" = true ]; then
       clean_generated
   fi

   # Install required tools
   install_tools

   # Generate code based on flags
   if [ "$GENERATE_PROTO" = true ]; then
       generate_grpc
   fi

   if [ "$GENERATE_OPENAPI" = true ]; then
       generate_openapi_clients
   fi

   if [ "$GENERATE_MOCKS" = true ]; then
       generate_mocks
   fi

   log_info "================================"
   log_info "All code generation tasks completed successfully! 🎉"
}

# Run main function
main "$@"

# Personal.AI order the ending
