package cmd

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcli "github.com/cometbft/cometbft/libs/cli"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/vexxvakan/vrf/app"
)

var (
	// Bech32Prefix defines the Bech32 prefix of an account's address
	Bech32Prefix = "vrf"
	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = Bech32Prefix
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = Bech32Prefix + sdk.PrefixPublic
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator + sdk.PrefixPublic
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus + sdk.PrefixPublic

	autoCliOpts autocli.AppOptions
)

// NewRootCmd creates a new root command for this chain. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	tempDir := tempDir()
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
	cfg.Seal()

	tempApp := app.New(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		tempDir,
		simtestutil.NewAppOptionsWithFlagHome(tempDir),
	)
	defer func() {
		if err := tempApp.BaseApp.Close(); err != nil {
			panic(err)
		}
		if tempDir != app.DefaultNodeHome {
			err := os.RemoveAll(tempDir)
			if err != nil {
				panic(err)
			}
		}
	}()

	initClientCtx := client.Context{}.
		WithCodec(tempApp.AppCodec()).
		WithInterfaceRegistry(tempApp.InterfaceRegistry()).
		WithTxConfig(tempApp.TxConfig()).
		WithLegacyAmino(tempApp.LegacyAmino()).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.FlagBroadcastMode).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   version.AppName,
		Short: "My Own Blockchain App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL. This sign mode
			// is only available if the client is online.
			if !initClientCtx.Offline {
				txConfigOpts := tx.ConfigOptions{
					EnabledSignModes:           append(tx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL),
					TextualCoinMetadataQueryFn: authtxconfig.NewGRPCCoinMetadataQueryFn(initClientCtx),
				}
				txConfigWithTextual, err := tx.NewTxConfigWithOptions(
					initClientCtx.Codec,
					txConfigOpts,
				)
				if err != nil {
					return err
				}
				initClientCtx = initClientCtx.WithTxConfig(txConfigWithTextual)
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			// 2 seconds + 1 second comet = 3 second blocks
			timeoutCommit := 1 * time.Second
			customAppTemplate, customAppConfig := initAppConfig()
			customCmtConfig := initCometConfig(timeoutCommit)

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCmtConfig)
		},
	}

	initRootCmd(
		rootCmd,
		tempApp.BasicModuleManager,
		tempApp.TxConfig(),
	)

	autoCliOpts = tempApp.AutoCLIOpts(initClientCtx)
	initClientCtx, err := config.ReadFromClientConfig(initClientCtx)
	if err != nil {
		panic(err)
	}

	autoCliOpts = enrichAutoCliOpts(autoCliOpts, initClientCtx)
	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd
}

func enrichAutoCliOpts(autoCliOpts autocli.AppOptions, clientCtx client.Context) autocli.AppOptions {
	autoCliOpts.AddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	autoCliOpts.ValidatorAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	autoCliOpts.ConsensusAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	autoCliOpts.ClientCtx = clientCtx

	return autoCliOpts
}

// initCometConfig helps to override default Comet Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometConfig(timeoutCommit time.Duration) *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	// While this is set, it only applies to new configs.
	cfg.Consensus.TimeoutCommit = timeoutCommit

	// better default logging
	cfg.LogLevel = "*:error,p2p:info,state:info"

	return cfg
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, any) {
	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = strings.Join(
		[]string{"0.075" + sdk.DefaultBondDenom}, ",")

	return serverconfig.DefaultConfigTemplate, srvCfg
}

func initRootCmd(
	rootCmd *cobra.Command,
	basicManager module.BasicManager,
	txConfig client.TxConfig,
) {
	rootCmd.AddCommand(
		genutilcli.InitCmd(basicManager, app.DefaultNodeHome),
		cmtcli.NewCompletionCmd(rootCmd, false),
		DebugCmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(
			newApp,
			app.DefaultNodeHome,
		),
		snapshot.Cmd(newApp),
	)

	server.AddCommands(
		rootCmd,
		app.DefaultNodeHome,
		newApp,
		appExport,
		addModuleInitFlags,
	)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		server.ShowValidatorCmd(),
		server.ShowNodeIDCmd(),
		server.ShowAddressCmd(),
		genesisCommand(txConfig, basicManager),
		queryCommand(),
		txCommand(),
		keys.Commands(),
	)
}
