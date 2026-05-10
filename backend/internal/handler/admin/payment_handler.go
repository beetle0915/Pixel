package admin

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles admin payment management.
type PaymentHandler struct {
	paymentService *service.PaymentService
	configService  *service.PaymentConfigService
}

// NewPaymentHandler creates a new admin PaymentHandler.
func NewPaymentHandler(paymentService *service.PaymentService, configService *service.PaymentConfigService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		configService:  configService,
	}
}

// --- Dashboard ---

// GetDashboard returns payment dashboard statistics.
// GET /api/v1/admin/payment/dashboard
func (h *PaymentHandler) GetDashboard(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}
	stats, err := h.paymentService.GetDashboardStats(c.Request.Context(), days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// --- Finance Ledger ---

// GetFinanceLedgerSummary returns redeem-code based finance ledger analytics.
// GET /api/v1/admin/finance/ledger/summary
func (h *PaymentHandler) GetFinanceLedgerSummary(c *gin.Context) {
	q, ok := parseFinanceLedgerQuery(c)
	if !ok {
		return
	}
	summary, err := h.paymentService.GetFinanceLedgerSummary(c.Request.Context(), q)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

// ListFinanceLedgerRecords returns paginated redeem-code finance ledger rows.
// GET /api/v1/admin/finance/ledger/records
func (h *PaymentHandler) ListFinanceLedgerRecords(c *gin.Context) {
	q, ok := parseFinanceLedgerQuery(c)
	if !ok {
		return
	}
	records, total, err := h.paymentService.ListFinanceLedgerRecords(c.Request.Context(), q)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, records, total, q.Page, q.PageSize)
}

// ExportFinanceLedgerRecords exports filtered redeem-code finance ledger rows as CSV.
// GET /api/v1/admin/finance/ledger/export
func (h *PaymentHandler) ExportFinanceLedgerRecords(c *gin.Context) {
	q, ok := parseFinanceLedgerQuery(c)
	if !ok {
		return
	}
	records, err := h.paymentService.ExportFinanceLedgerRecords(c.Request.Context(), q)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=finance_ledger.csv")
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"used_at", "source", "type", "amount", "user_id", "email", "username", "code", "notes", "out_trade_no", "payment_type", "pay_amount", "payment_status", "anomalies"})
	for _, record := range records {
		usedAt := ""
		if record.UsedAt != nil {
			usedAt = record.UsedAt.Format(time.RFC3339)
		}
		_ = w.Write([]string{
			usedAt,
			record.Source,
			record.Type,
			fmt.Sprintf("%.2f", record.Value),
			strconv.FormatInt(record.UserID, 10),
			record.UserEmail,
			record.Username,
			record.Code,
			record.Notes,
			record.OutTradeNo,
			record.PaymentType,
			fmt.Sprintf("%.2f", record.PayAmount),
			record.PaymentStatus,
			strings.Join(record.Anomalies, "|"),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		response.InternalError(c, "failed to export finance ledger")
		return
	}
	c.Status(http.StatusOK)
}

func parseFinanceLedgerQuery(c *gin.Context) (service.FinanceLedgerQuery, bool) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > 200 {
		pageSize = 200
	}
	timezone := strings.TrimSpace(c.DefaultQuery("timezone", "Asia/Shanghai"))
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		response.BadRequest(c, "Invalid timezone")
		return service.FinanceLedgerQuery{}, false
	}
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)
	if raw := strings.TrimSpace(c.Query("start_at")); raw != "" {
		parsed, err := parseFinanceLedgerTime(raw, loc, false)
		if err != nil {
			response.BadRequest(c, "Invalid start_at")
			return service.FinanceLedgerQuery{}, false
		}
		start = parsed
	}
	if raw := strings.TrimSpace(c.Query("end_at")); raw != "" {
		parsed, err := parseFinanceLedgerTime(raw, loc, true)
		if err != nil {
			response.BadRequest(c, "Invalid end_at")
			return service.FinanceLedgerQuery{}, false
		}
		end = parsed
	}
	if !end.After(start) {
		response.BadRequest(c, "end_at must be after start_at")
		return service.FinanceLedgerQuery{}, false
	}
	q := service.FinanceLedgerQuery{
		StartTime:   start,
		EndTime:     end,
		Timezone:    timezone,
		Type:        strings.TrimSpace(c.Query("type")),
		Source:      strings.TrimSpace(c.Query("source")),
		Search:      strings.TrimSpace(c.Query("search")),
		PaymentType: strings.TrimSpace(c.Query("payment_type")),
		AnomalyOnly: c.Query("anomaly_only") == "true",
		Page:        page,
		PageSize:    pageSize,
	}
	if raw := strings.TrimSpace(c.Query("min_amount")); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			response.BadRequest(c, "Invalid min_amount")
			return service.FinanceLedgerQuery{}, false
		}
		q.MinAmount = &value
	}
	if raw := strings.TrimSpace(c.Query("max_amount")); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			response.BadRequest(c, "Invalid max_amount")
			return service.FinanceLedgerQuery{}, false
		}
		q.MaxAmount = &value
	}
	return q, true
}

func parseFinanceLedgerTime(raw string, loc *time.Location, endOfDate bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDate {
		t = t.AddDate(0, 0, 1)
	}
	return t, nil
}

// --- Orders ---

