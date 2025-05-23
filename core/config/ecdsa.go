package config

import (
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math/big"
	"os"

	ecdsa2 "github.com/Layr-Labs/eigensdk-go/crypto/ecdsa"
	signer "github.com/Layr-Labs/eigensdk-go/signerv2"
	"github.com/yetanotherco/aligned_layer/core/utils"
)

type EcdsaConfig struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
	SignerFn   signer.SignerFn
}

type EcdsaConfigFromYaml struct {
	Ecdsa struct {
		PrivateKeyStorePath     string `yaml:"private_key_store_path"`
		PrivateKeyStorePassword string `yaml:"private_key_store_password"`
	} `yaml:"ecdsa"`
}

func NewEcdsaConfig(ecdsaConfigFilePath string, chainId *big.Int) *EcdsaConfig {
	if _, err := os.Stat(ecdsaConfigFilePath); errors.Is(err, os.ErrNotExist) {
		log.Fatal("Setup ecdsa config file does not exist")
	}

	var ecdsaConfigFromYaml EcdsaConfigFromYaml
	err := utils.ReadYamlConfig(ecdsaConfigFilePath, &ecdsaConfigFromYaml)
	if err != nil {
		log.Fatal("Error reading ecdsa config: ", err)
	}

	if ecdsaConfigFromYaml.Ecdsa.PrivateKeyStorePath == "" {
		log.Fatal("Ecdsa private key store path is empty")
	}

	ecdsaKeyPair, err := ecdsa2.ReadKey(ecdsaConfigFromYaml.Ecdsa.PrivateKeyStorePath, ecdsaConfigFromYaml.Ecdsa.PrivateKeyStorePassword)
	if err != nil {
		log.Fatal("Error reading ecdsa private key from file: ", err)
	}

	signerConfig := signer.Config{
		PrivateKey: ecdsaKeyPair,
	}
	signerFn, address, err := signer.SignerFromConfig(signerConfig, chainId)
	if err != nil {
		log.Fatal("Cannot create signer", "err", err)
	}

	return &EcdsaConfig{
		Address:    address,
		PrivateKey: ecdsaKeyPair,
		SignerFn:   signerFn,
	}
}
