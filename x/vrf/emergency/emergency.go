package emergency

import (
	"fmt"

	txsigning "cosmossdk.io/x/tx/signing"

	"google.golang.org/protobuf/types/known/anypb"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	vrfkeeper "github.com/vexxvakan/vrf/x/vrf/keeper"
	vrftypes "github.com/vexxvakan/vrf/x/vrf/types"
)

// VerifyEmergencyMsg performs the shared, deterministic authorization check
// for MsgVrfEmergencyDisable, as described in PRD §4.3.2.
//
// It is intended to be used both:
//   - In PreBlock (to decide whether to bypass VRF for the block), and
//   - In the Ante/DeliverTx path (to accept or reject the transaction).
//
// The function returns:
//   - found:      whether the tx contained at least one MsgVrfEmergencyDisable.
//   - authorized: whether at least one such message was signed by an
//     allow-listed authority.
//   - reason:     the free-form reason string from the first authorized msg.
func VerifyEmergencyMsg(
	ctx sdk.Context,
	tx sdk.Tx,
	ak authkeeper.AccountKeeper,
	vk *vrfkeeper.Keeper,
	signModeHandler *txsigning.HandlerMap,
) (found bool, authorized bool, reason string, err error) {
	if signModeHandler == nil {
		return false, false, "", fmt.Errorf("vrf: nil sign mode handler")
	}

	sigTx, ok := tx.(authsigning.Tx)
	if !ok {
		// Not a signable tx; treat as not containing an emergency message.
		return false, false, "", nil
	}

	msgs := sigTx.GetMsgs()
	var emergencyMsgs []*vrftypes.MsgVrfEmergencyDisable
	for _, msg := range msgs {
		emergencyMsg, ok := msg.(*vrftypes.MsgVrfEmergencyDisable)
		if ok {
			emergencyMsgs = append(emergencyMsgs, emergencyMsg)
		}
	}

	if len(emergencyMsgs) == 0 {
		return false, false, "", nil
	}

	// Emergency disable must be a dedicated transaction so that bypassing fees and
	// sequence checks cannot inadvertently apply to non-emergency messages.
	if len(emergencyMsgs) != len(msgs) {
		return true, false, "", fmt.Errorf("vrf: emergency disable tx must contain only MsgVrfEmergencyDisable messages")
	}

	// At this point we know the tx includes at least one MsgVrfEmergencyDisable.
	// Perform full signature verification using the same primitives as the
	// standard auth ante handlers, but without enforcing sequence-equality
	// checks (sequence is taken from the signature itself).
	if err := verifySignatures(ctx, sigTx, ak, signModeHandler); err != nil {
		return true, false, "", err
	}

	// Check allowlist: for each MsgVrfEmergencyDisable, ensure that the signer is
	// present in x/vrf's committee allowlist.
	for _, m := range emergencyMsgs {
		if err := m.ValidateBasic(); err != nil {
			return true, false, "", err
		}

		ok, err := vk.IsCommitteeMember(ctx.Context(), m.Authority)
		if err != nil {
			return true, false, "", err
		}

		if ok {
			return true, true, m.Reason, nil
		}
	}

	return true, false, "", nil
}

// verifySignatures verifies all signatures on the transaction using the x/tx
// HandlerMap. It is modeled after x/auth/ante's SigVerificationDecorator but
// deliberately does not enforce sequence equality between signatures and
// on-chain accounts, so that emergency messages can bypass sequence/nonce
func verifySignatures(
	ctx sdk.Context,
	sigTx authsigning.Tx,
	ak authkeeper.AccountKeeper,
	signModeHandler *txsigning.HandlerMap,
) error {
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return err
	}

	signers, err := sigTx.GetSigners()
	if err != nil {
		return err
	}

	if len(sigs) != len(signers) {
		return fmt.Errorf("vrf: invalid number of signers; expected %d, got %d", len(signers), len(sigs))
	}

	pubKeys, err := sigTx.GetPubKeys()
	if err != nil {
		return err
	}

	adaptableTx, ok := sigTx.(authsigning.V2AdaptableTx)
	if !ok {
		return fmt.Errorf("vrf: expected tx to implement V2AdaptableTx, got %T", sigTx)
	}
	txData := adaptableTx.GetSigningTxData()

	chainID := ctx.ChainID()

	for i, sig := range sigs {
		addr := sdk.AccAddress(signers[i])

		acc := ak.GetAccount(ctx.Context(), addr)
		if acc == nil {
			return fmt.Errorf("vrf: signer account %s does not exist", addr.String())
		}

		pubKey := pubKeys[i]
		if pubKey == nil {
			pubKey = acc.GetPubKey()
		}

		if pubKey == nil {
			return fmt.Errorf("vrf: missing public key for signer %s", addr.String())
		}

		anyPk, err := codectypes.NewAnyWithValue(pubKey)
		if err != nil {
			return err
		}

		signerData := txsigning.SignerData{
			Address:       addr.String(),
			ChainID:       chainID,
			AccountNumber: acc.GetAccountNumber(),
			// IMPORTANT: use the sequence from the signature itself instead of the
			// on-chain account so that MsgVrfEmergencyDisable bypasses sequence/nonce checks
			Sequence: sig.Sequence,
			PubKey: &anypb.Any{
				TypeUrl: anyPk.TypeUrl,
				Value:   anyPk.Value,
			},
		}

		if err := authsigning.VerifySignature(ctx, pubKey, signerData, sig.Data, signModeHandler, txData); err != nil {
			return err
		}
	}

	return nil
}