// ListOrders returns a paginated list of all payment orders.
// GET /api/v1/admin/payment/orders
func (h *PaymentHandler) ListOrders(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var userID int64
	if uid := c.Query("user_id"); uid != "" {
		if v, err := strconv.ParseInt(uid, 10, 64); err == nil {
			userID = v
		}
	}
	orders, total, err := h.paymentService.AdminListOrders(c.Request.Context(), userID, service.OrderListParams{
		Page:        page,
		PageSize:    pageSize,
		Status:      c.Query("status"),
		OrderType:   c.Query("order_type"),
		PaymentType: c.Query("payment_type"),
		Keyword:     c.Query("keyword"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, sanitizeAdminPaymentOrdersForResponse(orders), int64(total), page, pageSize)
}

// GetOrderDetail returns detailed information about a single order.
// GET /api/v1/admin/payment/orders/:id
func (h *PaymentHandler) GetOrderDetail(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	order, err := h.paymentService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	auditLogs, _ := h.paymentService.GetOrderAuditLogs(c.Request.Context(), orderID)
	response.Success(c, gin.H{"order": sanitizeAdminPaymentOrderForResponse(order), "auditLogs": auditLogs})
}

// CancelOrder cancels a pending order (admin).
// POST /api/v1/admin/payment/orders/:id/cancel
func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	msg, err := h.paymentService.AdminCancelOrder(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": msg})
}

// RetryFulfillment retries fulfillment for a paid order.
// POST /api/v1/admin/payment/orders/:id/retry
func (h *PaymentHandler) RetryFulfillment(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.paymentService.RetryFulfillment(c.Request.Context(), orderID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "fulfillment retried"})
}

func sanitizeAdminPaymentOrdersForResponse(orders []*dbent.PaymentOrder) []*dbent.PaymentOrder {
	if len(orders) == 0 {
		return orders
	}
	out := make([]*dbent.PaymentOrder, 0, len(orders))
	for _, order := range orders {
		out = append(out, sanitizeAdminPaymentOrderForResponse(order))
	}
	return out
}

func sanitizeAdminPaymentOrderForResponse(order *dbent.PaymentOrder) *dbent.PaymentOrder {
	if order == nil {
		return nil
	}
	cloned := *order
	cloned.ProviderSnapshot = nil
	return &cloned
}

// AdminProcessRefundRequest is the request body for admin refund processing.
type AdminProcessRefundRequest struct {
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	Force         bool    `json:"force"`
	DeductBalance bool    `json:"deduct_balance"`
}

// ProcessRefund processes a refund for an order (admin).
// POST /api/v1/admin/payment/orders/:id/refund
func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	orderID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req AdminProcessRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	plan, earlyResult, err := h.paymentService.PrepareRefund(c.Request.Context(), orderID, req.Amount, req.Reason, req.Force, req.DeductBalance)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if earlyResult != nil {
		response.Success(c, earlyResult)
		return
	}

	result, err := h.paymentService.ExecuteRefund(c.Request.Context(), plan)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// --- Subscription Plans ---

// ListPlans returns all subscription plans.
// GET /api/v1/admin/payment/plans
func (h *PaymentHandler) ListPlans(c *gin.Context) {
	plans, err := h.configService.ListPlans(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, plans)
}

// CreatePlan creates a new subscription plan.
// POST /api/v1/admin/payment/plans
func (h *PaymentHandler) CreatePlan(c *gin.Context) {
	var req service.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.CreatePlan(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, plan)
}

// UpdatePlan updates an existing subscription plan.
// PUT /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) UpdatePlan(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.configService.UpdatePlan(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, plan)
}

// DeletePlan deletes a subscription plan.
// DELETE /api/v1/admin/payment/plans/:id
func (h *PaymentHandler) DeletePlan(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeletePlan(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// --- Provider Instances ---

// ListProviders returns all payment provider instances.
// GET /api/v1/admin/payment/providers
func (h *PaymentHandler) ListProviders(c *gin.Context) {
	providers, err := h.configService.ListProviderInstancesWithConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, providers)
}

// CreateProvider creates a new payment provider instance.
// POST /api/v1/admin/payment/providers
func (h *PaymentHandler) CreateProvider(c *gin.Context) {
	var req service.CreateProviderInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.CreateProviderInstance(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Created(c, inst)
}

// UpdateProvider updates an existing payment provider instance.
// PUT /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) UpdateProvider(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateProviderInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	inst, err := h.configService.UpdateProviderInstance(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Success(c, inst)
}

// DeleteProvider deletes a payment provider instance.
// DELETE /api/v1/admin/payment/providers/:id
func (h *PaymentHandler) DeleteProvider(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.configService.DeleteProviderInstance(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.paymentService.RefreshProviders(c.Request.Context())
	response.Success(c, gin.H{"message": "deleted"})
}

// parseIDParam parses an int64 path parameter.
// Returns the parsed ID and true on success; on failure it writes a BadRequest response and returns false.
func parseIDParam(c *gin.Context, paramName string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(paramName), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid "+paramName)
		return 0, false
	}
	return id, true
}

// --- Config ---

// GetConfig returns the payment configuration (admin view).
// GET /api/v1/admin/payment/config
func (h *PaymentHandler) GetConfig(c *gin.Context) {
	cfg, err := h.configService.GetPaymentConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateConfig updates the payment configuration.
// PUT /api/v1/admin/payment/config
func (h *PaymentHandler) UpdateConfig(c *gin.Context) {
	var req service.UpdatePaymentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.configService.UpdatePaymentConfig(c.Request.Context(), req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "updated"})
}
