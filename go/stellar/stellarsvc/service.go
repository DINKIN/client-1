package stellarsvc

import (
	"context"
	"errors"

	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/protocol/keybase1"
	"github.com/keybase/client/go/protocol/stellar1"
	"github.com/keybase/client/go/stellar"
	"github.com/keybase/client/go/stellar/remote"
	"github.com/keybase/stellarnet"
)

// Service handlers

// CreateWallet creates and posts an initial stellar bundle for a user.
// Only succeeds if they do not already have one.
// Safe to call even if the user has a bundle already.
func CreateWallet(ctx context.Context, g *libkb.GlobalContext) (created bool, err error) {
	return stellar.CreateWallet(ctx, g)
}

type balancesResult struct {
	Status   libkb.AppStatus    `json:"status"`
	Balances []stellar1.Balance `json:"balances"`
}

func (b *balancesResult) GetAppStatus() *libkb.AppStatus {
	return &b.Status
}

func Balances(ctx context.Context, g *libkb.GlobalContext, accountID stellar1.AccountID) ([]stellar1.Balance, error) {
	apiArg := libkb.APIArg{
		Endpoint:    "stellar/balances",
		SessionType: libkb.APISessionTypeREQUIRED,
		Args:        libkb.HTTPArgs{"account_id": libkb.S{Val: string(accountID)}},
		NetContext:  ctx,
	}

	var res balancesResult
	if err := g.API.GetDecode(apiArg, &res); err != nil {
		return nil, err
	}

	return res.Balances, nil
}

func balanceXLM(ctx context.Context, g *libkb.GlobalContext, accountID stellar1.AccountID) (stellar1.Balance, error) {
	balances, err := Balances(ctx, g, accountID)
	if err != nil {
		return stellar1.Balance{}, err
	}

	for _, b := range balances {
		if b.Asset.Type == "native" {
			return b, nil
		}
	}

	return stellar1.Balance{}, errors.New("no native balance")
}

func lookupSenderPrimary(ctx context.Context, g *libkb.GlobalContext) (keybase1.StellarEntry, error) {
	bundle, err := remote.Fetch(ctx, g)
	if err != nil {
		return keybase1.StellarEntry{}, err
	}

	primary, err := bundle.PrimaryAccount()
	if err != nil {
		return keybase1.StellarEntry{}, err
	}
	if len(primary.Signers) == 0 {
		return keybase1.StellarEntry{}, errors.New("no signer for primary bundle")
	}
	if len(primary.Signers) > 1 {
		return keybase1.StellarEntry{}, errors.New("only single signer supported")
	}

	return primary, nil
}

type Recipient struct {
	Input     string
	User      *libkb.User
	AccountID stellarnet.AddressStr
}

// TODO: actually lookup the recipient and handle
// 1. stellar addresses GXXXXX
// 2. stellar federation address rebecca*keybase.io
// 3. keybase username
// 4. keybase assertion
func lookupRecipient(ctx context.Context, g *libkb.GlobalContext, to string) (*Recipient, error) {
	r := Recipient{
		Input: to,
	}

	// temporary: only handle stellar addresses
	var err error
	r.AccountID, err = stellarnet.NewAddressStr(to)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func Send(ctx context.Context, g *libkb.GlobalContext, arg stellar1.SendLocalArg) (stellar1.PaymentResult, error) {
	// look up sender wallet
	primary, err := lookupSenderPrimary(ctx, g)
	if err != nil {
		return stellar1.PaymentResult{}, err
	}
	primaryAccountID, err := stellarnet.NewAddressStr(primary.AccountID.String())
	if err != nil {
		return stellar1.PaymentResult{}, err
	}
	primarySeed, err := stellarnet.NewSeedStr(primary.Signers[0].SecureNoLogString())
	if err != nil {
		return stellar1.PaymentResult{}, err
	}
	senderAcct := stellarnet.NewAccount(primaryAccountID)

	// look up recipient
	recipient, err := lookupRecipient(ctx, g, arg.Recipient)
	if err != nil {
		return stellar1.PaymentResult{}, err
	}

	post := stellar1.PaymentPost{
		Members: stellar1.Members{
			FromStellar: stellar1.AccountID(primaryAccountID.String()),
			// FromKeybase: "XXXmyusernameXXX",  XXX TODO
			// FromUID:     "XXXmyuidXXX",      // XXX TODO
			// FromDeviceID: "", // XXX TODO
			ToStellar: stellar1.AccountID(recipient.AccountID.String()),
		},
	}

	// check if recipient account exists
	_, err = balanceXLM(ctx, g, stellar1.AccountID(recipient.AccountID.String()))
	if err != nil {
		// if no balance, create_account
		// if amount < 1 XLM, return error
		post.StellarAccountSeqno, post.SignedTransaction, err = senderAcct.CreateAccountXLMTransaction(primarySeed, recipient.AccountID, arg.Amount)
		if err != nil {
			return stellar1.PaymentResult{}, err
		}
	} else {
		// if balance, payment tx
		post.StellarAccountSeqno, post.SignedTransaction, err = senderAcct.PaymentXLMTransaction(primarySeed, recipient.AccountID, arg.Amount)
		if err != nil {
			return stellar1.PaymentResult{}, err
		}
	}

	return stellar1.PaymentResult{}, nil
}
