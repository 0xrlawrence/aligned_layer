package config

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/yetanotherco/aligned_layer/core/utils"
)

type AggregatorConfig struct {
	BaseConfig  *BaseConfig
	EcdsaConfig *EcdsaConfig
	BlsConfig   *BlsConfig
	Aggregator  struct {
		ServerIpPortAddress           string
		BlsPublicKeyCompendiumAddress common.Address
		AvsServiceManagerAddress      common.Address
		EnableMetrics                 bool
		MetricsIpPortAddress          string
		TelemetryIpPortAddress        string
		GarbageCollectorPeriod        time.Duration
		GarbageCollectorTasksAge      uint64
		GarbageCollectorTasksInterval uint64
		BlsServiceTaskTimeout         time.Duration
		// number of blocks to wait for a transaction to be confirmed
		// default: 0
		ConfirmationBlocks uint64
		// time to wait for a transaction to be broadcasted to the network
		// could be direct via eth_sendRawTransaction or indirect via a wallet service such as fireblocks
		// default: 2 minutes
		TxnBroadcastTimeout time.Duration
		// time to wait for a transaction to be confirmed (mined + confirmationBlocks blocks)
		// default: 5 * 12 seconds
		TxnConfirmationTimeout time.Duration
		// max number of times to retry sending a transaction before failing
		// this applies to every transaction attempt when a nonce is bumped
		// default: 3
		MaxSendTransactionRetry int
		// time to wait between checking for each transaction receipt
		// while monitoring transactions to get mined
		// default: 3 seconds
		GetTxReceiptTickerDuration time.Duration
		// default gas tip cap to use when eth_maxPriorityFeePerGas is not available
		// default: 5 gwei
		FallbackGasTipCap uint64
		// multiplier for gas limit to add a buffer and increase chance of tx getting included. Should be >= 1.0
		// default: 1.2
		GasMultiplier float64
		// multiplier for gas tip. Should be >= 1.0
		// default: 1.25
		GasTipMultiplier float64
	}
}

type AggregatorConfigFromYaml struct {
	Aggregator struct {
		ServerIpPortAddress           string         `yaml:"server_ip_port_address"`
		BlsPublicKeyCompendiumAddress common.Address `yaml:"bls_public_key_compendium_address"`
		AvsServiceManagerAddress      common.Address `yaml:"avs_service_manager_address"`
		EnableMetrics                 bool           `yaml:"enable_metrics"`
		MetricsIpPortAddress          string         `yaml:"metrics_ip_port_address"`
		TelemetryIpPortAddress        string         `yaml:"telemetry_ip_port_address"`
		GarbageCollectorPeriod        time.Duration  `yaml:"garbage_collector_period"`
		GarbageCollectorTasksAge      uint64         `yaml:"garbage_collector_tasks_age"`
		GarbageCollectorTasksInterval uint64         `yaml:"garbage_collector_tasks_interval"`
		BlsServiceTaskTimeout         time.Duration  `yaml:"bls_service_task_timeout"`
		ConfirmationBlocks            uint64         `yaml:"confirmation_blocks"`
		TxnBroadcastTimeout           time.Duration  `yaml:"txn_broadcast_timeout"`
		TxnConfirmationTimeout        time.Duration  `yaml:"txn_confirmation_timeout"`
		MaxSendTransactionRetry       int            `yaml:"max_send_transaction_retry"`
		GetTxReceiptTickerDuration    time.Duration  `yaml:"get_tx_receipt_ticker_duration"`
		FallbackGasTipCap             uint64         `yaml:"fallback_gas_tip_cap"`
		GasMultiplier                 float64        `yaml:"gas_multiplier"`
		GasTipMultiplier              float64        `yaml:"gas_tip_multiplier"`
	} `yaml:"aggregator"`
}

func NewAggregatorConfig(configFilePath string) *AggregatorConfig {

	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		log.Fatal("Setup config file does not exist")
	}

	baseConfig := NewBaseConfig(configFilePath)
	if baseConfig == nil {
		log.Fatal("Error reading base config: ")
	}

	ecdsaConfig := NewEcdsaConfig(configFilePath, baseConfig.ChainId)
	if ecdsaConfig == nil {
		log.Fatal("Error reading ecdsa config: ")
	}

	blsConfig := NewBlsConfig(configFilePath)
	if blsConfig == nil {
		log.Fatal("Error reading bls config: ")
	}

	var aggregatorConfigFromYaml AggregatorConfigFromYaml
	err := utils.ReadYamlConfig(configFilePath, &aggregatorConfigFromYaml)
	if err != nil {
		log.Fatal("Error reading aggregator config: ", err)
	}

	return &AggregatorConfig{
		BaseConfig:  baseConfig,
		EcdsaConfig: ecdsaConfig,
		BlsConfig:   blsConfig,
		Aggregator: struct {
			ServerIpPortAddress           string
			BlsPublicKeyCompendiumAddress common.Address
			AvsServiceManagerAddress      common.Address
			EnableMetrics                 bool
			MetricsIpPortAddress          string
			TelemetryIpPortAddress        string
			GarbageCollectorPeriod        time.Duration
			GarbageCollectorTasksAge      uint64
			GarbageCollectorTasksInterval uint64
			BlsServiceTaskTimeout         time.Duration
			ConfirmationBlocks            uint64
			TxnBroadcastTimeout           time.Duration
			TxnConfirmationTimeout        time.Duration
			MaxSendTransactionRetry       int
			GetTxReceiptTickerDuration    time.Duration
			FallbackGasTipCap             uint64
			GasMultiplier                 float64
			GasTipMultiplier              float64
		}(aggregatorConfigFromYaml.Aggregator),
	}
}
