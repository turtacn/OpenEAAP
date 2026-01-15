#!/bin/bash

# OpenEAAP Test Script
# 运行单元测试、集成测试、端到端测试，生成测试覆盖率报告
# Usage: ./scripts/test.sh [unit|integration|e2e|all|coverage]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# 配置
COVERAGE_DIR="coverage"
COVERAGE_FILE="${COVERAGE_DIR}/coverage.out"
COVERAGE_HTML="${COVERAGE_DIR}/coverage.html"
TEST_TIMEOUT="10m"
INTEGRATION_TIMEOUT="20m"
E2E_TIMEOUT="30m"

# 打印带颜色的消息
print_info() {
   echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
   echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
   echo -e "${RED}[ERROR]${NC} $1"
}

# 创建覆盖率目录
prepare_coverage_dir() {
   print_info "准备覆盖率目录..."
   mkdir -p "$COVERAGE_DIR"
   rm -f "$COVERAGE_FILE" "$COVERAGE_HTML"
}

# 检查依赖
check_dependencies() {
   print_info "检查依赖..."

   # 检查 Go
   if ! command -v go &> /dev/null; then
       print_error "Go 未安装"
       exit 1
   fi

   # 检查 Docker（集成测试和 E2E 测试需要）
   if ! command -v docker &> /dev/null; then
       print_warn "Docker 未安装，集成测试和 E2E 测试可能失败"
   fi

   print_info "依赖检查完成"
}

# 运行单元测试
run_unit_tests() {
   print_info "运行单元测试..."

   go test \
       -v \
       -race \
       -timeout "$TEST_TIMEOUT" \
       -coverprofile="${COVERAGE_DIR}/unit.out" \
       -covermode=atomic \
       $(go list ./... | grep -v /test/integration | grep -v /test/e2e) \
       2>&1 | tee "${COVERAGE_DIR}/unit_test.log"

   local exit_code=${PIPESTATUS[0]}

   if [ $exit_code -eq 0 ]; then
       print_info "✅ 单元测试通过"
   else
       print_error "❌ 单元测试失败"
       return $exit_code
   fi
}

# 启动集成测试依赖服务
start_integration_services() {
   print_info "启动集成测试依赖服务（Docker Compose）..."

   if [ ! -f "docker-compose.test.yml" ]; then
       print_warn "docker-compose.test.yml 不存在，跳过服务启动"
       return 0
   fi

   docker-compose -f docker-compose.test.yml up -d

   # 等待服务就绪
   print_info "等待服务就绪..."
   sleep 10

   # 检查 PostgreSQL
   print_info "检查 PostgreSQL..."
   docker-compose -f docker-compose.test.yml exec -T postgres pg_isready -U openeeap || {
       print_error "PostgreSQL 未就绪"
       return 1
   }

   # 检查 Redis
   print_info "检查 Redis..."
   docker-compose -f docker-compose.test.yml exec -T redis redis-cli ping | grep -q PONG || {
       print_error "Redis 未就绪"
       return 1
   }

   print_info "所有依赖服务已就绪"
}

# 停止集成测试依赖服务
stop_integration_services() {
   print_info "停止集成测试依赖服务..."

   if [ -f "docker-compose.test.yml" ]; then
       docker-compose -f docker-compose.test.yml down -v
   fi
}

# 运行集成测试
run_integration_tests() {
   print_info "运行集成测试..."

   # 启动依赖服务
   start_integration_services || {
       print_error "启动依赖服务失败"
       return 1
   }

   # 设置测试环境变量
   export TEST_ENV="integration"
   export DATABASE_URL="postgres://openeeap:password@localhost:5432/openeeap_test?sslmode=disable"
   export REDIS_URL="redis://localhost:6379/0"
   export MILVUS_URL="localhost:19530"

   # 运行集成测试
   go test \
       -v \
       -race \
       -timeout "$INTEGRATION_TIMEOUT" \
       -coverprofile="${COVERAGE_DIR}/integration.out" \
       -covermode=atomic \
       -tags=integration \
       ./test/integration/... \
       2>&1 | tee "${COVERAGE_DIR}/integration_test.log"

   local exit_code=${PIPESTATUS[0]}

   # 清理
   stop_integration_services

   if [ $exit_code -eq 0 ]; then
       print_info "✅ 集成测试通过"
   else
       print_error "❌ 集成测试失败"
       return $exit_code
   fi
}

# 运行端到端测试
run_e2e_tests() {
   print_info "运行端到端测试..."

   # 构建应用
   print_info "构建应用..."
   make build || {
       print_error "构建应用失败"
       return 1
   }

   # 启动完整环境
   print_info "启动完整测试环境..."
   docker-compose -f docker-compose.test.yml up -d

   # 启动应用
   print_info "启动应用..."
   ./bin/server --config configs/test.yaml &
   SERVER_PID=$!

   # 等待应用就绪
   print_info "等待应用就绪..."
   for i in {1..30}; do
       if curl -s http://localhost:8080/health > /dev/null; then
           print_info "应用已就绪"
           break
       fi
       if [ $i -eq 30 ]; then
           print_error "应用启动超时"
           kill $SERVER_PID
           stop_integration_services
           return 1
       fi
       sleep 2
   done

   # 设置测试环境变量
   export TEST_ENV="e2e"
   export API_BASE_URL="http://localhost:8080"

   # 运行 E2E 测试
   go test \
       -v \
       -timeout "$E2E_TIMEOUT" \
       -tags=e2e \
       ./test/e2e/... \
       2>&1 | tee "${COVERAGE_DIR}/e2e_test.log"

   local exit_code=${PIPESTATUS[0]}

   # 清理
   print_info "清理测试环境..."
   kill $SERVER_PID
   stop_integration_services

   if [ $exit_code -eq 0 ]; then
       print_info "✅ 端到端测试通过"
   else
       print_error "❌ 端到端测试失败"
       return $exit_code
   fi
}

