package actions

import (
	"context"
	"time"

	"github.com/Layr-Labs/eigensdk-go/types"
	operator "github.com/yetanotherco/aligned_layer/operator/pkg"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
	"github.com/yetanotherco/aligned_layer/core/config"
)

var QuorumNumberFlag = &cli.UintFlag{
	Name:     "quorum-number",
	Required: true,
	Usage:    "Specifies the quorum to register with. Possible values: 0 - register with the `eth` quorum, 1 - register with the `ali` quorum.",
}

var registerFlags = []cli.Flag{
	config.ConfigFileFlag,
	QuorumNumberFlag,
}

var RegisterCommand = &cli.Command{
	Name:        "register",
	Usage:       "Register operator with Aligned Layer",
	Description: "CLI command to register opeartor with Aligned Layer",
	Flags:       registerFlags,
	Action:      registerOperatorMain,
}

func registerOperatorMain(ctx *cli.Context) error {
	operatorConfig := config.NewOperatorConfig(ctx.String(config.ConfigFileFlag.Name))
	ecdsaConfig := config.NewEcdsaConfig(ctx.String(config.ConfigFileFlag.Name), operatorConfig.BaseConfig.ChainId)

	quorumNumbersBytes := []byte{byte(QuorumNumberFlag.Value)}
	quorumNumbers := types.QuorumNums{types.QuorumNum(QuorumNumberFlag.Value)}

	// Generate salt and expiry
	privateKeyBytes := []byte(operatorConfig.BlsConfig.KeyPair.PrivKey.String())
	salt := [32]byte{}

	copy(salt[:], crypto.Keccak256([]byte("churn"), []byte(time.Now().String()), quorumNumbersBytes, privateKeyBytes))

	err := operator.RegisterOperator(context.Background(), operatorConfig, ecdsaConfig, quorumNumbers, salt)
	if err != nil {
		operatorConfig.BaseConfig.Logger.Error("Failed to register operator", "err", err)
		return err
	}

	return nil
}
