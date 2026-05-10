package service

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/redeemcode"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/domain"
)

const (
	FinanceLedgerSourceOnlinePayment = "online_payment"
	FinanceLedgerSourceRedeemCode    = "redeem_code"
	FinanceLedgerSourceAdminGrant    = "admin_grant"
	FinanceLedgerSourceAdminDeduct   = "admin_deduct"
	FinanceLedgerSourceEntitlement   = "entitlement"
)

type FinanceLedgerQuery struct {
	StartTime   time.Time
	EndTime     time.Time
	Timezone    string
	Type        string
	Source      string
	Search      string
	PaymentType string
	MinAmount   *float64
	MaxAmount   *float64
	AnomalyOnly bool
	Page        int
	PageSize    int
}

type FinanceLedgerSummary struct {
	TotalAddedAmount        float64                         `json:"total_added_amount"`
	UserRechargeAmount      float64                         `json:"user_recharge_amount"`
	OnlinePaymentAmount     float64                         `json:"online_payment_amount"`
	RedeemCodeAmount        float64                         `json:"redeem_code_amount"`
	AdminGrantedAmount      float64                         `json:"admin_granted_amount"`
	AdminDeductedAmount     float64                         `json:"admin_deducted_amount"`
	CumulativeAddedAmount   float64                         `json:"cumulative_added_amount"`
	UniqueUsers             int                             `json:"unique_users"`
	RecordCount             int                             `json:"record_count"`
	SourceDistribution      []FinanceLedgerDistributionItem `json:"source_distribution"`
	TypeDistribution        []FinanceLedgerDistributionItem `json:"type_distribution"`
	PaymentTypeDistribution []FinanceLedgerDistributionItem `json:"payment_type_distribution"`
	DailySeries             []FinanceLedgerSeriesPoint      `json:"daily_series"`
	TopUsers                []FinanceLedgerTopUser          `json:"top_users"`
	Anomalies               FinanceLedgerAnomalySummary     `json:"anomalies"`
}

type FinanceLedgerDistributionItem struct {
	Key    string  `json:"key"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type FinanceLedgerSeriesPoint struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type FinanceLedgerTopUser struct {
	UserID   int64   `json:"user_id"`
	Email    string  `json:"email"`
	Username string  `json:"username"`
	Amount   float64 `json:"amount"`
	Count    int     `json:"count"`
}

type FinanceLedgerAnomalySummary struct {
	OrphanPaymentRedeemCodes     int `json:"orphan_payment_redeem_codes"`
	PaymentOrdersWithoutRedeem   int `json:"payment_orders_without_redeem"`
	MissingUsers                 int `json:"missing_users"`
	AdminAdjustmentsWithoutNotes int `json:"admin_adjustments_without_notes"`
	NegativeAdjustments          int `json:"negative_adjustments"`
}

type FinanceLedgerRecord struct {
	ID             int64      `json:"id"`
	Code           string     `json:"code"`
	Type           string     `json:"type"`
	Source         string     `json:"source"`
	Value          float64    `json:"value"`
	Status         string     `json:"status"`
	UsedAt         *time.Time `json:"used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	Notes          string     `json:"notes"`
	UserID         int64      `json:"user_id,omitempty"`
	UserEmail      string     `json:"user_email"`
	Username       string     `json:"username"`
	GroupID        *int64     `json:"group_id,omitempty"`
	GroupName      string     `json:"group_name,omitempty"`
	ValidityDays   int        `json:"validity_days"`
	PaymentOrderID int64      `json:"payment_order_id,omitempty"`
	OutTradeNo     string     `json:"out_trade_no"`
	PaymentType    string     `json:"payment_type"`
	PayAmount      float64    `json:"pay_amount"`
	PaymentStatus  string     `json:"payment_status"`
	Anomalies      []string   `json:"anomalies,omitempty"`
}

func (s *PaymentService) GetFinanceLedgerSummary(ctx context.Context, q FinanceLedgerQuery) (*FinanceLedgerSummary, error) {
	records, err := s.loadFinanceLedgerRecords(ctx, q)
	if err != nil {
		return nil, err
	}
	cumulative, err := s.financeLedgerCumulativeAdded(ctx)
	if err != nil {
		return nil, err
	}
	out := buildFinanceLedgerSummary(records, q)
	out.CumulativeAddedAmount = roundFinanceAmount(cumulative)
	return out, nil
}

