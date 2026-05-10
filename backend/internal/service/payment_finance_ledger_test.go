package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/payment"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

func TestFinanceLedgerSummaryClassifiesRedeemCodesAndOrders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newFinanceLedgerTestClient(t)
	userA := createFinanceLedgerUser(t, ctx, client, "alice@example.com")
	userB := createFinanceLedgerUser(t, ctx, client, "bob@example.com")
	group := client.Group.Create().
		SetName("Codex Plus").
		SetPlatform(domain.PlatformOpenAI).
		SetStatus(domain.StatusActive).
		SaveX(ctx)

	start := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	onlineUsedAt := start.Add(2 * time.Hour)
	cardUsedAt := start.Add(3 * time.Hour)
	adminGrantUsedAt := start.Add(4 * time.Hour)
	adminDeductUsedAt := start.Add(5 * time.Hour)
	entitlementUsedAt := start.Add(6 * time.Hour)
	olderUsedAt := start.Add(-24 * time.Hour)

	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "PAY-ONLINE-1",
		Type:   domain.RedeemTypeBalance,
		Value:  10,
		UsedBy: userA.ID,
		UsedAt: onlineUsedAt,
	})
	client.PaymentOrder.Create().
		SetUserID(userA.ID).
		SetUserEmail(userA.Email).
		SetUserName(userA.Username).
		SetAmount(10).
		SetPayAmount(72).
		SetFeeRate(0).
		SetRechargeCode("PAY-ONLINE-1").
		SetOutTradeNo("out-online-1").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-online-1").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(onlineUsedAt.Add(30 * time.Minute)).
		SetPaidAt(onlineUsedAt.Add(-time.Minute)).
		SetCompletedAt(onlineUsedAt).
		SetClientIP("127.0.0.1").
		SetSrcHost("test").
		SaveX(ctx)

	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "CARD-CODE-1",
		Type:   domain.RedeemTypeBalance,
		Value:  5,
		UsedBy: userB.ID,
		UsedAt: cardUsedAt,
	})
	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "ADM-GRANT-1",
		Type:   AdjustmentTypeAdminBalance,
		Value:  3,
		UsedBy: userA.ID,
		UsedAt: adminGrantUsedAt,
	})
	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "ADM-DEDUCT-1",
		Type:   AdjustmentTypeAdminBalance,
		Value:  -2,
		UsedBy: userB.ID,
		UsedAt: adminDeductUsedAt,
		Notes:  "refund correction",
	})
	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:         "SUB-1",
		Type:         domain.RedeemTypeSubscription,
		Value:        30,
		UsedBy:       userB.ID,
		UsedAt:       entitlementUsedAt,
		GroupID:      &group.ID,
		ValidityDays: 30,
	})
	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "PAY-ORPHAN-1",
		Type:   domain.RedeemTypeBalance,
		Value:  7,
		UsedBy: userA.ID,
		UsedAt: start.Add(7 * time.Hour),
	})
	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "OLD-CARD-1",
		Type:   domain.RedeemTypeBalance,
		Value:  100,
		UsedBy: userA.ID,
		UsedAt: olderUsedAt,
	})

	svc := &PaymentService{entClient: client}
	summary, err := svc.GetFinanceLedgerSummary(ctx, FinanceLedgerQuery{StartTime: start, EndTime: end, Timezone: "UTC"})
	require.NoError(t, err)

	require.Equal(t, 25.0, summary.TotalAddedAmount)
	require.Equal(t, 22.0, summary.UserRechargeAmount)
	require.Equal(t, 10.0, summary.OnlinePaymentAmount)
	require.Equal(t, 12.0, summary.RedeemCodeAmount)
	require.Equal(t, 3.0, summary.AdminGrantedAmount)
	require.Equal(t, -2.0, summary.AdminDeductedAmount)
	require.Equal(t, 2, summary.UniqueUsers)
	require.Equal(t, 125.0, summary.CumulativeAddedAmount)
	require.Len(t, summary.SourceDistribution, 5)
	require.Equal(t, 10.0, findFinanceLedgerSource(summary.SourceDistribution, FinanceLedgerSourceOnlinePayment).Amount)
	require.Equal(t, 12.0, findFinanceLedgerSource(summary.SourceDistribution, FinanceLedgerSourceRedeemCode).Amount)
	require.Equal(t, 3.0, findFinanceLedgerSource(summary.SourceDistribution, FinanceLedgerSourceAdminGrant).Amount)
	require.Equal(t, -2.0, findFinanceLedgerSource(summary.SourceDistribution, FinanceLedgerSourceAdminDeduct).Amount)
	require.Equal(t, 30.0, findFinanceLedgerSource(summary.SourceDistribution, FinanceLedgerSourceEntitlement).Amount)
	require.Equal(t, 22.0, findFinanceLedgerType(summary.TypeDistribution, domain.RedeemTypeBalance).Amount)
	require.Equal(t, 1, summary.Anomalies.OrphanPaymentRedeemCodes)
	require.Equal(t, 1, summary.Anomalies.AdminAdjustmentsWithoutNotes)
	require.Len(t, summary.TopUsers, 2)
	require.Equal(t, userA.ID, summary.TopUsers[0].UserID)
	require.Equal(t, 20.0, summary.TopUsers[0].Amount)
}

