import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(__dirname, '../FinanceLedgerView.vue')
const apiPath = resolve(__dirname, '../../../api/admin/financeLedger.ts')
const adminApiIndexPath = resolve(__dirname, '../../../api/admin/index.ts')
const routerPath = resolve(__dirname, '../../../router/index.ts')
const sidebarPath = resolve(__dirname, '../../../components/layout/AppSidebar.vue')
const zhPath = resolve(__dirname, '../../../i18n/locales/zh.ts')
const enPath = resolve(__dirname, '../../../i18n/locales/en.ts')

describe('admin finance ledger dashboard exposure', () => {
  it('registers the admin route, sidebar item, and admin API barrel export', () => {
    expect(existsSync(viewPath)).toBe(true)
    expect(existsSync(apiPath)).toBe(true)

    const routerSource = readFileSync(routerPath, 'utf8')
    const sidebarSource = readFileSync(sidebarPath, 'utf8')
    const adminApiIndexSource = readFileSync(adminApiIndexPath, 'utf8')

    expect(routerSource).toContain("path: '/admin/finance/ledger'")
    expect(routerSource).toContain("component: () => import('@/views/admin/FinanceLedgerView.vue')")
    expect(sidebarSource).toContain("path: '/admin/finance/ledger'")
    expect(sidebarSource).toContain("t('nav.financeLedger')")
    expect(adminApiIndexSource).toContain('financeLedgerAPI')
  })

  it('uses dedicated finance ledger endpoints and supports CSV export', () => {
    expect(existsSync(apiPath)).toBe(true)

    const apiSource = readFileSync(apiPath, 'utf8')

    expect(apiSource).toContain('/admin/finance/ledger/summary')
    expect(apiSource).toContain('/admin/finance/ledger/records')
    expect(apiSource).toContain('/admin/finance/ledger/export')
    expect(apiSource).toContain("responseType: 'blob'")
  })

  it('renders summary cards, analysis panels, filters, records, and export controls', () => {
    expect(existsSync(viewPath)).toBe(true)

    const viewSource = readFileSync(viewPath, 'utf8')

    expect(viewSource).toContain('summaryCards')
    expect(viewSource).toContain('sourceDistribution')
    expect(viewSource).toContain('typeDistribution')
    expect(viewSource).toContain('topUsers')
    expect(viewSource).toContain('anomaly_only')
    expect(viewSource).toContain('exportCsv')
  })

  it('adds Chinese and English labels for the dashboard', () => {
    const zhSource = readFileSync(zhPath, 'utf8')
    const enSource = readFileSync(enPath, 'utf8')

    expect(zhSource).toContain("financeLedger: '财务流水'")
    expect(enSource).toContain("financeLedger: 'Finance Ledger'")
    expect(zhSource).toContain('financeLedger: {')
    expect(enSource).toContain('financeLedger: {')
  })
})
