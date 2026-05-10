<template>
  <AppLayout>
    <div class="space-y-5">
      <section class="card p-5">
        <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div class="inline-flex max-w-full overflow-x-auto rounded-lg border border-gray-200 bg-white p-1 dark:border-dark-600 dark:bg-dark-800">
            <button
              v-for="preset in rangePresets"
              :key="preset.key"
              type="button"
              class="whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="activePreset === preset.key
                ? 'bg-gray-900 text-white shadow-sm dark:bg-white dark:text-gray-900'
                : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
              @click="setPreset(preset.key)"
            >
              {{ preset.label }}
            </button>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <button type="button" class="btn btn-secondary h-10" :disabled="loading" @click="loadData">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              <span>{{ t('admin.financeLedger.refresh') }}</span>
            </button>
            <button type="button" class="btn btn-primary h-10" :disabled="exporting" @click="exportCsv">
              <Icon name="download" size="sm" />
              <span>{{ exporting ? t('admin.financeLedger.exporting') : t('admin.financeLedger.exportCsv') }}</span>
            </button>
          </div>
        </div>

        <div class="mt-5 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
          <label class="block">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.filters.startDate') }}
            </span>
            <input v-model="filters.startDate" type="date" class="input" @change="activePreset = 'custom'" />
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.filters.endDate') }}
            </span>
            <input v-model="filters.endDate" type="date" class="input" @change="activePreset = 'custom'" />
          </label>
          <label class="block xl:col-span-2">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('common.search') }}
            </span>
            <input
              v-model.trim="filters.search"
              type="search"
              class="input"
              :placeholder="t('admin.financeLedger.filters.searchPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </label>
        </div>

        <div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
          <label class="block">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.filters.source') }}
            </span>
            <select v-model="filters.source" class="input">
              <option v-for="option in sourceOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.filters.type') }}
            </span>
            <select v-model="filters.type" class="input">
              <option v-for="option in typeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.filters.paymentType') }}
            </span>
            <select v-model="filters.paymentType" class="input">
              <option value="">{{ t('admin.financeLedger.filters.allPaymentTypes') }}</option>
              <option value="alipay">Alipay</option>
              <option value="wxpay">WeChat Pay</option>
              <option value="stripe">Stripe</option>
              <option value="card">Card</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.filters.minAmount') }}
            </span>
            <input v-model.trim="filters.minAmount" type="number" step="0.01" min="0" class="input" />
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.filters.maxAmount') }}
            </span>
            <input v-model.trim="filters.maxAmount" type="number" step="0.01" min="0" class="input" />
          </label>
          <div class="flex items-end gap-2">
            <label class="flex h-10 flex-1 items-center gap-2 rounded-lg border border-gray-200 px-3 text-sm text-gray-700 dark:border-dark-600 dark:text-gray-300">
              <input v-model="filters.anomalyOnly" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span>{{ t('admin.financeLedger.filters.anomalyOnly') }}</span>
            </label>
          </div>
        </div>

        <div class="mt-4 flex flex-wrap items-center justify-end gap-2">
          <button type="button" class="btn btn-secondary h-10" @click="resetFilters">{{ t('admin.financeLedger.filters.reset') }}</button>
          <button type="button" class="btn btn-primary h-10" @click="applyFilters">{{ t('admin.financeLedger.filters.apply') }}</button>
        </div>
      </section>

      <div v-if="loading && !summary" class="flex items-center justify-center py-16">
        <LoadingSpinner />
      </div>

      <template v-else>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div v-for="card in summaryCards" :key="card.key" class="card min-h-[118px] border-l-4 p-4" :class="card.borderClass">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</p>
                <p class="mt-2 break-words text-2xl font-semibold text-gray-900 dark:text-white">{{ card.value }}</p>
              </div>
              <span class="mt-1 h-2.5 w-2.5 flex-shrink-0 rounded-full" :class="card.dotClass"></span>
            </div>
            <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ card.meta }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-5 xl:grid-cols-3">
          <section class="card p-5">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.financeLedger.sections.sourceDistribution') }}</h3>
            <DistributionList :items="sourceDistribution" :label-for="sourceLabel" :format-amount="formatAmount" />
          </section>

          <section class="card p-5">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.financeLedger.sections.typeDistribution') }}</h3>
            <DistributionList :items="typeDistribution" :label-for="typeLabel" :format-amount="formatAmount" />
          </section>

          <section class="card p-5">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.financeLedger.sections.anomalies') }}</h3>
            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <div class="flex items-center justify-between gap-4 py-3">
                <span class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.financeLedger.anomalies.total') }}</span>
                <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatInteger(totalAnomalies) }}</span>
              </div>
              <div v-for="row in anomalyRows" :key="row.key" class="flex items-center justify-between gap-4 py-3">
                <span class="min-w-0 truncate text-sm text-gray-500 dark:text-gray-400">{{ row.label }}</span>
                <span class="text-sm font-medium" :class="row.value > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-gray-400'">
                  {{ formatInteger(row.value) }}
                </span>
              </div>
            </div>
          </section>
        </div>

        <div class="grid grid-cols-1 gap-5 xl:grid-cols-2">
          <section class="card p-5">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.financeLedger.sections.dailySeries') }}</h3>
            <div v-if="dailySeries.length" class="space-y-3">
              <div v-for="point in dailySeries" :key="point.date" class="grid grid-cols-[96px_minmax(0,1fr)_88px] items-center gap-3">
                <span class="text-sm text-gray-500 dark:text-gray-400">{{ point.date }}</span>
                <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full rounded-full bg-sky-500" :style="{ width: seriesWidth(point.amount) }"></div>
                </div>
                <span class="text-right text-sm font-medium text-gray-900 dark:text-white">{{ formatAmount(point.amount) }}</span>
              </div>
            </div>
            <div v-else class="flex h-36 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.noData') }}
            </div>
          </section>

          <section class="card p-5">
            <h3 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.financeLedger.sections.topUsers') }}</h3>
            <div v-if="topUsers.length" class="divide-y divide-gray-100 dark:divide-dark-700">
              <div v-for="user in topUsers" :key="user.user_id" class="flex items-center justify-between gap-4 py-3">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ user.email || user.username || `#${user.user_id}` }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.financeLedger.cards.metaRecords', { count: user.count }) }}</p>
                </div>
                <span class="text-right text-sm font-semibold text-gray-900 dark:text-white">{{ formatAmount(user.amount) }}</span>
              </div>
            </div>
            <div v-else class="flex h-36 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.noData') }}
            </div>
          </section>
        </div>

        <section class="card overflow-hidden">
          <div class="flex flex-col gap-2 border-b border-gray-100 p-5 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.financeLedger.sections.records') }}</h3>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.financeLedger.table.showing', { start: displayStart, end: displayEnd, total: pagination.total }) }}
            </span>
          </div>

          <div v-if="recordsLoading" class="flex items-center justify-center py-12">
            <LoadingSpinner />
          </div>

          <div v-else-if="records.length" class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th v-for="column in tableColumns" :key="column" class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                    {{ column }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="record in records" :key="record.id" class="hover:bg-gray-50 dark:hover:bg-dark-800">
                  <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(record.used_at) }}</td>
                  <td class="max-w-[220px] px-4 py-3">
                    <div class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ record.user_email || record.username || `#${record.user_id ?? '-'}` }}</div>
                    <div v-if="record.group_name" class="truncate text-xs text-gray-500 dark:text-gray-400">{{ record.group_name }}</div>
                  </td>
                  <td class="whitespace-nowrap px-4 py-3 text-sm font-semibold" :class="record.value < 0 ? 'text-rose-600 dark:text-rose-400' : 'text-gray-900 dark:text-white'">
                    {{ formatAmount(record.value) }}
                  </td>
                  <td class="whitespace-nowrap px-4 py-3">
                    <span class="rounded-full px-2 py-1 text-xs font-medium" :class="sourceBadgeClass(record.source)">
                      {{ sourceLabel(record.source) }}
                    </span>
                  </td>
                  <td class="max-w-[220px] px-4 py-3">
                    <div class="break-all font-mono text-xs text-gray-700 dark:text-gray-300">{{ record.code }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ typeLabel(record.type) }} · {{ record.status }}</div>
                  </td>
                  <td class="max-w-[220px] px-4 py-3">
                    <div v-if="record.payment_order_id" class="truncate text-sm text-gray-700 dark:text-gray-300">
                      {{ record.payment_type || '-' }} · {{ formatAmount(record.pay_amount) }}
                    </div>
                    <div v-else class="text-sm text-gray-400">{{ t('admin.financeLedger.table.noPayment') }}</div>
                    <div v-if="record.out_trade_no" class="truncate text-xs text-gray-500 dark:text-gray-400">{{ record.out_trade_no }}</div>
                  </td>
                  <td class="max-w-[260px] px-4 py-3 text-sm text-gray-600 dark:text-gray-300">
                    <span class="line-clamp-2">{{ record.notes || '-' }}</span>
                  </td>
                  <td class="max-w-[260px] px-4 py-3">
                    <div v-if="record.anomalies?.length" class="flex flex-wrap gap-1">
                      <span v-for="anomaly in record.anomalies" :key="`${record.id}-${anomaly}`" class="rounded-full bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                        {{ anomalyLabel(anomaly) }}
                      </span>
                    </div>
                    <span v-else class="text-sm text-gray-400">{{ t('admin.financeLedger.anomalies.none') }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="flex h-44 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.financeLedger.table.empty') }}
          </div>

          <div class="flex flex-col gap-3 border-t border-gray-100 p-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <span>{{ t('admin.financeLedger.table.page', { page: pagination.page, pages: Math.max(pagination.pages, 1) }) }}</span>
              <select v-model.number="pageSize" class="input h-9 w-24" @change="applyFilters">
                <option :value="20">20</option>
                <option :value="50">50</option>
                <option :value="100">100</option>
                <option :value="200">200</option>
              </select>
            </div>
            <div class="flex items-center gap-2">
              <button type="button" class="btn btn-secondary h-9" :disabled="currentPage <= 1 || recordsLoading" @click="changePage(currentPage - 1)">
                {{ t('admin.financeLedger.table.prev') }}
              </button>
              <button type="button" class="btn btn-secondary h-9" :disabled="currentPage >= Math.max(pagination.pages, 1) || recordsLoading" @click="changePage(currentPage + 1)">
                {{ t('admin.financeLedger.table.next') }}
              </button>
            </div>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { financeLedgerAPI } from '@/api/admin/financeLedger'
import type {
  FinanceLedgerDistributionItem,
  FinanceLedgerQueryParams,
  FinanceLedgerRecord,
  FinanceLedgerSource,
  FinanceLedgerSummary
} from '@/api/admin/financeLedger'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'

type RangePresetKey = 'today' | 'yesterday' | 'last7Days' | 'thisMonth' | 'custom'

interface LedgerFilters {
  startDate: string
  endDate: string
  source: '' | FinanceLedgerSource
  type: string
  paymentType: string
  search: string
  minAmount: string
  maxAmount: string
  anomalyOnly: boolean
}

const DistributionList = defineComponent({
  name: 'DistributionList',
  props: {
    items: {
      type: Array as () => FinanceLedgerDistributionItem[],
      required: true,
    },
    labelFor: {
      type: Function as unknown as () => (key: string) => string,
      required: true,
    },
    formatAmount: {
      type: Function as unknown as () => (value: number) => string,
      required: true,
    },
  },
  setup(props) {
    const maxAmount = computed(() => Math.max(...props.items.map((item) => Math.abs(item.amount)), 1))
    const barWidth = (amount: number) => `${Math.max(4, Math.min(100, (Math.abs(amount) / maxAmount.value) * 100))}%`

    return () => props.items.length
      ? h('div', { class: 'space-y-3' }, props.items.map((item) => h('div', { key: item.key }, [
          h('div', { class: 'mb-1 flex items-center justify-between gap-3' }, [
            h('span', { class: 'truncate text-sm text-gray-600 dark:text-gray-300' }, props.labelFor(item.key)),
            h('span', { class: 'whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white' }, props.formatAmount(item.amount)),
          ]),
          h('div', { class: 'h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700' }, [
            h('div', { class: 'h-full rounded-full bg-emerald-500', style: { width: barWidth(item.amount) } }),
          ]),
          h('div', { class: 'mt-1 text-xs text-gray-500 dark:text-gray-400' }, `${item.count}`),
        ])))
      : h('div', { class: 'flex h-36 items-center justify-center text-sm text-gray-500 dark:text-gray-400' }, '-')
  },
})

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const recordsLoading = ref(false)
const exporting = ref(false)
const summary = ref<FinanceLedgerSummary | null>(null)
const records = ref<FinanceLedgerRecord[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const activePreset = ref<RangePresetKey>('today')
const pagination = ref({
  total: 0,
  page: 1,
  page_size: 20,
  pages: 0,
})

const filters = ref<LedgerFilters>({
  startDate: '',
  endDate: '',
  source: '',
  type: '',
  paymentType: '',
  search: '',
  minAmount: '',
  maxAmount: '',
  anomalyOnly: false,
})

const rangePresets = computed(() => [
  { key: 'today' as const, label: t('admin.financeLedger.presets.today') },
  { key: 'yesterday' as const, label: t('admin.financeLedger.presets.yesterday') },
  { key: 'last7Days' as const, label: t('admin.financeLedger.presets.last7Days') },
  { key: 'thisMonth' as const, label: t('admin.financeLedger.presets.thisMonth') },
])

const sourceOptions = computed(() => [
  { value: '', label: t('admin.financeLedger.filters.allSources') },
  { value: 'online_payment', label: t('admin.financeLedger.sources.online_payment') },
  { value: 'redeem_code', label: t('admin.financeLedger.sources.redeem_code') },
  { value: 'admin_grant', label: t('admin.financeLedger.sources.admin_grant') },
  { value: 'admin_deduct', label: t('admin.financeLedger.sources.admin_deduct') },
  { value: 'entitlement', label: t('admin.financeLedger.sources.entitlement') },
] as Array<{ value: '' | FinanceLedgerSource, label: string }>)

const typeOptions = computed(() => [
  { value: '', label: t('admin.financeLedger.filters.allTypes') },
  { value: 'balance', label: t('admin.financeLedger.types.balance') },
  { value: 'admin_balance', label: t('admin.financeLedger.types.admin_balance') },
  { value: 'subscription', label: t('admin.financeLedger.types.subscription') },
])

const emptySummary: FinanceLedgerSummary = {
  total_added_amount: 0,
  user_recharge_amount: 0,
  online_payment_amount: 0,
  redeem_code_amount: 0,
  admin_granted_amount: 0,
  admin_deducted_amount: 0,
  cumulative_added_amount: 0,
  unique_users: 0,
  record_count: 0,
  source_distribution: [],
  type_distribution: [],
  payment_type_distribution: [],
  daily_series: [],
  top_users: [],
  anomalies: {
    orphan_payment_redeem_codes: 0,
    payment_orders_without_redeem: 0,
    missing_users: 0,
    admin_adjustments_without_notes: 0,
    negative_adjustments: 0,
  },
}

const ledgerSummary = computed(() => summary.value ?? emptySummary)
const sourceDistribution = computed(() => ledgerSummary.value.source_distribution)
const typeDistribution = computed(() => ledgerSummary.value.type_distribution)
const dailySeries = computed(() => ledgerSummary.value.daily_series)
const topUsers = computed(() => ledgerSummary.value.top_users)

const tableColumns = computed(() => [
  t('admin.financeLedger.table.usedAt'),
  t('admin.financeLedger.table.user'),
  t('admin.financeLedger.table.amount'),
  t('admin.financeLedger.table.source'),
  t('admin.financeLedger.table.code'),
  t('admin.financeLedger.table.payment'),
  t('admin.financeLedger.table.notes'),
  t('admin.financeLedger.table.anomalies'),
])

const summaryCards = computed(() => {
  const data = ledgerSummary.value
  return [
    makeCard('total', t('admin.financeLedger.cards.totalAdded'), formatAmount(data.total_added_amount), t('admin.financeLedger.cards.metaRecords', { count: data.record_count }), 'border-emerald-500', 'bg-emerald-500'),
    makeCard('user', t('admin.financeLedger.cards.userRecharge'), formatAmount(data.user_recharge_amount), t('admin.financeLedger.cards.metaUsers', { count: data.unique_users }), 'border-sky-500', 'bg-sky-500'),
    makeCard('online', t('admin.financeLedger.cards.onlinePayment'), formatAmount(data.online_payment_amount), t('admin.financeLedger.sources.online_payment'), 'border-indigo-500', 'bg-indigo-500'),
    makeCard('redeem', t('admin.financeLedger.cards.redeemCode'), formatAmount(data.redeem_code_amount), t('admin.financeLedger.sources.redeem_code'), 'border-amber-500', 'bg-amber-500'),
    makeCard('grant', t('admin.financeLedger.cards.adminGranted'), formatAmount(data.admin_granted_amount), t('admin.financeLedger.sources.admin_grant'), 'border-teal-500', 'bg-teal-500'),
    makeCard('deduct', t('admin.financeLedger.cards.adminDeducted'), formatAmount(data.admin_deducted_amount), t('admin.financeLedger.sources.admin_deduct'), 'border-rose-500', 'bg-rose-500'),
    makeCard('cumulative', t('admin.financeLedger.cards.cumulative'), formatAmount(data.cumulative_added_amount), t('admin.financeLedger.cards.metaCumulative'), 'border-violet-500', 'bg-violet-500'),
    makeCard('records', t('admin.financeLedger.cards.records'), formatInteger(data.record_count), t('admin.financeLedger.cards.metaRecords', { count: data.record_count }), 'border-gray-400', 'bg-gray-400'),
  ]
})

const anomalyRows = computed(() => [
  { key: 'orphan_payment_redeem_code', label: anomalyLabel('orphan_payment_redeem_code'), value: ledgerSummary.value.anomalies.orphan_payment_redeem_codes },
  { key: 'missing_user', label: anomalyLabel('missing_user'), value: ledgerSummary.value.anomalies.missing_users },
  { key: 'admin_adjustment_without_notes', label: anomalyLabel('admin_adjustment_without_notes'), value: ledgerSummary.value.anomalies.admin_adjustments_without_notes },
  { key: 'negative_adjustment', label: anomalyLabel('negative_adjustment'), value: ledgerSummary.value.anomalies.negative_adjustments },
  { key: 'payment_orders_without_redeem', label: anomalyLabel('payment_orders_without_redeem'), value: ledgerSummary.value.anomalies.payment_orders_without_redeem },
])

const totalAnomalies = computed(() => anomalyRows.value.reduce((sum, row) => sum + row.value, 0))
const maxSeriesAmount = computed(() => Math.max(...dailySeries.value.map((point) => Math.abs(point.amount)), 1))
const displayStart = computed(() => pagination.value.total === 0 ? 0 : (pagination.value.page - 1) * pagination.value.page_size + 1)
const displayEnd = computed(() => Math.min(pagination.value.total, pagination.value.page * pagination.value.page_size))

function makeCard(key: string, label: string, value: string, meta: string, borderClass: string, dotClass: string) {
  return { key, label, value, meta, borderClass, dotClass }
}

function setPreset(preset: Exclude<RangePresetKey, 'custom'>) {
  activePreset.value = preset
  const today = startOfDay(new Date())
  let start = new Date(today)
  let end = new Date(today)

  if (preset === 'yesterday') {
    start = addDays(today, -1)
    end = addDays(today, -1)
  } else if (preset === 'last7Days') {
    start = addDays(today, -6)
  } else if (preset === 'thisMonth') {
    start = new Date(today.getFullYear(), today.getMonth(), 1)
  }

  filters.value.startDate = formatDateParam(start)
  filters.value.endDate = formatDateParam(end)
  currentPage.value = 1
  void loadData()
}

function resetFilters() {
  filters.value.source = ''
  filters.value.type = ''
  filters.value.paymentType = ''
  filters.value.search = ''
  filters.value.minAmount = ''
  filters.value.maxAmount = ''
  filters.value.anomalyOnly = false
  setPreset('today')
}

function applyFilters() {
  currentPage.value = 1
  void loadData()
}

function changePage(page: number) {
  if (page < 1 || page > Math.max(pagination.value.pages, 1)) return
  currentPage.value = page
  void loadRecords()
}

async function loadData() {
  loading.value = true
  recordsLoading.value = true
  try {
    const [summaryRes, recordsRes] = await Promise.all([
      financeLedgerAPI.getSummary(buildQueryParams(false)),
      financeLedgerAPI.getRecords(buildQueryParams(true)),
    ])
    summary.value = summaryRes.data
    records.value = recordsRes.data.items
    pagination.value = {
      total: recordsRes.data.total,
      page: recordsRes.data.page,
      page_size: recordsRes.data.page_size,
      pages: recordsRes.data.pages,
    }
    currentPage.value = recordsRes.data.page
    pageSize.value = recordsRes.data.page_size
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.financeLedger.errors', t('admin.financeLedger.loadFailed')))
  } finally {
    loading.value = false
    recordsLoading.value = false
  }
}

async function loadRecords() {
  recordsLoading.value = true
  try {
    const recordsRes = await financeLedgerAPI.getRecords(buildQueryParams(true))
    records.value = recordsRes.data.items
    pagination.value = {
      total: recordsRes.data.total,
      page: recordsRes.data.page,
      page_size: recordsRes.data.page_size,
      pages: recordsRes.data.pages,
    }
    currentPage.value = recordsRes.data.page
    pageSize.value = recordsRes.data.page_size
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.financeLedger.errors', t('admin.financeLedger.loadFailed')))
  } finally {
    recordsLoading.value = false
  }
}

async function exportCsv() {
  exporting.value = true
  try {
    const blob = await financeLedgerAPI.exportCsv(buildQueryParams(false))
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `finance-ledger-${filters.value.startDate || 'start'}-${filters.value.endDate || 'end'}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    appStore.showSuccess(t('admin.financeLedger.exportSuccess'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.financeLedger.errors', t('admin.financeLedger.exportFailed')))
  } finally {
    exporting.value = false
  }
}

function buildQueryParams(includePagination: boolean): FinanceLedgerQueryParams {
  const params: FinanceLedgerQueryParams = {
    start_at: filters.value.startDate || undefined,
    end_at: filters.value.endDate || undefined,
    timezone: userTimezone(),
    source: filters.value.source || undefined,
    type: filters.value.type || undefined,
    search: filters.value.search || undefined,
    payment_type: filters.value.paymentType || undefined,
    min_amount: parseAmount(filters.value.minAmount),
    max_amount: parseAmount(filters.value.maxAmount),
    anomaly_only: filters.value.anomalyOnly ? true : undefined,
  }
  if (includePagination) {
    params.page = currentPage.value
    params.page_size = pageSize.value
  }
  return params
}

function sourceLabel(source: string): string {
  const key = `admin.financeLedger.sources.${source || 'unknown'}`
  const label = t(key)
  return label === key ? (source || t('admin.financeLedger.sources.unknown')) : label
}

function typeLabel(type: string): string {
  const key = `admin.financeLedger.types.${type || 'unknown'}`
  const label = t(key)
  return label === key ? (type || t('admin.financeLedger.types.unknown')) : label
}

function anomalyLabel(anomaly: string): string {
  const key = `admin.financeLedger.anomalies.${anomaly}`
  const label = t(key)
  return label === key ? anomaly : label
}

function sourceBadgeClass(source: string): string {
  switch (source) {
    case 'online_payment':
      return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
    case 'redeem_code':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'admin_grant':
      return 'bg-teal-50 text-teal-700 dark:bg-teal-900/30 dark:text-teal-300'
    case 'admin_deduct':
      return 'bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

function seriesWidth(amount: number): string {
  return `${Math.max(4, Math.min(100, (Math.abs(amount) / maxSeriesAmount.value) * 100))}%`
}

function formatAmount(value: number): string {
  return currencyFormatter.format(value || 0)
}

function formatInteger(value: number): string {
  return integerFormatter.format(value || 0)
}

function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function parseAmount(value: string): number | undefined {
  if (!value.trim()) return undefined
  const n = Number(value)
  return Number.isFinite(n) ? n : undefined
}

function startOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function addDays(date: Date, days: number): Date {
  const out = new Date(date)
  out.setDate(out.getDate() + days)
  return out
}

function formatDateParam(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function userTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai'
  } catch {
    return 'Asia/Shanghai'
  }
}

const currencyFormatter = new Intl.NumberFormat(undefined, {
  style: 'currency',
  currency: 'USD',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

const integerFormatter = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 0,
})

onMounted(() => {
  const today = startOfDay(new Date())
  filters.value.startDate = formatDateParam(today)
  filters.value.endDate = formatDateParam(today)
  void loadData()
})
</script>