# 合并覆盖率报告
merge_coverage() {
   print_info "合并覆盖率报告..."

   # 合并所有覆盖率文件
   echo "mode: atomic" > "$COVERAGE_FILE"

   for file in "${COVERAGE_DIR}"/*.out; do
       if [ -f "$file" ] && [ "$file" != "$COVERAGE_FILE" ]; then
           tail -n +2 "$file" >> "$COVERAGE_FILE"
       fi
   done

   print_info "覆盖率报告已合并到 $COVERAGE_FILE"
}

# 生成覆盖率报告
generate_coverage_report() {
   print_info "生成覆盖率报告..."

   if [ ! -f "$COVERAGE_FILE" ]; then
       print_warn "覆盖率文件不存在，跳过报告生成"
       return 0
   fi

   # 生成 HTML 报告
   go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"
   print_info "HTML 覆盖率报告: $COVERAGE_HTML"

   # 生成总览
   go tool cover -func="$COVERAGE_FILE" | tee "${COVERAGE_DIR}/coverage_summary.txt"

   # 提取总覆盖率
   TOTAL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}')
   print_info "📊 总覆盖率: ${GREEN}${TOTAL_COVERAGE}${NC}"

   # 检查覆盖率阈值
   COVERAGE_THRESHOLD="70.0"
   COVERAGE_VALUE=$(echo "$TOTAL_COVERAGE" | sed 's/%//')

   if (( $(echo "$COVERAGE_VALUE >= $COVERAGE_THRESHOLD" | bc -l) )); then
       print_info "✅ 覆盖率达标 (>= ${COVERAGE_THRESHOLD}%)"
   else
       print_warn "⚠️  覆盖率未达标 (< ${COVERAGE_THRESHOLD}%)"
   fi
}

# 生成 JUnit XML 报告（用于 CI）
generate_junit_report() {
   print_info "生成 JUnit XML 报告..."

   # 安装 go-junit-report（如果未安装）
   if ! command -v go-junit-report &> /dev/null; then
       print_info "安装 go-junit-report..."
       go install github.com/jstemmer/go-junit-report/v2@latest
   fi

   # 转换测试日志为 JUnit XML
   for log_file in "${COVERAGE_DIR}"/*_test.log; do
       if [ -f "$log_file" ]; then
           xml_file="${log_file%.log}.xml"
           cat "$log_file" | go-junit-report -set-exit-code > "$xml_file"
           print_info "JUnit 报告: $xml_file"
       fi
   done
}

# 清理测试环境
cleanup() {
   print_info "清理测试环境..."

   # 停止可能残留的服务
   stop_integration_services

   # 清理临时文件
   rm -f /tmp/openeeap_test_*

   print_info "清理完成"
}

# 显示使用帮助
show_usage() {
   cat << EOF
OpenEAAP 测试脚本

用法:
   ./scripts/test.sh [命令]

命令:
   unit         运行单元测试
   integration  运行集成测试
   e2e          运行端到端测试
   all          运行所有测试（默认）
   coverage     生成覆盖率报告
   clean        清理测试环境
   help         显示此帮助信息

示例:
   ./scripts/test.sh unit              # 只运行单元测试
   ./scripts/test.sh integration       # 只运行集成测试
   ./scripts/test.sh all               # 运行所有测试
   ./scripts/test.sh coverage          # 生成覆盖率报告

环境变量:
   TEST_TIMEOUT            单元测试超时时间（默认: 10m）
   INTEGRATION_TIMEOUT     集成测试超时时间（默认: 20m）
   E2E_TIMEOUT             E2E 测试超时时间（默认: 30m）
   COVERAGE_THRESHOLD      覆盖率阈值（默认: 70.0%）

EOF
}

# 主函数
main() {
   local command="${1:-all}"

   case "$command" in
       unit)
           check_dependencies
           prepare_coverage_dir
           run_unit_tests
           ;;
       integration)
           check_dependencies
           prepare_coverage_dir
           run_integration_tests
           ;;
       e2e)
           check_dependencies
           prepare_coverage_dir
           run_e2e_tests
           ;;
       all)
           check_dependencies
           prepare_coverage_dir

           print_info "=========================================="
           print_info "开始运行所有测试"
           print_info "=========================================="

           # 运行单元测试
           run_unit_tests || exit 1

           # 运行集成测试
           run_integration_tests || exit 1

           # 运行 E2E 测试
           run_e2e_tests || exit 1

           # 合并并生成覆盖率报告
           merge_coverage
           generate_coverage_report
           generate_junit_report

           print_info "=========================================="
           print_info "✅ 所有测试通过！"
           print_info "=========================================="
           ;;
       coverage)
           merge_coverage
           generate_coverage_report
           ;;
       clean)
           cleanup
           ;;
       help)
           show_usage
           ;;
       *)
           print_error "未知命令: $command"
           show_usage
           exit 1
           ;;
   esac
}

# 捕获 Ctrl+C
trap cleanup EXIT

# 执行主函数
main "$@"

# Personal.AI order the ending
