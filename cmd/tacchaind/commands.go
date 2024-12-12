// SPDX-License-Identifier: BUSL-1.1-or-later
// SPDX-FileCopyrightText: 2024 Web3 Technologies Inc. <https://asphere.xyz/>
// Copyright (c) 2024 Web3 Technologies Inc. All rights reserved.
// Use of this software is governed by the Business Source License included in the LICENSE file <https://github.com/Asphere-xyz/tacchain/blob/main/LICENSE>.
package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcli "github.com/cometbft/cometbft/libs/cli"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cosmossdk.io/log"
	"cosmossdk.io/simapp/params"
	confixcmd "cosmossdk.io/tools/confix/cmd"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	ethermintdebug "github.com/evmos/ethermint/client/debug"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/Asphere-xyz/tacchain/app"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmcli "github.com/CosmWasm/wasmd/x/wasm/client/cli"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	ethermintclient "github.com/evmos/ethermint/client"
	ethermintserver "github.com/evmos/ethermint/server"
	ethermintserverconfig "github.com/evmos/ethermint/server/config"
)

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// The following code snippet is just for reference.

	type CustomAppConfig struct {
		serverconfig.Config

		Wasm wasmtypes.WasmConfig `mapstructure:"wasm"`

		// Ethermint config
		EVM     ethermintserverconfig.EVMConfig     `mapstructure:"evm"`
		JSONRPC ethermintserverconfig.JSONRPCConfig `mapstructure:"json-rpc"`
		TLS     ethermintserverconfig.TLSConfig     `mapstructure:"tls"`
	}

	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In TacChain, we set the min gas prices to 0.
	srvCfg.MinGasPrices = "0" + sdk.DefaultBondDenom

	// srvCfg.BaseConfig.IAVLDisableFastNode = true // disable fastnode by default

	customAppConfig := CustomAppConfig{
		Config:  *srvCfg,
		Wasm:    wasmtypes.DefaultWasmConfig(),
		EVM:     *ethermintserverconfig.DefaultEVMConfig(),
		JSONRPC: *ethermintserverconfig.DefaultJSONRPCConfig(),
		TLS:     *ethermintserverconfig.DefaultTLSConfig(),
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate +
		wasmtypes.DefaultConfigTemplate() +
		ethermintserverconfig.DefaultConfigTemplate

	return customAppTemplate, customAppConfig
}

func initRootCmd(
	appInstance *app.TacChainApp,
	encodingConfig params.EncodingConfig,
	rootCmd *cobra.Command,
	txConfig client.TxConfig,
	basicManager module.BasicManager,
) {
	cfg := sdk.GetConfig()
	cfg.Seal()

	rootCmd.AddCommand(
		ethermintclient.ValidateChainID(
			genutilcli.InitCmd(appInstance.BasicModuleManager, app.DefaultNodeHome),
		),
		cmtcli.NewCompletionCmd(rootCmd, true),
		NewTestnetCmd(basicManager, banktypes.GenesisBalancesIterator{}),
		ethermintdebug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(newApp, app.DefaultNodeHome),
		snapshot.Cmd(newApp),
	)

	wasmcli.ExtendUnsafeResetAllCmd(rootCmd)

	ethermintserver.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(txConfig, basicManager),
		queryCommand(),
		txCommand(),
		ethermintclient.KeyCommands(app.DefaultNodeHome),
	)
}

func addModuleInitFlags(cmd *cobra.Command) {
	crisis.AddModuleInitFlags(cmd)
	wasm.AddModuleInitFlags(cmd)
}

// genesisCommand builds genesis-related `tacchaind genesis` command. Users may provide application specific commands as a parameter
func genesisCommand(txConfig client.TxConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	genesisCmd := genutilcli.Commands(txConfig, basicManager, app.DefaultNodeHome)

	// Replace add-genesis-account cmd with our own (./genaccounts.go)
	// This main difference with the original AddGenesisAccountCmd is that it assigns genesis account to EthAccount instead of BaseAccount.
	// Command logic taken from https://github.com/b-harvest/ethermint/blob/v0.22.0-sdk50-1/cmd/ethermintd/genaccounts.go
	// Logic is copy-pasted as we cannot access ethermint's AddGenesisAccountCmd(), because it is located in main package
	for _, subCmd := range genesisCmd.Commands() {
		if subCmd.Name() == "add-genesis-account" {
			genesisCmd.RemoveCommand(subCmd)
			genesisCmd.AddCommand(AddGenesisAccountCmd(app.DefaultNodeHome))
			break
		}
	}

	for _, subCmd := range cmds {
		genesisCmd.AddCommand(subCmd)
	}

	return genesisCmd
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		rpc.ValidatorCommand(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// newApp creates the application
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	if chainID := cast.ToString(appOpts.Get(flags.FlagChainID)); len(chainID) == 0 {
		v := viper.New()
		home := cast.ToString(appOpts.Get(flags.FlagHome))
		v.AddConfigPath(filepath.Join(home, "config"))
		v.SetConfigName("client")
		v.SetConfigType("toml")

		if err := v.ReadInConfig(); err != nil {
			panic(err)
		}

		conf := new(config.ClientConfig)
		if err := v.Unmarshal(conf); err != nil {
			panic(err)
		}

		appOptsViper, ok := appOpts.(*viper.Viper)
		if !ok {
			panic("appOpts is not viper.Viper")
		}
		appOptsViper.Set(flags.FlagChainID, conf.ChainID)
		appOpts = appOptsViper
	}

	baseappOptions := server.DefaultBaseappOptions(appOpts)

	var wasmOpts []wasmkeeper.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}

	return app.NewTacChainApp(
		logger,
		db,
		traceStore,
		true,
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		appOpts,
		wasmOpts,
		baseappOptions...,
	)
}

// appExport creates a new TacChain app (optionally at a given height) and exports state.
func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var tacChainApp *app.TacChainApp
	// this check is necessary as we use the flag in x/upgrade.
	// we can exit more gracefully by checking the flag here.
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home is not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}

	// overwrite the FlagInvCheckPeriod
	viperAppOpts.Set(server.FlagInvCheckPeriod, 1)
	appOpts = viperAppOpts

	var emptyWasmOpts []wasmkeeper.Option
	tacChainApp = app.NewTacChainApp(
		logger,
		db,
		traceStore,
		height == -1,
		uint(1),
		appOpts,
		emptyWasmOpts,
		baseapp.SetChainID(app.DefaultChainID),
	)

	if height != -1 {
		if err := tacChainApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return tacChainApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

func tempDir() string {
	dir, err := os.MkdirTemp("", ".tacchaind")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(dir)
	return dir
}
