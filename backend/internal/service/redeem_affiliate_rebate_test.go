//go:build unit

package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

type redeemAffiliateAccruerSpy struct {
	calls      int
	userID     int64
	baseAmount float64
	txSeen     bool
}

func (s *redeemAffiliateAccruerSpy) AccrueInviteRebate(ctx context.Context, inviteeUserID int64, baseRechargeAmount float64) (float64, error) {
	s.calls++
	s.userID = inviteeUserID
	s.baseAmount = baseRechargeAmount
	s.txSeen = dbent.TxFromContext(ctx) != nil
	return 10, nil
}

func TestRedeemBalanceAccruesAffiliateRebateForPositiveBalanceCode(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	const userID int64 = 42
	redeemRepo := &paymentOrderLifecycleRedeemRepo{
		codesByCode: map[string]*RedeemCode{
			"SELF-SERVICE-BALANCE": {
				ID:     1,
				Code:   "SELF-SERVICE-BALANCE",
				Type:   RedeemTypeBalance,
				Value:  50,
				Status: StatusUnused,
			},
		},
	}
	userRepo := &mockUserRepo{
		getByIDUser: &User{ID: userID, Balance: 0},
	}
	var credited float64
	userRepo.updateBalanceFn = func(_ context.Context, gotUserID int64, amount float64) error {
		require.Equal(t, userID, gotUserID)
		credited += amount
		return nil
	}
	affiliateSpy := &redeemAffiliateAccruerSpy{}
	svc := NewRedeemService(
		redeemRepo,
		userRepo,
		nil,
		nil,
		nil,
		client,
		nil,
		affiliateSpy,
	)

	got, err := svc.Redeem(ctx, userID, "SELF-SERVICE-BALANCE")

	require.NoError(t, err)
	require.Equal(t, StatusUsed, got.Status)
	require.Equal(t, 50.0, credited)
	require.Equal(t, 1, affiliateSpy.calls)
	require.Equal(t, userID, affiliateSpy.userID)
	require.Equal(t, 50.0, affiliateSpy.baseAmount)
	require.True(t, affiliateSpy.txSeen, "affiliate rebate should be accrued inside the redeem transaction")
}

func TestRedeemBalanceDoesNotAccrueAffiliateRebateForNegativeBalanceCode(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	const userID int64 = 43
	redeemRepo := &paymentOrderLifecycleRedeemRepo{
		codesByCode: map[string]*RedeemCode{
			"REFUND-BALANCE": {
				ID:     2,
				Code:   "REFUND-BALANCE",
				Type:   RedeemTypeBalance,
				Value:  -30,
				Status: StatusUnused,
			},
		},
	}
	userRepo := &mockUserRepo{
		getByIDUser: &User{ID: userID, Balance: 100},
	}
	userRepo.updateBalanceFn = func(_ context.Context, gotUserID int64, amount float64) error {
		require.Equal(t, userID, gotUserID)
		require.Equal(t, -30.0, amount)
		return nil
	}
	affiliateSpy := &redeemAffiliateAccruerSpy{}
	svc := NewRedeemService(
		redeemRepo,
		userRepo,
		nil,
		nil,
		nil,
		client,
		nil,
		affiliateSpy,
	)

	_, err := svc.Redeem(ctx, userID, "REFUND-BALANCE")

	require.NoError(t, err)
	require.Equal(t, 0, affiliateSpy.calls)
}

func TestRedeemBalanceCanSuppressAffiliateRebateForPaymentFulfillment(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	const userID int64 = 44
	redeemRepo := &paymentOrderLifecycleRedeemRepo{
		codesByCode: map[string]*RedeemCode{
			"INTERNAL-PAYMENT-BALANCE": {
				ID:     3,
				Code:   "INTERNAL-PAYMENT-BALANCE",
				Type:   RedeemTypeBalance,
				Value:  80,
				Status: StatusUnused,
			},
		},
	}
	userRepo := &mockUserRepo{
		getByIDUser: &User{ID: userID, Balance: 0},
	}
	userRepo.updateBalanceFn = func(_ context.Context, gotUserID int64, amount float64) error {
		require.Equal(t, userID, gotUserID)
		require.Equal(t, 80.0, amount)
		return nil
	}
	affiliateSpy := &redeemAffiliateAccruerSpy{}
	svc := NewRedeemService(
		redeemRepo,
		userRepo,
		nil,
		nil,
		nil,
		client,
		nil,
		affiliateSpy,
	)

	_, err := svc.Redeem(withRedeemAffiliateRebateSuppressed(ctx), userID, "INTERNAL-PAYMENT-BALANCE")

	require.NoError(t, err)
	require.Equal(t, 0, affiliateSpy.calls)
}
