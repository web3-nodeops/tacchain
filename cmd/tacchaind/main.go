// SPDX-License-Identifier: BUSL-1.1-or-later
// SPDX-FileCopyrightText: 2024 Web3 Technologies Inc. <https://asphere.xyz/>
// Copyright (c) 2024 Web3 Technologies Inc. All rights reserved.
// Use of this software is governed by the Business Source License included in the LICENSE file <https://github.com/Asphere-xyz/tacchain/blob/main/LICENSE>.
package main

import (
	"os"

	"cosmossdk.io/log"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/Asphere-xyz/tacchain/app"
)

func main() {
	rootCmd := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		log.NewLogger(rootCmd.OutOrStderr()).Error("failure when running app", "err", err)
		os.Exit(1)
	}
}
