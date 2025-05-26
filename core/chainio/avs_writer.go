package chainio

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/Layr-Labs/eigensdk-go/chainio/clients/wallet"
	"github.com/Layr-Labs/eigensdk-go/chainio/txmgr/geometric"
	"math/big"
	"time"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients"
	"github.com/Layr-Labs/eigensdk-go/chainio/clients/avsregistry"
	"github.com/Layr-Labs/eigensdk-go/chainio/clients/eth"
	"github.com/Layr-Labs/eigensdk-go/logging"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	servicemanager "github.com/yetanotherco/aligned_layer/contracts/bindings/AlignedLayerServiceManager"
	retry "github.com/yetanotherco/aligned_layer/core"
	"github.com/yetanotherco/aligned_layer/core/config"
	"github.com/yetanotherco/aligned_layer/core/utils"
	"github.com/yetanotherco/aligned_layer/metrics"
)

type AvsWriter struct {
	*avsregistry.ChainWriter
	AvsContractBindings *AvsServiceBindings
	logger              logging.Logger
	TxManager           geometric.GeometricTxManager
	TxManagerFallback   geometric.GeometricTxManager
	Client              eth.InstrumentedClient
	ClientFallback      eth.InstrumentedClient
	metrics             *metrics.Metrics
}

func NewAvsWriterFromConfig(baseConfig *config.BaseConfig, ecdsaConfig *config.EcdsaConfig, metrics *metrics.Metrics, geometricTxnManagerParams geometric.GeometricTxnManagerParams) (*AvsWriter, error) {

	buildAllConfig := clients.BuildAllConfig{
		EthHttpUrl:                 baseConfig.EthRpcUrl,
		EthWsUrl:                   baseConfig.EthWsUrl,
		RegistryCoordinatorAddr:    baseConfig.AlignedLayerDeploymentConfig.AlignedLayerRegistryCoordinatorAddr.String(),
		OperatorStateRetrieverAddr: baseConfig.AlignedLayerDeploymentConfig.AlignedLayerOperatorStateRetrieverAddr.String(),
		AvsName:                    "AlignedLayer",
		PromMetricsIpPortAddress:   baseConfig.EigenMetricsIpPortAddress,
		ServiceManagerAddress:      baseConfig.AlignedLayerDeploymentConfig.AlignedLayerServiceManagerAddr.String(),
	}

	clients, err := clients.BuildAll(
		buildAllConfig,
		ecdsaConfig.PrivateKey,
		baseConfig.Logger,
	)
	if err != nil {
		baseConfig.Logger.Error("Cannot build signer config", "err", err)
		return nil, err
	}

	avsServiceBindings, err := NewAvsServiceBindings(
		baseConfig.AlignedLayerDeploymentConfig.AlignedLayerServiceManagerAddr,
		baseConfig.AlignedLayerDeploymentConfig.AlignedLayerOperatorStateRetrieverAddr,
		baseConfig.EthRpcClient,
		baseConfig.EthRpcClientFallback,
		baseConfig.Logger,
	)
	if err != nil {
		baseConfig.Logger.Error("Cannot create avs service bindings", "err", err)
		return nil, err
	}

	privateKeyWallet, err := wallet.NewPrivateKeyWallet(
		&baseConfig.EthRpcClient,
		ecdsaConfig.SignerFn,
		ecdsaConfig.Address,
		baseConfig.Logger,
	)
	if err != nil {
		baseConfig.Logger.Error("Cannot create private key wallet", "err", err)
		return nil, err
	}

	privateKeyWalletFallback, err := wallet.NewPrivateKeyWallet(
		&baseConfig.EthRpcClientFallback,
		ecdsaConfig.SignerFn,
		ecdsaConfig.Address,
		baseConfig.Logger,
	)
	if err != nil {
		baseConfig.Logger.Error("Cannot create private key wallet", "err", err)
		return nil, err
	}

	txManager := geometric.NewGeometricTxnManager(
		&baseConfig.EthRpcClient,
		privateKeyWallet,
		baseConfig.Logger,
		geometric.NewNoopMetrics(), // TODO: Set a correct metrics instance
		geometricTxnManagerParams,
	)

	txManagerFallback := geometric.NewGeometricTxnManager(
		&baseConfig.EthRpcClientFallback,
		privateKeyWalletFallback,
		baseConfig.Logger,
		geometric.NewNoopMetrics(), // TODO: Set a correct metrics instance
		geometricTxnManagerParams,
	)

	chainWriter := clients.AvsRegistryChainWriter

	return &AvsWriter{
		ChainWriter:         chainWriter,
		AvsContractBindings: avsServiceBindings,
		logger:              baseConfig.Logger,
		TxManager:           *txManager,
		TxManagerFallback:   *txManagerFallback,
		Client:              baseConfig.EthRpcClient,
		ClientFallback:      baseConfig.EthRpcClientFallback,
		metrics:             metrics,
	}, nil
}

