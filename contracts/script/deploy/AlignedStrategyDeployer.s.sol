pragma solidity ^0.8.12;

import "forge-std/Script.sol";
import "forge-std/StdJson.sol";

import {IStrategyManager} from
    "../../eigenlayer_contracts/eigenlayer-contracts/src/contracts/interfaces/IStrategyManager.sol";
import {IStrategy} from "../../eigenlayer_contracts/eigenlayer-contracts/src/contracts/interfaces/IStrategy.sol";
import {PauserRegistry} from
    "../../eigenlayer_contracts/eigenlayer-contracts/src/contracts/permissions/PauserRegistry.sol";
import {StrategyBaseTVLLimits} from
    "../../eigenlayer_contracts/eigenlayer-contracts/src/contracts/strategies/StrategyBaseTVLLimits.sol";
import "@openzeppelin/contracts/token/ERC20/presets/ERC20PresetFixedSupply.sol";

contract AlignedStrategyDeployerScript is Script {
    function run(string calldata configFile, string calldata eigenOutputFile) external {
        string memory configData = vm.readFile(configFile);
        string memory outputData = vm.readFile(eigenOutputFile);

        address strategyManagerAddress = stdJson.readAddress(outputData, ".address.alignedAggregatorAddress");
        IStrategyManager strategyManager = IStrategyManager(strategyManagerAddress);

        address eigenLayerPauserRegAddress = stdJson.readAddress(outputData, ".address.eigenLayerPauserReg");
        PauserRegistry eigenLayerPauserReg = PauserRegistry(eigenLayerPauserRegAddress);

        string calldata strategyVersion = stdJson.readString(configData, ".strategy.version");
        string calldata strategyMaxPerDeposit = stdJson.readString(configData, ".strategy.maxPerDeposit");
        string calldata strategyMaxDeposits = stdJson.readString(configData, ".strategy.maxDeposits");
        string calldata tokenOwner = stdJson.readString(configData, ".strategy.token.owner");
        string calldata tokenName = stdJson.readString(configData, ".strategy.token.name");
        string calldata tokenSymbol = stdJson.readString(configData, ".strategy.token.symbol");

        vm.startBroadcast();

        IStrategy[] memory strategiesToWhitelist;

        // Deploy Aligned strategy
        IStrategy alignedStrategyImplementation =
            new StrategyBaseTVLLimits(strategyManager, eigenLayerPauserReg, strategyVersion);
        IERC20 alignedStrategyToken =
            new ERC20PresetFixedSupply(tokenName, tokenSymbol, uint256(type(uint128).max), tokenOwner);
        IStrategy alignedStrategy = IStrategy(
            new TransparentUpgradeableProxy(
                address(alignedStrategyImplementation),
                address(eigenLayerProxyAdmin),
                abi.encodeWithSelector(
                    StrategyBaseTVLLimits.initialize.selector,
                    strategyMaxPerDeposit,
                    strategyMaxDeposits,
                    IERC20(address(alignedStrategyToken))
                )
            )
        );

        // Whitelist strategy
        strategiesToWhitelist[0] = alignedStrategy;
        strategyManager.addStrategiesToDepositWhitelist(strategiesToWhitelist);

        vm.stopBroadcast();

        vm.writeJson(eigenOutputFile, ".strategy.alignedStrategy", address(alignedStrategy));
        vm.writeJson(eigenOutputFile, ".strategy.alignedStrategyImplementation", address(alignedStrategyImplementation));
        vm.writeJson(eigenOutputFile, ".strategy.token.address", address(alignedStrategyToken));
    }
}
