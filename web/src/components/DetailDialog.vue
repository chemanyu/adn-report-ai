<script setup>
import { inject } from 'vue'
import DataTable from './DataTable.vue'
const { state, ...actions } = inject('workspace')
</script>

<template>
  <dialog
    id="detail-dialog"
    class="detail-dialog"
    :ref="(element) => actions.dialogRef('detail', element)"
  >
    <div class="dialog-header">
      <div>
        <span class="eyebrow">SETTLEMENT DETAILS</span>
        <h2 id="detail-title">{{ state.detail?.upload.filename || '正在加载结算明细…' }}</h2>
      </div>
      <button
        class="icon-button"
        data-close="detail-dialog"
        @click="actions.closeDialog('detail')"
        aria-label="关闭"
      >
        ×
      </button>
    </div>
    <div class="dialog-content">
      <p class="muted small">
        这里展示当前有效明细；同名文件、同一工作表重复上传会更新此记录，未匹配的旧明细保留。
      </p>
      <div id="detail-meta" class="detail-meta">
        <template v-if="state.detail"
          ><div><span>广告主</span>{{ state.detail.upload.advertisers?.join('、') || '—' }}</div>
          <div><span>运营人员</span>{{ state.detail.upload.operator }}</div>
          <div><span>上传人员</span>{{ state.detail.upload.uploader }}</div>
          <div><span>工作表</span>{{ state.detail.upload.sheet }}</div>
          <div>
            <span>结算日期</span
            >{{
              state.detail.upload.date_from
                ? `${state.detail.upload.date_from} 至 ${state.detail.upload.date_to}`
                : '无当前有效明细'
            }}
          </div>
          <div><span>结算金额</span>{{ actions.money(state.detail.upload.total_amount) }} 元</div>
          <div>
            <span>最近上传时间</span>{{ actions.datetime(state.detail.upload.updated_at) }}
          </div></template
        >
      </div>
      <div id="detail-error" class="notice error" v-if="state.detailError">
        {{ state.detailError }}
      </div>
      <details v-if="state.detail" class="upload-history">
        <summary>上传历史（{{ state.detail.upload.history?.length || 0 }} 次）</summary>
        <p class="muted small">历史仅记录上传操作，明细和下载反映当前有效数据。</p>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>上传时间</th>
                <th>上传人员</th>
                <th>运营人员</th>
                <th>新增行数</th>
                <th>更新行数</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="event in state.detail.upload.history" :key="event.id">
                <td>{{ actions.datetime(event.uploaded_at) }}</td>
                <td>{{ event.uploader }}</td>
                <td>{{ event.operator }}</td>
                <td>
                  {{
                    event.inserted_rows == null ? '历史未记录' : actions.number(event.inserted_rows)
                  }}
                </td>
                <td>
                  {{
                    event.updated_rows == null ? '历史未记录' : actions.number(event.updated_rows)
                  }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </details>
      <div class="preview-heading">
        <h3>结算数据</h3>
        <a
          id="detail-download"
          class="button"
          v-if="state.detail"
          :href="actions.sitePath(`/api/uploads/${state.detail.upload.id}/csv`)"
          >↓ 下载当前明细 CSV</a
        >
      </div>
      <div class="table-wrap detail-table">
        <DataTable
          id="detail-table"
          :columns="state.detail?.upload.columns || []"
          :rows="state.detail?.rows || []"
          :class="{ loading: state.detailLoading }"
        />
      </div>
      <div class="pagination">
        <span id="detail-page-label"
          >共 {{ actions.number(state.detail?.upload.row_count || 0) }} 行 · 第
          {{ state.detailPage }} /
          {{ Math.max(1, Math.ceil((state.detail?.upload.row_count || 0) / 100)) }} 页</span
        >
        <div>
          <button
            id="detail-prev"
            class="button"
            :disabled="state.detailLoading || state.detailPage <= 1"
            @click="actions.detailPage(-1)"
          >
            上一页</button
          ><button
            id="detail-next"
            class="button"
            :disabled="
              state.detailLoading ||
              !state.detail ||
              state.detailPage * 100 >= state.detail.upload.row_count
            "
            @click="actions.detailPage(1)"
          >
            下一页
          </button>
        </div>
      </div>
    </div>
  </dialog>
</template>
