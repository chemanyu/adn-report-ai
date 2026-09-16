<script setup>
import { computed } from 'vue'
import { formatDecimal } from '../utils/format'

const props = defineProps({
  detail: { type: Boolean, default: false },
  columns: { type: Array, default: () => [] },
  rows: { type: Array, default: () => [] },
})

const requiredColumns = ['代理商', '广告主', '日期', '任务名称', '结算数', '结算单价', '结算金额']
const displayColumns = computed(() => {
  if (props.detail) {
    return [
      ['代理商', 'agency'],
      ['代理商ID', 'agency_id'],
      ['广告主', 'advertiser'],
      ['广告主ID', 'advertiser_id'],
      ['项目ID', 'project_ids'],
      ['日期', 'date'],
      ['任务名称', 'task_name'],
      ['结算数', 'count'],
      ['结算单价', 'price'],
      ['结算金额', 'amount'],
    ].map(([name, field]) => ({ name, field, index: field }))
  }
  const columns = props.columns.map((name, index) => ({ name, index }))
  const required = requiredColumns.flatMap((name) =>
    columns.filter((column) => column.name.trim() === name),
  )
  const extra = columns.filter((column) => !requiredColumns.includes(column.name.trim()))
  return [...required, ...extra]
})

function displayValue(row, column) {
  const value = column.field ? row[column.field] : row.values[column.index]
  if (column.field === 'project_ids') return value?.length ? value.join('、') : '0'
  if (column.field === 'agency_id' || column.field === 'advertiser_id')
    return value && value !== '0' ? String(value) : '—'
  return requiredColumns.slice(4).includes(column.name.trim())
    ? formatDecimal(value)
    : String(value ?? '')
}
</script>

<template>
  <table>
    <thead>
      <tr>
        <th v-for="column in displayColumns" :key="column.index">{{ column.name }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="(row, index) in rows" :key="index">
        <td v-for="column in displayColumns" :key="column.index" :title="displayValue(row, column)">
          {{ displayValue(row, column) }}
        </td>
      </tr>
    </tbody>
  </table>
</template>