func TestFinanceLedgerRecordsFilterBySourceSearchAndPaymentType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newFinanceLedgerTestClient(t)
	userA := createFinanceLedgerUser(t, ctx, client, "alice@example.com")
	userB := createFinanceLedgerUser(t, ctx, client, "bob@example.com")
	start := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "PAY-ALIPAY-1",
		Type:   domain.RedeemTypeBalance,
		Value:  10,
		UsedBy: userA.ID,
		UsedAt: start.Add(1 * time.Hour),
	})
	client.PaymentOrder.Create().
		SetUserID(userA.ID).
		SetUserEmail(userA.Email).
		SetUserName(userA.Username).
		SetAmount(10).
		SetPayAmount(72).
		SetFeeRate(0).
		SetRechargeCode("PAY-ALIPAY-1").
		SetOutTradeNo("out-alipay-1").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-alipay-1").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(start.Add(2 * time.Hour)).
		SetPaidAt(start.Add(50 * time.Minute)).
		SetCompletedAt(start.Add(1 * time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("test").
		SaveX(ctx)
	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "CARD-BOB-1",
		Type:   domain.RedeemTypeBalance,
		Value:  4,
		UsedBy: userB.ID,
		UsedAt: start.Add(2 * time.Hour),
	})
	createFinanceLedgerRedeemCode(t, ctx, client, financeLedgerRedeemFixture{
		Code:   "ADM-ALICE-1",
		Type:   AdjustmentTypeAdminBalance,
		Value:  6,
		UsedBy: userA.ID,
		UsedAt: start.Add(3 * time.Hour),
		Notes:  "gift for alice",
	})

	svc := &PaymentService{entClient: client}

	adminRecords, total, err := svc.ListFinanceLedgerRecords(ctx, FinanceLedgerQuery{
		StartTime: start,
		EndTime:   end,
		Source:    FinanceLedgerSourceAdminGrant,
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "ADM-ALICE-1", adminRecords[0].Code)
	require.Equal(t, FinanceLedgerSourceAdminGrant, adminRecords[0].Source)

	onlineRecords, total, err := svc.ListFinanceLedgerRecords(ctx, FinanceLedgerQuery{
		StartTime:   start,
		EndTime:     end,
		Source:      FinanceLedgerSourceOnlinePayment,
		PaymentType: payment.TypeAlipay,
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "PAY-ALIPAY-1", onlineRecords[0].Code)
	require.Equal(t, "out-alipay-1", onlineRecords[0].OutTradeNo)

	searchRecords, total, err := svc.ListFinanceLedgerRecords(ctx, FinanceLedgerQuery{
		StartTime: start,
		EndTime:   end,
		Search:    "bob@example.com",
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "CARD-BOB-1", searchRecords[0].Code)
}

type financeLedgerRedeemFixture struct {
	Code         string
	Type         string
	Value        float64
	UsedBy       int64
	UsedAt       time.Time
	Notes        string
	GroupID      *int64
	ValidityDays int
}

func newFinanceLedgerTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	dbName := "file:" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared&_fk=1"
	db, err := sql.Open("sqlite", dbName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func createFinanceLedgerUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) *dbent.User {
	t.Helper()
	return client.User.Create().
		SetEmail(email).
		SetUsername(email[:5]).
		SetPasswordHash("hash").
		SetRole(domain.RoleUser).
		SetStatus(domain.StatusActive).
		SetBalance(0).
		SaveX(ctx)
}

func createFinanceLedgerRedeemCode(t *testing.T, ctx context.Context, client *dbent.Client, f financeLedgerRedeemFixture) *dbent.RedeemCode {
	t.Helper()
	validityDays := f.ValidityDays
	if validityDays == 0 {
		validityDays = 30
	}
	create := client.RedeemCode.Create().
		SetCode(f.Code).
		SetType(f.Type).
		SetValue(f.Value).
		SetStatus(domain.StatusUsed).
		SetUsedBy(f.UsedBy).
		SetUsedAt(f.UsedAt).
		SetValidityDays(validityDays)
	if f.Notes != "" {
		create.SetNotes(f.Notes)
	}
	if f.GroupID != nil {
		create.SetGroupID(*f.GroupID)
	}
	return create.SaveX(ctx)
}

func findFinanceLedgerSource(items []FinanceLedgerDistributionItem, source string) FinanceLedgerDistributionItem {
	for _, item := range items {
		if item.Key == source {
			return item
		}
	}
	return FinanceLedgerDistributionItem{}
}

func findFinanceLedgerType(items []FinanceLedgerDistributionItem, typ string) FinanceLedgerDistributionItem {
	for _, item := range items {
		if item.Key == typ {
			return item
		}
	}
	return FinanceLedgerDistributionItem{}
}
