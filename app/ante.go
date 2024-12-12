// SPDX-License-Identifier: BUSL-1.1-or-later
// SPDX-FileCopyrightText: 2024 Web3 Technologies Inc. <https://asphere.xyz/>
// Copyright (c) 2024 Web3 Technologies Inc. All rights reserved.
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

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	ethermintante "github.com/evmos/ethermint/app/ante"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper, CosmWasm keeper and Ethermint keeper.
type HandlerOptions struct {
	authante.HandlerOptions

	IBCKeeper *keeper.Keeper

	// CosmWasm
	WasmConfig            *wasmTypes.WasmConfig
	WasmKeeper            *wasmkeeper.Keeper
	TXCounterStoreService corestoretypes.KVStoreService
	CircuitKeeper         *circuitkeeper.Keeper

	// Ethermint
	FeeMarketKeeper   ethermintante.FeeMarketKeeper
	EvmKeeper         ethermintante.EVMKeeper
	DisabledAuthzMsgs []string
	MaxTxGasWanted    uint64
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

		defer ethermintante.Recover(ctx.Logger(), &err)

		// disable vesting message types
		for _, msg := range tx.GetMsgs() {
			switch msg.(type) {
			case *vestingtypes.MsgCreateVestingAccount,
				*vestingtypes.MsgCreatePeriodicVestingAccount,
				*vestingtypes.MsgCreatePermanentLockedAccount:
				return ctx, errors.New("vesting messages are not supported")
			}
		}

		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 0 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/ethermint.evm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler, err = newEthAnteHandler(options)
				case "/ethermint.types.v1.ExtensionOptionsWeb3Tx":
					// Deprecated: Handle as normal Cosmos SDK tx, except signature is checked for Legacy EIP712 representation
					anteHandler, err = newLegacyCosmosAnteHandlerEip712(options)
				case "/ethermint.types.v1.ExtensionOptionDynamicFeeTx":
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
	evmAccountKeeper, ok := options.AccountKeeper.(evmtypes.AccountKeeper)
	if !ok {
		return nil, errors.New("account keeper does not implement evmtypes.AccountKeeper")
	}

	return sdk.ChainAnteDecorators(
		ethermintante.NewEthSetUpContextDecorator(options.EvmKeeper),                         // outermost AnteDecorator. SetUpContext must be called first
		ethermintante.NewEthMempoolFeeDecorator(options.EvmKeeper),                           // Check eth effective gas price against minimal-gas-prices
		ethermintante.NewEthMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper), // Check eth effective gas price against the global MinGasPrice
		ethermintante.NewEthValidateBasicDecorator(options.EvmKeeper),
		ethermintante.NewEthSigVerificationDecorator(options.EvmKeeper),
		ethermintante.NewEthAccountVerificationDecorator(evmAccountKeeper, options.EvmKeeper),
		ethermintante.NewCanTransferDecorator(options.EvmKeeper),
		ethermintante.NewEthGasConsumeDecorator(options.EvmKeeper, options.MaxTxGasWanted),
		ethermintante.NewEthIncrementSenderSequenceDecorator(evmAccountKeeper), // innermost AnteDecorator.
		ethermintante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
		ethermintante.NewEthEmitEventDecorator(options.EvmKeeper), // emit eth tx hash and index at the very last ante handler.
	), nil
}

func newCosmosAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	return sdk.ChainAnteDecorators(
		authante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		authante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(options.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		authante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, nil),
		// SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(options.AccountKeeper),
		authante.NewValidateSigCountDecorator(options.AccountKeeper),
		authante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		authante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		authante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	), nil
}

// Deprecated: NewLegacyCosmosAnteHandlerEip712 creates an AnteHandler to process legacy EIP-712
// transactions, as defined by the presence of an ExtensionOptionsWeb3Tx extension.
func newLegacyCosmosAnteHandlerEip712(options HandlerOptions) (sdk.AnteHandler, error) {
	evmAccountKeeper, ok := options.AccountKeeper.(evmtypes.AccountKeeper)
	if !ok {
		return nil, errors.New("account keeper does not implement evmtypes.AccountKeeper")
	}

	return sdk.ChainAnteDecorators(
		ethermintante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		// disable the Msg types that cannot be included on an authz.MsgExec msgs field
		ethermintante.NewAuthzLimiterDecorator(options.DisabledAuthzMsgs),
		authante.NewSetUpContextDecorator(),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		authante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		ethermintante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		authante.NewValidateMemoDecorator(options.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		authante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(options.AccountKeeper),
		authante.NewValidateSigCountDecorator(options.AccountKeeper),
		authante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		// Note: signature verification uses EIP instead of the cosmos signature validator
		ethermintante.NewLegacyEip712SigVerificationDecorator(evmAccountKeeper, options.SignModeHandler),
		authante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		ethermintante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	), nil
}
