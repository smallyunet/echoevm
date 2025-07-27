#!/bin/bash

# EchoEVM Contract Utilities
# Helper functions for working with smart contracts

# Source the main helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

# Contract paths
get_contracts_root() {
    echo "$(get_project_root)/test"
}

get_contract_artifacts_dir() {
    echo "$(get_contracts_root)/contract/artifacts/contracts"
}

get_binary_contracts_dir() {
    echo "$(get_contracts_root)/bins/build"
}

# Contract validation
validate_contract_artifact() {
    local artifact_path="$1"
    
    if [ ! -f "$artifact_path" ]; then
        log_error "Contract artifact not found: $artifact_path"
        return 1
    fi
    
    # Check if it's valid JSON
    if ! jq empty "$artifact_path" 2>/dev/null; then
        log_error "Invalid JSON in contract artifact: $artifact_path"
        return 1
    fi
    
    # Check for required fields
    local abi=$(jq -r '.abi // empty' "$artifact_path")
    local bytecode=$(jq -r '.bytecode // empty' "$artifact_path")
    
    if [ -z "$abi" ] || [ "$abi" = "null" ]; then
        log_error "Missing ABI in contract artifact: $artifact_path"
        return 1
    fi
    
    if [ -z "$bytecode" ] || [ "$bytecode" = "null" ]; then
        log_error "Missing bytecode in contract artifact: $artifact_path"
        return 1
    fi
    
    return 0
}

validate_binary_contract() {
    local binary_path="$1"
    
    if [ ! -f "$binary_path" ]; then
        log_error "Binary contract not found: $binary_path"
        return 1
    fi
    
    # Check if file is not empty
    if [ ! -s "$binary_path" ]; then
        log_error "Empty binary contract file: $binary_path"
        return 1
    fi
    
    # Check if it contains valid hex data
    if ! grep -q '^[0-9a-fA-F]*$' "$binary_path"; then
        log_error "Invalid hex data in binary contract: $binary_path"
        return 1
    fi
    
    return 0
}

# Contract discovery
find_contract_artifacts() {
    local pattern="$1"
    local artifacts_dir=$(get_contract_artifacts_dir)
    
    find "$artifacts_dir" -name "*.json" -path "*/$pattern*" 2>/dev/null
}

find_binary_contracts() {
    local pattern="$1"
    local binary_dir=$(get_binary_contracts_dir)
    
    find "$binary_dir" -name "*.bin" -name "*$pattern*" 2>/dev/null
}

list_all_contracts() {
    local artifacts_dir=$(get_contract_artifacts_dir)
    
    echo "Available contract artifacts:"
    find "$artifacts_dir" -name "*.json" | while read -r artifact; do
        local contract_name=$(basename "$(dirname "$artifact")")
        local category=$(basename "$(dirname "$(dirname "$artifact")")")
        echo "  $category/$contract_name"
    done
}

# Contract function analysis
get_contract_functions() {
    local artifact_path="$1"
    
    if ! validate_contract_artifact "$artifact_path"; then
        return 1
    fi
    
    jq -r '.abi[] | select(.type == "function") | .name' "$artifact_path"
}

get_function_signature() {
    local artifact_path="$1"
    local function_name="$2"
    
    if ! validate_contract_artifact "$artifact_path"; then
        return 1
    fi
    
    jq -r --arg fname "$function_name" '
        .abi[] | 
        select(.type == "function" and .name == $fname) | 
        .name + "(" + ([.inputs[].type] | join(",")) + ")"
    ' "$artifact_path"
}