// SendAggregatedResponse continuously sends a RespondToTask transaction until it is included in the blockchain.
// This function:
//  1. Simulates the transaction to calculate the nonce and initial gas price without broadcasting it.
//  2. Repeatedly attempts to send the transaction, bumping the gas price after `timeToWaitBeforeBump` has passed.
//  3. Monitors for the receipt of previously sent transactions or checks the state to confirm if the response
//     has already been processed (e.g., by another transaction).
//  4. Validates that the aggregator and batcher have sufficient balance to cover transaction costs before sending.
//
// Returns:
//   - A transaction receipt if the transaction is successfully included in the blockchain.
//   - If no receipt is found, but the batch state indicates the response has already been processed, it exits
//     without an error (returning `nil, nil`).
//   - An error if the process encounters a fatal issue (e.g., permanent failure in verifying balances or state).
func (w *AvsWriter) SendAggregatedResponse(batchIdentifierHash [32]byte, batchMerkleRoot [32]byte, senderAddress [20]byte, nonSignerStakesAndSignature servicemanager.IBLSSignatureCheckerNonSignerStakesAndSignature) (*types.Receipt, error) {
	txOpts, err := w.TxManager.GetNoSendTxOpts()
	if err != nil {
		w.logger.Errorf("Failed to get transaction options: %v", err)
		return nil, err
	}
	// This is used to simulate the transaction and get the transaction ready for sending
	tx, err := w.RespondToTaskV2Retryable(txOpts, batchMerkleRoot, senderAddress, nonSignerStakesAndSignature, retry.SendToChainRetryParams())
	if err != nil {
		w.logger.Errorf("Failed to simulate transaction: %v", err)
		return nil, err
	}

	batchMerkleRootHashString := hex.EncodeToString(batchMerkleRoot[:])

	respondToTaskV2Func := func() (*types.Receipt, error) {
		// We compare both Aggregator funds and Batcher balance in Aligned against respondToTaskFeeLimit
		// Both are required to have some balance, more details inside the function
		err = w.checkAggAndBatcherHaveEnoughBalance(tx, *txOpts, batchIdentifierHash, senderAddress)
		if err != nil {
			w.logger.Errorf("Permanent error when checking aggregator and batcher balances for MerkleRoot %v. err: %v", batchMerkleRootHashString, err)
			return nil, retry.PermanentError{Inner: err}
		}

		w.logger.Infof("Sending RespondToTask transaction (%v) for MerkleRoot %v", tx.Hash().Hex(), batchMerkleRootHashString)
		receipt, err := w.SendTransactionRetryable(context.Background(), tx, retry.SendToChainRetryParams())
		if err != nil {
			w.logger.Errorf("RespondToTask transaction (%v) for MerkleRoot %v error: %v", tx.Hash().Hex(), batchMerkleRootHashString, err)
			return nil, err
		}
		w.logger.Infof("RespondToTask transaction (%v) sent for MerkleRoot %v. %+v", tx.Hash().Hex(), batchMerkleRootHashString, receipt)
		w.updateAggregatorGasCostMetrics(receipt, batchIdentifierHash) // At this point receipt is not nil, so we can safely update the metrics
		return receipt, nil
	}
	return retry.RetryWithData(respondToTaskV2Func, retry.RespondToTaskV2())
}

