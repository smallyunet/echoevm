#!/bin/bash

# EchoEVM Test Helper Functions
# Common utilities for test scripts

# Colors and formatting
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export YELLOW='\033[1;33m'
export BLUE='\033[0;34m'
export PURPLE='\033[0;35m'
export CYAN='\033[0;36m'
export WHITE='\033[1;37m'
export BOLD='\033[1m'
export NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${PURPLE}[DEBUG]${NC} $1"
    fi
}

# Progress indicators
show_progress() {
    local current=$1
    local total=$2
    local message=$3
    local percent=$((current * 100 / total))
    echo -e "${CYAN}[${current}/${total}] (${percent}%) ${message}${NC}"
}

# Timer functions
start_timer() {
    export TIMER_START=$(date +%s)
}

end_timer() {
    local end_time=$(date +%s)
    local duration=$((end_time - TIMER_START))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))
    echo "${minutes}m ${seconds}s"
}

# File and directory utilities
ensure_directory() {
    local dir=$1
    if [ ! -d "$dir" ]; then
        log_info "Creating directory: $dir"
        mkdir -p "$dir"
    fi
}

get_project_root() {
    # Find project root by looking for go.mod
    local current_dir=$(pwd)
    while [ "$current_dir" != "/" ]; do
        if [ -f "$current_dir/go.mod" ]; then
            echo "$current_dir"
            return 0
        fi
        current_dir=$(dirname "$current_dir")
    done
    echo "."
}

# Contract utilities
check_contract_artifacts() {
    local artifacts_dir="$1"
    if [ ! -d "$artifacts_dir" ]; then
        log_warning "Contract artifacts not found at $artifacts_dir"
        log_info "Attempting to build contracts..."
        return 1
    fi
    return 0
}

build_contracts() {
    local contract_dir="$1"
    log_info "Building contracts in $contract_dir"
    
    if [ -f "$contract_dir/package.json" ]; then
        cd "$contract_dir"
        if command -v npm &> /dev/null; then
            npm install && npm run compile
            cd - > /dev/null
            return $?
        else
            log_error "npm not found. Cannot build contracts."
            return 1
        fi
    else
        log_error "No package.json found in $contract_dir"
        return 1
    fi
}

# Test execution utilities
run_with_timeout() {
    local timeout_seconds=$1
    local command=$2
    
    if command -v timeout &> /dev/null; then
        timeout "$timeout_seconds" bash -c "$command"
    else
        # macOS doesn't have timeout command by default
        eval "$command"
    fi
}

capture_output() {
    local command=$1
    local output_file=$2
    
    eval "$command" > "$output_file" 2>&1
    return $?
}

# Test result formatting
format_test_result() {
    local test_name="$1"
    local status="$2"  # PASSED, FAILED, SKIPPED
    local duration="$3"
    local details="$4"
    
    case $status in
        "PASSED")
            echo -e "${GREEN}✓${NC} $test_name ${CYAN}($duration)${NC}"
            ;;
        "FAILED")
            echo -e "${RED}✗${NC} $test_name ${CYAN}($duration)${NC}"
            if [ -n "$details" ]; then
                echo -e "  ${RED}Error:${NC} $details"
            fi
            ;;
        "SKIPPED")
            echo -e "${YELLOW}⊝${NC} $test_name ${CYAN}(skipped)${NC}"
            if [ -n "$details" ]; then
                echo -e "  ${YELLOW}Reason:${NC} $details"
            fi
            ;;
    esac
}

# Report generation
generate_summary_report() {
    local total_tests=$1
    local passed_tests=$2
    local failed_tests=$3
    local skipped_tests=$4
    local duration=$5
    
    echo ""
    echo -e "${BOLD}${BLUE}========================================${NC}"
    echo -e "${BOLD}${BLUE}Test Summary Report${NC}"
    echo -e "${BOLD}${BLUE}========================================${NC}"
    echo -e "Total tests:   ${BLUE}$total_tests${NC}"
    echo -e "Passed:        ${GREEN}$passed_tests${NC}"
    echo -e "Failed:        ${RED}$failed_tests${NC}"
    echo -e "Skipped:       ${YELLOW}$skipped_tests${NC}"
    echo -e "Duration:      ${CYAN}$duration${NC}"
    
    local success_rate=0
    if [ $total_tests -gt 0 ]; then
        success_rate=$((passed_tests * 100 / total_tests))
    fi
    echo -e "Success rate:  ${CYAN}${success_rate}%${NC}"
    
    if [ $failed_tests -eq 0 ]; then
        echo -e "\n${GREEN}${BOLD}🎉 All tests passed!${NC}"
    else
        echo -e "\n${RED}${BOLD}❌ Some tests failed!${NC}"
    fi
}

# Environment detection
detect_environment() {
    if [ -n "$CI" ]; then
        echo "ci"
    elif [ -n "$GITHUB_ACTIONS" ]; then
        echo "github_actions"
    elif [ -n "$JENKINS_URL" ]; then
        echo "jenkins"
    else
        echo "development"
    fi
}

# Cleanup functions
cleanup_temp_files() {
    log_debug "Cleaning up temporary files..."
    rm -f /tmp/test_output.txt
    rm -f /tmp/echoevm_test_*
}

# Trap for cleanup
setup_cleanup_trap() {
    trap cleanup_temp_files EXIT
}

# Initialize helpers
init_test_helpers() {
    log_debug "Initializing test helpers..."
    setup_cleanup_trap
    ensure_directory "$(get_project_root)/test/reports"
}

# Check if running in verbose mode
is_verbose() {
    [ "$VERBOSE" = true ] || [ "$V" = "1" ] || [ "$DEBUG" = "1" ]
}
