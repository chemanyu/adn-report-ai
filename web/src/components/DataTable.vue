<script setup>
import { computed } from 'vue'
import { formatDecimal } from '../utils/format'

const props = defineProps({
  columns: { type: Array, default: () => [] },
  rows: { type: Array, default: () => [] },
})

const requiredColumns = ['日期', '结算数', '结算单价', '结算金额']
const displayColumns = computed(() => {
  const columns = props.columns.map((name, index) => ({ name, index }))
  const required = requiredColumns.flatMap((name) =>
    columns.filter((column) => column.name.trim() === name),
  )
  const extra = columns.filter((column) => !requiredColumns.includes(column.name.trim()))
  return [...required, ...extra]
})

function displayValue(row, column) {
  const value = row.values[column.index]
  return requiredColumns.slice(1).includes(column.name.trim())
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
      <tr v-for="(row, index) in rows" :key="row.source_row ?? index">
        <td v-for="column in displayColumns" :key="column.index" :title="displayValue(row, column)">
          {{ displayValue(row, column) }}
        </td>
      </tr>
    </tbody>
  </table>
</template>
