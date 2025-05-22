#!/bin/bash

# Configuration
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
DEFAULT_RPC_URL="http://localhost:8545"
DEPLOYMENT_FILE="contracts/script/output/devnet/eigenlayer_deployment_output.json"

# Function to validate input arguments
validate_args() {
    if [[ "$#" -ne 2 ]]; then
        echo "Usage: $0 <config_file> <amount>"
        exit 1
    fi
}

# Function to set RPC_URL
set_rpc_url() {
    RPC_URL=${RPC_URL:-$DEFAULT_RPC_URL}
    echo "Using RPC_URL: $RPC_URL"
}

# Function to get operator address from config file
get_operator_address() {
    local config_file="$1"
    OPERATOR_ADDRESS=$(yq -r '.operator.address' "$config_file")
    if [[ -z "$OPERATOR_ADDRESS" ]]; then
        echo "Error: Could not read operator address from $config_file"
        exit 1
    fi
}

# Function to normalize address (remove 0x prefix and handle leading zeros)
normalize_address() {
      local address=$1
      # Remove '0x' prefix, take last 40 characters, then add '0x' back
      echo "0x$(echo "$address" | sed 's/^0x//' | tail -c 41)"
}

# Function to get mock token address
get_mock_token_address() {
    local strategy_type="$1"
    local mock_strategy_address=$(jq -r ".addresses.strategies.$strategy_type" "$DEPLOYMENT_FILE")
    local mock_token_address=$(cast call "$mock_strategy_address" "underlyingToken()" --rpc-url "$RPC_URL")

    if [[ -z "$mock_token_address" ]]; then
        echo "Error: Mock token address is empty for $strategy_type. Please deploy contracts first."
        exit 1
    fi

    normalize_address "$mock_token_address"
}

# Function to mint tokens
mint_tokens() {
    local token_address="$1"
    local recipient="$2"
    local amount="$3"

    echo "Minting $amount tokens to $recipient"
    echo "Token address: $token_address"

    cast send "$token_address" \
        "transfer(address recipient, uint256 amount)(bool)" \
        "$recipient" "$amount" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL"
}

# Main execution
main() {
    validate_args "$@"
    set_rpc_url

    local config_file="$1"
    local amount="$2"

    get_operator_address "$config_file"

    # Mint WETH tokens
    local weth_token_address=$(get_mock_token_address "WETH")
    mint_tokens "$weth_token_address" "$OPERATOR_ADDRESS" "$amount"

    # Mint ALI tokens
    local ali_token_address=$(get_mock_token_address "ALI")
    mint_tokens "$ali_token_address" "$OPERATOR_ADDRESS" "$amount"
}

# Execute main function
main "$@"