// Calculates the transaction cost from the receipt and updates the total amount paid by the aggregator metric
// Then, it compares that tx cost with the batcher respondToTaskFeeLimit.
// If the tx cost was higher, it means the aggregator has paid the difference for the batcher (txCost - respondToTaskFeeLimit) and so metrics are updated accordingly.
func (w *AvsWriter) updateAggregatorGasCostMetrics(receipt *types.Receipt, batchIdentifierHash [32]byte) {
	batchState, err := w.BatchesStateRetryable(&bind.CallOpts{}, batchIdentifierHash, retry.NetworkRetryParams())
	if err != nil {
		return
	}
	respondToTaskFeeLimit := batchState.RespondToTaskFeeLimit

	txCost := new(big.Int).Mul(big.NewInt(int64(receipt.GasUsed)), receipt.EffectiveGasPrice)

	txCostInEth := utils.WeiToEth(txCost)
	w.metrics.AddAggregatorGasCostPaidTotal(txCostInEth)

	if respondToTaskFeeLimit.Cmp(txCost) < 0 {
		aggregatorDifferencePaid := new(big.Int).Sub(txCost, respondToTaskFeeLimit)
		aggregatorDifferencePaidInEth := utils.WeiToEth(aggregatorDifferencePaid)
		w.metrics.AddAggregatorGasPaidForBatcher(aggregatorDifferencePaidInEth)
		w.metrics.IncAggregatorPaidForBatcher()
		w.logger.Warnf("cost of transaction was higher than Batch.RespondToTaskFeeLimit, aggregator has paid the for the difference, aprox: %vethers", aggregatorDifferencePaidInEth)
	}
}

func (w *AvsWriter) checkAggAndBatcherHaveEnoughBalance(tx *types.Transaction, txOpts bind.TransactOpts, batchIdentifierHash [32]byte, senderAddress [20]byte) error {
	w.logger.Info("Checking if aggregator and batcher have enough balance for the transaction")
	aggregatorAddress := txOpts.From
	txGasAsBigInt := new(big.Int).SetUint64(tx.Gas())
	txGasPrice := tx.GasPrice()
	txCost := new(big.Int).Mul(txGasAsBigInt, txGasPrice)
	w.logger.Info("Transaction cost", "cost", txCost)

	batchState, err := w.BatchesStateRetryable(&bind.CallOpts{}, batchIdentifierHash, retry.NetworkRetryParams())
	if err != nil {
		w.logger.Error("Failed to get batch state", "error", err)
		w.logger.Info("Proceeding to check balances against transaction cost")
		return w.compareBalances(txCost, aggregatorAddress, senderAddress)
	}
	respondToTaskFeeLimit := batchState.RespondToTaskFeeLimit
	w.logger.Info("Checking balance against Batch RespondToTaskFeeLimit", "RespondToTaskFeeLimit", respondToTaskFeeLimit)
	// Note: we compare both Aggregator funds and Batcher balance in Aligned against respondToTaskFeeLimit
	// Batcher will pay up to respondToTaskFeeLimit, for this he needs that amount of funds in Aligned
	// Aggregator will pay any extra cost, for this he needs at least respondToTaskFeeLimit in his balance
	return w.compareBalances(respondToTaskFeeLimit, aggregatorAddress, senderAddress)
}

func (w *AvsWriter) compareBalances(amount *big.Int, aggregatorAddress common.Address, senderAddress [20]byte) error {
	if err := w.compareAggregatorBalance(amount, aggregatorAddress); err != nil {
		return err
	}
	if err := w.compareBatcherBalance(amount, senderAddress); err != nil {
		return err
	}
	return nil
}

func (w *AvsWriter) compareAggregatorBalance(amount *big.Int, aggregatorAddress common.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	aggregatorBalance, err := w.BalanceAtRetryable(ctx, aggregatorAddress, nil, retry.NetworkRetryParams())
	if err != nil {
		// Ignore and continue.
		w.logger.Error("failed to get aggregator balance: %v", err)
		return nil
	}
	w.logger.Info("Aggregator balance", "balance", aggregatorBalance)
	if aggregatorBalance.Cmp(amount) < 0 {
		return fmt.Errorf("cost is higher than Aggregator balance")
	}
	return nil
}

func (w *AvsWriter) compareBatcherBalance(amount *big.Int, senderAddress [20]byte) error {
	// Get batcher balance
	batcherBalance, err := w.BatcherBalancesRetryable(&bind.CallOpts{}, senderAddress, retry.NetworkRetryParams())
	if err != nil {
		// Ignore and continue.
		w.logger.Error("Failed to get batcherBalance", "error", err)
		return nil
	}
	w.logger.Info("Batcher balance", "balance", batcherBalance)
	if batcherBalance.Cmp(amount) < 0 {
		return fmt.Errorf("cost is higher than Batcher balance")
	}
	return nil
}
