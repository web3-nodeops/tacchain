// SPDX-License-Identifier: BUSL-1.1-or-later
// SPDX-FileCopyrightText: 2025 Web3 Technologies Inc. <https://asphere.xyz/>
// Copyright (c) 2025 Web3 Technologies Inc. All rights reserved.
// Use of this software is governed by the Business Source License included in the LICENSE file <https://github.com/Asphere-xyz/tacchain/blob/main/LICENSE>.
package app

import (
	"math/big"
	"os"

	simappparams "cosmossdk.io/simapp/params"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/gogoproto/proto"
	evmv1 "github.com/evmos/ethermint/api/ethermint/evm/v1"
	ethermintcmdcfg "github.com/evmos/ethermint/cmd/config"
	ethermintcodec "github.com/evmos/ethermint/encoding/codec"
	"github.com/evmos/ethermint/ethereum/eip712"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	protov2 "google.golang.org/protobuf/proto"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DisplayDenom  = "tac"
	BaseDenom     = "utac"
	BaseDenomUnit = 18

	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address.
	Bech32PrefixAccAddr = "tac"

	NodeDir        = ".tacchaind"
	AppName        = "TacChainApp"
	DefaultChainID = "tacchain_2390-1"
)

var (
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key.
	Bech32PrefixAccPub = Bech32PrefixAccAddr + "pub"
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address.
	Bech32PrefixValAddr = Bech32PrefixAccAddr + "valoper"
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key.
	Bech32PrefixValPub = Bech32PrefixAccAddr + "valoperpub"
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address.
	Bech32PrefixConsAddr = Bech32PrefixAccAddr + "valcons"
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key.
	Bech32PrefixConsPub = Bech32PrefixAccAddr + "valconspub"

	DefaultNodeHome = os.ExpandEnv("$HOME/") + NodeDir

	// PowerReduction defines the default power reduction value for staking
	PowerReduction = sdkmath.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(BaseDenomUnit), nil))
)

func init() {
	SetAddressPrefixes()
	RegisterDenoms()
}

// RegisterDenoms registers token denoms.
func RegisterDenoms() {
	sdk.DefaultBondDenom = BaseDenom
	sdk.DefaultPowerReduction = PowerReduction

	config := sdk.GetConfig()
	ethermintcmdcfg.SetBip44CoinType(config)

	if err := sdk.RegisterDenom(DisplayDenom, sdkmath.LegacyOneDec()); err != nil {
		panic(err)
	}

	if err := sdk.RegisterDenom(BaseDenom, sdkmath.LegacyNewDecWithPrec(1, BaseDenomUnit)); err != nil {
		panic(err)
	}
}

// SetAddressPrefixes builds the Config with Bech32 addressPrefix and publKeyPrefix for accounts, validators, and consensus nodes and verifies that addreeses have correct format.
func SetAddressPrefixes() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
	config.SetAddressVerifier(wasmtypes.VerifyAddressLen())
}

func MakeEncodingConfig() simappparams.EncodingConfig {
	signingOptions := signing.Options{
		AddressCodec: address.Bech32Codec{
			Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
		},
		ValidatorAddressCodec: address.Bech32Codec{
			Bech32Prefix: sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
		},
	}

	// evm/MsgEthereumTx
	signingOptions.DefineCustomGetSigners(protov2.MessageName(&evmv1.MsgEthereumTx{}), evmtypes.GetSignersFromMsgEthereumTxV2)

	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: signingOptions,
	})
	if err != nil {
		panic(err)
	}

	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	encodingConfig := simappparams.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             appCodec,
		TxConfig:          authtx.NewTxConfig(appCodec, authtx.DefaultSignModes),
		Amino:             legacyAmino,
	}

	ethermintcodec.RegisterLegacyAminoCodec(legacyAmino)
	ethermintcodec.RegisterInterfaces(interfaceRegistry)

	legacytx.RegressionTestingAminoCodec = legacyAmino

	eip712.SetEncodingConfig(encodingConfig)

	return encodingConfig
}
