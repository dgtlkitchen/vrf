package ante

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	vrfante "github.com/dgtlkitchen/vrf/x/vrf/ante"
	vrfkeeper "github.com/dgtlkitchen/vrf/x/vrf/keeper"
)

type SigVerificationDecoratorOption = authante.SigVerificationDecoratorOption

var DefaultSigVerificationGasConsumer = authante.DefaultSigVerificationGasConsumer

// HandlerOptions extends the SDK's auth ante options by adding VRF-specific
// wiring. In particular, it supports gasless emergency-disable transactions.
type HandlerOptions struct {
	AccountKeeper          authante.AccountKeeper
	BankKeeper             authtypes.BankKeeper
	ExtensionOptionChecker authante.ExtensionOptionChecker
	FeegrantKeeper         authante.FeegrantKeeper
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         func(meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	TxFeeChecker           authante.TxFeeChecker
	SigVerifyOptions       []authante.SigVerificationDecoratorOption

	VrfKeeper *vrfkeeper.Keeper
}

func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}
	if options.VrfKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "vrf keeper is required for ante builder")
	}
	accountKeeper, ok := options.AccountKeeper.(authkeeper.AccountKeeper)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrLogic, "account keeper must be authkeeper.AccountKeeper, got %T", options.AccountKeeper)
	}
	sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = authante.DefaultSigVerificationGasConsumer
	}

	anteDecorators := []sdk.AnteDecorator{
		authante.NewSetUpContextDecorator(),
		authante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(options.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),

		// VRF emergency-disable transactions must be gasless and bypass
		// sequence/nonce checks. This decorator performs deterministic signature
		// verification and short-circuits the ante chain for authorized txs.
		vrfante.NewEmergencyDisableDecorator(accountKeeper, options.VrfKeeper, options.SignModeHandler),

		authante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		authante.NewSetPubKeyDecorator(options.AccountKeeper),
		authante.NewValidateSigCountDecorator(options.AccountKeeper),
		authante.NewSigGasConsumeDecorator(options.AccountKeeper, sigGasConsumer),
		authante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler, options.SigVerifyOptions...),
		authante.NewIncrementSequenceDecorator(options.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