func (s *PaymentService) ListFinanceLedgerRecords(ctx context.Context, q FinanceLedgerQuery) ([]FinanceLedgerRecord, int64, error) {
	records, err := s.loadFinanceLedgerRecords(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(records))
	page, pageSize := normalizeFinanceLedgerPagination(q.Page, q.PageSize)
	start := (page - 1) * pageSize
	if start >= len(records) {
		return []FinanceLedgerRecord{}, total, nil
	}
	end := start + pageSize
	if end > len(records) {
		end = len(records)
	}
	return records[start:end], total, nil
}

func (s *PaymentService) ExportFinanceLedgerRecords(ctx context.Context, q FinanceLedgerQuery) ([]FinanceLedgerRecord, error) {
	return s.loadFinanceLedgerRecords(ctx, q)
}

func (s *PaymentService) loadFinanceLedgerRecords(ctx context.Context, q FinanceLedgerQuery) ([]FinanceLedgerRecord, error) {
	query := s.entClient.RedeemCode.Query().
		Where(redeemcode.StatusEQ(domain.StatusUsed), redeemcode.UsedAtNotNil()).
		WithUser().
		WithGroup()
	if !q.StartTime.IsZero() {
		query = query.Where(redeemcode.UsedAtGTE(q.StartTime))
	}
	if !q.EndTime.IsZero() {
		query = query.Where(redeemcode.UsedAtLT(q.EndTime))
	}
	if q.Type != "" {
		query = query.Where(redeemcode.TypeEQ(q.Type))
	}
	if q.MinAmount != nil {
		query = query.Where(redeemcode.ValueGTE(*q.MinAmount))
	}
	if q.MaxAmount != nil {
		query = query.Where(redeemcode.ValueLTE(*q.MaxAmount))
	}
	if search := strings.TrimSpace(q.Search); search != "" {
		query = query.Where(redeemcode.Or(
			redeemcode.CodeContainsFold(search),
			redeemcode.NotesContainsFold(search),
			redeemcode.HasUserWith(
				dbuser.Or(
					dbuser.EmailContainsFold(search),
					dbuser.UsernameContainsFold(search),
				),
			),
		))
	}

	codes, err := query.Order(dbent.Desc(redeemcode.FieldUsedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	orders, err := s.financeLedgerOrdersByRechargeCode(ctx, codes)
	if err != nil {
		return nil, err
	}
	records := make([]FinanceLedgerRecord, 0, len(codes))
	for _, code := range codes {
		order := orders[code.Code]
		record := financeLedgerRecordFromEntities(code, order)
		if !financeLedgerRecordMatches(record, q) {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *PaymentService) financeLedgerOrdersByRechargeCode(ctx context.Context, codes []*dbent.RedeemCode) (map[string]*dbent.PaymentOrder, error) {
	rechargeCodes := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if code == nil || code.Code == "" {
			continue
		}
		if _, ok := seen[code.Code]; ok {
			continue
		}
		seen[code.Code] = struct{}{}
		rechargeCodes = append(rechargeCodes, code.Code)
	}
	if len(rechargeCodes) == 0 {
		return map[string]*dbent.PaymentOrder{}, nil
	}
	orders, err := s.entClient.PaymentOrder.Query().
		Where(paymentorder.RechargeCodeIn(rechargeCodes...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*dbent.PaymentOrder, len(orders))
	for _, order := range orders {
		if order != nil {
			out[order.RechargeCode] = order
		}
	}
	return out, nil
}

func (s *PaymentService) financeLedgerCumulativeAdded(ctx context.Context) (float64, error) {
	codes, err := s.entClient.RedeemCode.Query().
		Where(
			redeemcode.StatusEQ(domain.StatusUsed),
			redeemcode.ValueGT(0),
			redeemcode.TypeIn(domain.RedeemTypeBalance, AdjustmentTypeAdminBalance),
		).
		All(ctx)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, code := range codes {
		total += code.Value
	}
	return total, nil
}

func financeLedgerRecordFromEntities(code *dbent.RedeemCode, order *dbent.PaymentOrder) FinanceLedgerRecord {
	record := FinanceLedgerRecord{
		ID:           code.ID,
		Code:         code.Code,
		Type:         code.Type,
		Source:       financeLedgerSource(code, order),
		Value:        roundFinanceAmount(code.Value),
		Status:       code.Status,
		UsedAt:       code.UsedAt,
		CreatedAt:    code.CreatedAt,
		Notes:        financeLedgerStringValue(code.Notes),
		GroupID:      code.GroupID,
		ValidityDays: code.ValidityDays,
	}
	if code.Edges.User != nil {
		record.UserID = code.Edges.User.ID
		record.UserEmail = code.Edges.User.Email
		record.Username = code.Edges.User.Username
	} else if code.UsedBy != nil {
		record.UserID = *code.UsedBy
	}
	if code.Edges.Group != nil {
		record.GroupName = code.Edges.Group.Name
	}
	if order != nil {
		record.PaymentOrderID = order.ID
		record.OutTradeNo = order.OutTradeNo
		record.PaymentType = order.PaymentType
		record.PayAmount = roundFinanceAmount(order.PayAmount)
		record.PaymentStatus = order.Status
	}
	record.Anomalies = financeLedgerRecordAnomalies(record)
	return record
}

func financeLedgerSource(code *dbent.RedeemCode, order *dbent.PaymentOrder) string {
	switch code.Type {
	case domain.RedeemTypeBalance:
		if order != nil {
			return FinanceLedgerSourceOnlinePayment
		}
		return FinanceLedgerSourceRedeemCode
	case AdjustmentTypeAdminBalance:
		if code.Value < 0 {
			return FinanceLedgerSourceAdminDeduct
		}
		return FinanceLedgerSourceAdminGrant
	default:
		return FinanceLedgerSourceEntitlement
	}
}

func financeLedgerRecordAnomalies(record FinanceLedgerRecord) []string {
	var out []string
	if record.UserID == 0 {
		out = append(out, "missing_user")
	}
	if strings.HasPrefix(record.Code, "PAY-") && record.PaymentOrderID == 0 {
		out = append(out, "orphan_payment_redeem_code")
	}
	if record.Type == AdjustmentTypeAdminBalance && strings.TrimSpace(record.Notes) == "" {
		out = append(out, "admin_adjustment_without_notes")
	}
	if record.Type == AdjustmentTypeAdminBalance && record.Value < 0 {
		out = append(out, "negative_adjustment")
	}
	return out
}

func financeLedgerRecordMatches(record FinanceLedgerRecord, q FinanceLedgerQuery) bool {
	if q.Source != "" && record.Source != q.Source {
		return false
	}
	if q.PaymentType != "" && record.PaymentType != q.PaymentType {
		return false
	}
	if q.AnomalyOnly && len(record.Anomalies) == 0 {
		return false
	}
	return true
}

func buildFinanceLedgerSummary(records []FinanceLedgerRecord, q FinanceLedgerQuery) *FinanceLedgerSummary {
	out := &FinanceLedgerSummary{RecordCount: len(records)}
	uniqueUsers := make(map[int64]struct{})
	sources := make(map[string]*FinanceLedgerDistributionItem)
	types := make(map[string]*FinanceLedgerDistributionItem)
	paymentTypes := make(map[string]*FinanceLedgerDistributionItem)
	topUsers := make(map[int64]*FinanceLedgerTopUser)
	series := make(map[string]*FinanceLedgerSeriesPoint)
	loc := financeLedgerLocation(q.Timezone)

	for _, record := range records {
		if record.UserID > 0 {
			uniqueUsers[record.UserID] = struct{}{}
		}
		if isFinanceLedgerPositiveBalance(record) {
			out.TotalAddedAmount += record.Value
		}
		if record.Type == domain.RedeemTypeBalance && record.Value > 0 {
			out.UserRechargeAmount += record.Value
		}
		switch record.Source {
		case FinanceLedgerSourceOnlinePayment:
			out.OnlinePaymentAmount += record.Value
		case FinanceLedgerSourceRedeemCode:
			if record.Type == domain.RedeemTypeBalance {
				out.RedeemCodeAmount += record.Value
			}
		case FinanceLedgerSourceAdminGrant:
			out.AdminGrantedAmount += record.Value
		case FinanceLedgerSourceAdminDeduct:
			out.AdminDeductedAmount += record.Value
		}
		addFinanceLedgerDistribution(sources, record.Source, record.Value)
		addFinanceLedgerDistribution(types, record.Type, record.Value)
		if record.PaymentType != "" {
			addFinanceLedgerDistribution(paymentTypes, record.PaymentType, record.PayAmount)
		}
		if isFinanceLedgerPositiveBalance(record) && record.UserID > 0 {
			user := topUsers[record.UserID]
			if user == nil {
				user = &FinanceLedgerTopUser{UserID: record.UserID, Email: record.UserEmail, Username: record.Username}
				topUsers[record.UserID] = user
			}
			user.Amount += record.Value
			user.Count++
		}
		if record.UsedAt != nil {
			key := record.UsedAt.In(loc).Format("2006-01-02")
			point := series[key]
			if point == nil {
				point = &FinanceLedgerSeriesPoint{Date: key}
				series[key] = point
			}
			if isFinanceLedgerPositiveBalance(record) {
				point.Amount += record.Value
			}
			point.Count++
		}
		accumulateFinanceLedgerAnomalies(&out.Anomalies, record)
	}

	out.TotalAddedAmount = roundFinanceAmount(out.TotalAddedAmount)
	out.UserRechargeAmount = roundFinanceAmount(out.UserRechargeAmount)
	out.OnlinePaymentAmount = roundFinanceAmount(out.OnlinePaymentAmount)
	out.RedeemCodeAmount = roundFinanceAmount(out.RedeemCodeAmount)
	out.AdminGrantedAmount = roundFinanceAmount(out.AdminGrantedAmount)
	out.AdminDeductedAmount = roundFinanceAmount(out.AdminDeductedAmount)
	out.UniqueUsers = len(uniqueUsers)
	out.SourceDistribution = sortedFinanceLedgerDistribution(sources)
	out.TypeDistribution = sortedFinanceLedgerDistribution(types)
	out.PaymentTypeDistribution = sortedFinanceLedgerDistribution(paymentTypes)
	out.DailySeries = sortedFinanceLedgerSeries(series)
	out.TopUsers = sortedFinanceLedgerTopUsers(topUsers)
	return out
}

func isFinanceLedgerPositiveBalance(record FinanceLedgerRecord) bool {
	return record.Value > 0 && (record.Type == domain.RedeemTypeBalance || record.Type == AdjustmentTypeAdminBalance)
}

func addFinanceLedgerDistribution(items map[string]*FinanceLedgerDistributionItem, key string, amount float64) {
	if key == "" {
		key = "unknown"
	}
	item := items[key]
	if item == nil {
		item = &FinanceLedgerDistributionItem{Key: key}
		items[key] = item
	}
	item.Amount += amount
	item.Count++
}

func sortedFinanceLedgerDistribution(items map[string]*FinanceLedgerDistributionItem) []FinanceLedgerDistributionItem {
	out := make([]FinanceLedgerDistributionItem, 0, len(items))
	for _, item := range items {
		item.Amount = roundFinanceAmount(item.Amount)
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Amount == out[j].Amount {
			return out[i].Key < out[j].Key
		}
		return out[i].Amount > out[j].Amount
	})
	return out
}

func sortedFinanceLedgerSeries(items map[string]*FinanceLedgerSeriesPoint) []FinanceLedgerSeriesPoint {
	out := make([]FinanceLedgerSeriesPoint, 0, len(items))
	for _, item := range items {
		item.Amount = roundFinanceAmount(item.Amount)
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

func sortedFinanceLedgerTopUsers(items map[int64]*FinanceLedgerTopUser) []FinanceLedgerTopUser {
	out := make([]FinanceLedgerTopUser, 0, len(items))
	for _, item := range items {
		item.Amount = roundFinanceAmount(item.Amount)
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Amount == out[j].Amount {
			return out[i].UserID < out[j].UserID
		}
		return out[i].Amount > out[j].Amount
	})
	if len(out) > topUsersLimit {
		out = out[:topUsersLimit]
	}
	return out
}

func accumulateFinanceLedgerAnomalies(out *FinanceLedgerAnomalySummary, record FinanceLedgerRecord) {
	for _, anomaly := range record.Anomalies {
		switch anomaly {
		case "orphan_payment_redeem_code":
			out.OrphanPaymentRedeemCodes++
		case "missing_user":
			out.MissingUsers++
		case "admin_adjustment_without_notes":
			out.AdminAdjustmentsWithoutNotes++
		case "negative_adjustment":
			out.NegativeAdjustments++
		}
	}
}

func financeLedgerLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}

func normalizeFinanceLedgerPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func roundFinanceAmount(v float64) float64 {
	return math.Round(v*100) / 100
}

func financeLedgerStringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
