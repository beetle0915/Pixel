import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export type FinanceLedgerSource =
  | 'online_payment'
  | 'redeem_code'
  | 'admin_grant'
  | 'admin_deduct'
  | 'entitlement'

export interface FinanceLedgerQueryParams {
  start_at?: string
  end_at?: string
  timezone?: string
  type?: string
  source?: FinanceLedgerSource | ''
  search?: string
  payment_type?: string
  min_amount?: number
  max_amount?: number
  anomaly_only?: boolean
  page?: number
  page_size?: number
}

export interface FinanceLedgerDistributionItem {
  key: string
  amount: number
  count: number
}

export interface FinanceLedgerSeriesPoint {
  date: string
  amount: number
  count: number
}

export interface FinanceLedgerTopUser {
  user_id: number
  email: string
  username: string
  amount: number
  count: number
}

export interface FinanceLedgerAnomalySummary {
  orphan_payment_redeem_codes: number
  payment_orders_without_redeem: number
  missing_users: number
  admin_adjustments_without_notes: number
  negative_adjustments: number
}

export interface FinanceLedgerSummary {
  total_added_amount: number
  user_recharge_amount: number
  online_payment_amount: number
  redeem_code_amount: number
  admin_granted_amount: number
  admin_deducted_amount: number
  cumulative_added_amount: number
  unique_users: number
  record_count: number
  source_distribution: FinanceLedgerDistributionItem[]
  type_distribution: FinanceLedgerDistributionItem[]
  payment_type_distribution: FinanceLedgerDistributionItem[]
  daily_series: FinanceLedgerSeriesPoint[]
  top_users: FinanceLedgerTopUser[]
  anomalies: FinanceLedgerAnomalySummary
}

export interface FinanceLedgerRecord {
  id: number
  code: string
  type: string
  source: FinanceLedgerSource | string
  value: number
  status: string
  used_at?: string | null
  created_at: string
  notes: string
  user_id?: number
  user_email: string
  username: string
  group_id?: number
  group_name?: string
  validity_days: number
  payment_order_id?: number
  out_trade_no: string
  payment_type: string
  pay_amount: number
  payment_status: string
  anomalies?: string[]
}

export const financeLedgerAPI = {
  getSummary(params?: FinanceLedgerQueryParams) {
    return apiClient.get<FinanceLedgerSummary>('/admin/finance/ledger/summary', { params })
  },

  getRecords(params?: FinanceLedgerQueryParams) {
    return apiClient.get<BasePaginationResponse<FinanceLedgerRecord>>('/admin/finance/ledger/records', { params })
  },

  async exportCsv(params?: FinanceLedgerQueryParams): Promise<Blob> {
    const { data } = await apiClient.get<Blob>('/admin/finance/ledger/export', {
      params,
      responseType: 'blob',
    })
    return data
  },
}

export default financeLedgerAPI
