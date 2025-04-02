// SPDX-License-Identifier: BUSL-1.1-or-later
// SPDX-FileCopyrightText: 2025 Web3 Technologies Inc. <https://asphere.xyz/>
// Copyright (c) 2025 Web3 Technologies Inc. All rights reserved.
// Use of this software is governed by the Business Source License included in the LICENSE file <https://github.com/Asphere-xyz/tacchain/blob/main/LICENSE>.
package app

import (
	"errors"
	"fmt"

	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"

	corestoretypes "cosmossdk.io/core/store"
	circuitante "cosmossdk.io/x/circuit/ante"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	// vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	// bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	evmcosmosante "github.com/cosmos/evm/ante/cosmos"
	evmante "github.com/cosmos/evm/ante/evm"
	evmanteinterfaces "github.com/cosmos/evm/ante/interfaces"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper, CosmWasm keeper and Ethermint keeper.
type HandlerOptions struct {
	authante.HandlerOptions

	AccountKeeper evmanteinterfaces.AccountKeeper

	IBCKeeper *keeper.Keeper

	// CosmWasm
	WasmConfig            *wasmTypes.NodeConfig
	WasmKeeper            *wasmkeeper.Keeper
	TXCounterStoreService corestoretypes.KVStoreService
	CircuitKeeper         *circuitkeeper.Keeper

	// Ethermint

	FeeMarketKeeper evmanteinterfaces.FeeMarketKeeper
	EvmKeeper       evmanteinterfaces.EVMKeeper
	MaxTxGasWanted  uint64
}

// NewAnteHandler returns an ante handler responsible for attempting to route an
// Ethereum or SDK transaction to an internal ante handler for performing
// transaction-level processing (e.g. fee payment, signature verification) before
// being passed onto it's respective handler.
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errors.New("account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return nil, errors.New("bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return nil, errors.New("sign mode handler is required for ante builder")
	}
	if options.WasmConfig == nil {
		return nil, errors.New("wasm config is required for ante builder")
	}
	if options.TXCounterStoreService == nil {
		return nil, errors.New("wasm store service is required for ante builder")
	}
	if options.CircuitKeeper == nil {
		return nil, errors.New("circuit keeper is required for ante builder")
	}
	if options.FeeMarketKeeper == nil {
		return nil, errors.New("fee market keeper is required for ante builder")
	}
	if options.EvmKeeper == nil {
		return nil, errors.New("evm keeper is required for ante builder")
	}

	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		var anteHandler sdk.AnteHandler

		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 0 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler = sdk.ChainAnteDecorators(
						evmante.NewEVMMonoDecorator(
							options.AccountKeeper,
							options.FeeMarketKeeper,
							options.EvmKeeper,
							options.MaxTxGasWanted,
						),
					)
				// TODO: test if EIP712 works (should as per https://github.com/cosmos/evm/blob/main/ante/evm/ante_test.go#L263)
				// if not, uncomment newLegacyCosmosAnteHandlerEip712 and replace eip handler with https://github.com/cosmos/evm/blob/main/ante/cosmos/eip712.go#L47
				// case "/cosmos.evm.vm.v1.ExtensionOptionsWeb3Tx":
				// Deprecated: Handle as normal Cosmos SDK tx, except signature is checked for Legacy EIP712 representation
				// anteHandler, err = newLegacyCosmosAnteHandlerEip712(options)
				case "/cosmos.evm.types.v1.ExtensionOptionDynamicFeeTx":
					// cosmos-sdk tx with dynamic fee extension
					anteHandler, err = newCosmosAnteHandler(options)
				default:
					return ctx, errors.New(fmt.Sprintf("rejecting tx with unsupported extension option: %s", typeURL))
				}

				if err != nil {
					return ctx, err
				}

				return anteHandler(ctx, tx, sim)
			}
		}

		// handle as totally normal Cosmos SDK tx
		switch tx.(type) {
		case sdk.Tx:
			anteHandler, err = newCosmosAnteHandler(options)
		default:
			return ctx, errors.New("invalid transaction type")
		}

		if err != nil {
			return ctx, err
		}

		return anteHandler(ctx, tx, sim)
	}, nil
}

func newEthAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	// evmAccountKeeper, ok := options.AccountKeeper.(evmtypes.AccountKeeper)
	// if !ok {
	// 	return nil, errors.New("account keeper does not implement evmtypes.AccountKeeper")
	// }

	return sdk.ChainAnteDecorators(
	// ethermintante.NewEthSetUpContextDecorator(options.EvmKeeper),                         // outermost AnteDecorator. SetUpContext must be called first
	// ethermintante.NewEthMempoolFeeDecorator(options.EvmKeeper),                           // Check eth effective gas price against minimal-gas-prices
	// ethermintante.NewEthMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper), // Check eth effective gas price against the global MinGasPrice
	// ethermintante.NewEthValidateBasicDecorator(options.EvmKeeper),
	// ethermintante.NewEthSigVerificationDecorator(options.EvmKeeper),
	// ethermintante.NewEthAccountVerificationDecorator(evmAccountKeeper, options.EvmKeeper),
	// ethermintante.NewCanTransferDecorator(options.EvmKeeper),
	// ethermintante.NewEthGasConsumeDecorator(options.EvmKeeper, options.MaxTxGasWanted),
	// ethermintante.NewEthIncrementSenderSequenceDecorator(evmAccountKeeper), // innermost AnteDecorator.
	// ethermintante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	// ethermintante.NewEthEmitEventDecorator(options.EvmKeeper), // emit eth tx hash and index at the very last ante handler.
	), nil
}

func newCosmosAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	return sdk.ChainAnteDecorators(
		evmcosmosante.NewRejectMessagesDecorator(), // reject MsgEthereumTxs
		evmcosmosante.NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
		),
		authante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
		wasmkeeper.NewTxContractsDecorator(),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		authante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(options.AccountKeeper),
		evmcosmosante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		authante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		authante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(options.AccountKeeper),
		authante.NewValidateSigCountDecorator(options.AccountKeeper),
		authante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		authante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		authante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		evmante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	), nil
}

// TODO: test if EIP712 works (should as per https://github.com/cosmos/evm/blob/main/ante/evm/ante_test.go#L263)
// if not, uncomment newLegacyCosmosAnteHandlerEip712 and replace eip handler with https://github.com/cosmos/evm/blob/main/ante/cosmos/eip712.go#L47
// Deprecated: NewLegacyCosmosAnteHandlerEip712 creates an AnteHandler to process legacy EIP-712
// transactions, as defined by the presence of an ExtensionOptionsWeb3Tx extension.
// func newLegacyCosmosAnteHandlerEip712(options HandlerOptions) (sdk.AnteHandler, error) {
// 	evmAccountKeeper, ok := options.AccountKeeper.(evmtypes.AccountKeeper)
// 	if !ok {
// 		return nil, errors.New("account keeper does not implement evmtypes.AccountKeeper")
// 	}

// 	ethermintOptions := ethermintante.HandlerOptions{
// 		AccountKeeper:          evmAccountKeeper,
// 		BankKeeper:             options.BankKeeper.(bankkeeper.BaseKeeper),
// 		SignModeHandler:        options.SignModeHandler,
// 		FeegrantKeeper:         options.HandlerOptions.FeegrantKeeper,
// 		SigGasConsumer:         ethermintante.DefaultSigVerificationGasConsumer,
// 		IBCKeeper:              options.IBCKeeper,
// 		EvmKeeper:              options.EvmKeeper,
// 		FeeMarketKeeper:        options.FeeMarketKeeper,
// 		MaxTxGasWanted:         options.MaxTxGasWanted,
// 		ExtensionOptionChecker: etherminttypes.HasDynamicFeeExtensionOption,
// 		TxFeeChecker:           ethermintante.NewDynamicFeeChecker(options.EvmKeeper),
// 		DisabledAuthzMsgs: []string{
// 			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
// 			sdk.MsgTypeURL(&vestingtypes.MsgCreateVestingAccount{}),
// 			sdk.MsgTypeURL(&vestingtypes.MsgCreatePermanentLockedAccount{}),
// 			sdk.MsgTypeURL(&vestingtypes.MsgCreatePeriodicVestingAccount{}),
// 		},
// 	}

// 	// Deprecated: Handle as normal Cosmos SDK tx, except signature is checked for Legacy EIP712 representation
// 	return ethermintante.NewLegacyCosmosAnteHandlerEip712(ethermintOptions), nil
// }