analyze_contract() {
    local artifact_path="$1"
    
    if ! validate_contract_artifact "$artifact_path"; then
        return 1
    fi
    
    local contract_name=$(jq -r '.contractName' "$artifact_path")
    local source_name=$(jq -r '.sourceName' "$artifact_path")
    
    echo -e "${BLUE}Contract Analysis: $contract_name${NC}"
    echo -e "Source: $source_name"
    echo -e "Artifact: $artifact_path"
    echo ""
    
    echo -e "${YELLOW}Functions:${NC}"
    local functions=$(get_contract_functions "$artifact_path")
    if [ -n "$functions" ]; then
        echo "$functions" | while read -r func; do
            local signature=$(get_function_signature "$artifact_path" "$func")
            echo "  - $signature"
        done
    else
        echo "  No public functions found"
    fi
    echo ""
    
    echo -e "${YELLOW}Events:${NC}"
    local events=$(jq -r '.abi[] | select(.type == "event") | .name' "$artifact_path")
    if [ -n "$events" ]; then
        echo "$events" | while read -r event; do
            echo "  - $event"
        done
    else
        echo "  No events found"
    fi
}

# Contract execution helpers
build_echoevm_command() {
    local contract_type="$1"  # "artifact" or "bin"
    local contract_path="$2"
    local function_sig="$3"
    local args="$4"
    
    local cmd="go run ./cmd/echoevm run"
    
    case $contract_type in
        "artifact")
            cmd="$cmd -artifact $contract_path"
            ;;
        "bin")
            cmd="$cmd -bin $contract_path"
            ;;
        *)
            log_error "Invalid contract type: $contract_type. Use 'artifact' or 'bin'"
            return 1
            ;;
    esac
    
    cmd="$cmd -function \"$function_sig\""
    
    if [ -n "$args" ]; then
        cmd="$cmd -args \"$args\""
    else
        cmd="$cmd -args \"\""
    fi
    
    echo "$cmd"
}

# Test case generation
generate_test_cases_for_contract() {
    local artifact_path="$1"
    local output_file="$2"
    
    if ! validate_contract_artifact "$artifact_path"; then
        return 1
    fi
    
    local contract_name=$(jq -r '.contractName' "$artifact_path")
    local functions=$(get_contract_functions "$artifact_path")
    
    echo "# Auto-generated test cases for $contract_name" > "$output_file"
    echo "# Generated on $(date)" >> "$output_file"
    echo "" >> "$output_file"
    
    echo "[$contract_name.functions]" >> "$output_file"
    
    echo "$functions" | while read -r func; do
        local signature=$(get_function_signature "$artifact_path" "$func")
        cat >> "$output_file" << EOF
${func}_test = {
    artifact = "$artifact_path",
    function = "$signature",
    args = "0",  # TODO: Add appropriate test arguments
    expected_success = true,
    description = "Test $func function"
}

EOF
    done
    
    log_success "Generated test cases in $output_file"
}

# Contract building and management
ensure_contracts_built() {
    local contract_dir="$(get_contracts_root)/contract"
    local artifacts_dir=$(get_contract_artifacts_dir)
    
    if ! check_contract_artifacts "$artifacts_dir"; then
        log_info "Building contracts..."
        if build_contracts "$contract_dir"; then
            log_success "Contracts built successfully"
        else
            log_error "Failed to build contracts"
            return 1
        fi
    else
        log_debug "Contract artifacts found"
    fi
    
    return 0
}

# Contract test execution
execute_contract_test() {
    local contract_type="$1"
    local contract_path="$2"
    local function_sig="$3"
    local args="$4"
    local timeout="${5:-30}"
    
    local cmd=$(build_echoevm_command "$contract_type" "$contract_path" "$function_sig" "$args")
    
    if [ -z "$cmd" ]; then
        return 1
    fi
    
    log_debug "Executing: $cmd"
    
    start_timer
    if run_with_timeout "$timeout" "$cmd"; then
        local duration=$(end_timer)
        log_success "Contract test completed in $duration"
        return 0
    else
        local duration=$(end_timer)
        log_error "Contract test failed after $duration"
        return 1
    fi
}

# Initialize contract utilities
init_contract_utils() {
    log_debug "Initializing contract utilities..."
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_error "jq is required for contract utilities. Please install it."
        return 1
    fi
    
    # Ensure contracts are built
    ensure_contracts_built
}
