<script setup>
import { inject } from 'vue'
import DataTable from './DataTable.vue'
const { state, ...actions } = inject('workspace')
</script>

<template>
  <dialog
    id="upload-dialog"
    class="upload-dialog"
    :ref="(element) => actions.dialogRef('upload', element)"
    @cancel="state.uploading && $event.preventDefault()"
  >
    <div class="dialog-header">
      <div>
        <span class="eyebrow">NEW SETTLEMENT</span>
        <h2>上传结算文件</h2>
      </div>
      <button
        class="icon-button"
        data-close="upload-dialog"
        @click="actions.closeDialog('upload')"
        :disabled="state.uploading"
        aria-label="关闭"
      >
        ×
      </button>
    </div>
    <form id="upload-form" @submit.prevent="actions.upload">
      <fieldset class="upload-fields" :disabled="state.uploading">
        <div class="dialog-content">
          <label
            id="dropzone"
            class="dropzone"
            :class="{ drag: state.dragging }"
            @dragover.prevent="state.dragging = true"
            @dragleave="state.dragging = false"
            @drop.prevent="actions.dropFile"
            ><input
              id="file"
              type="file"
              accept=".xlsx"
              @change="actions.chooseFile($event.target.files[0])"
            /><span class="upload-symbol">↑</span
            ><strong id="file-label">{{
              state.file ? state.file.name : '点击选择，或拖拽 Excel 文件到这里'
            }}</strong
            ><span class="muted small">支持 .xlsx · 最大 20 MB · 最多 20,000 行</span></label
          >
          <div class="notice">
            必须包含：<b>代理商、广告主、日期、任务名称、结算数、结算单价、结算金额</b>。结算三列允许为 0 或空，广告主直接作为业务账户，其他列原样保留，首行为表头。
          </div>
          <div class="notice">
            仅在本人的数据内，按代理商、广告主、日期、任务名称匹配更新，未匹配则新增；文件名不参与匹配，本次文件未包含的旧记录保留。
          </div>
          <label id="sheet-label" v-if="state.sheets.length"
            >选择工作表<select id="sheet" v-model="state.sheet" @change="actions.preview">
              <option v-for="sheet in state.sheets" :key="sheet" :value="sheet">
                {{ sheet }}
              </option></select
            ><small class="muted">每次上传一个工作表，避免明细与汇总重复计入。</small></label
          >
          <div class="form-row">
            <label
              >运营人员<input
                id="operator"
                maxlength="191"
                required
                placeholder="填写负责此次结算的运营人员"
                v-model="state.operator" /></label
            >
          </div>
          <div id="validation" role="status" aria-live="polite">
            <div v-if="state.previewLoading" class="notice">正在读取并校验文件，请稍候…</div>
            <div v-if="state.uploadError" class="notice error">{{ state.uploadError }}</div>
            <template v-else-if="state.preview"
              ><div v-if="state.valid" class="notice success">
                ✓ 校验通过 · {{ actions.number(state.preview.row_count) }} 行数据 · 结算金额
                {{ actions.money(state.preview.result.total_amount) }} 元
              </div>
              <div v-else class="notice error">
                <b>校验未通过，请修改 Excel 后重新选择文件。</b>
                <ul>
                  <li v-for="(error, index) in state.preview.result.errors" :key="index">
                    {{ error }}
                  </li>
                </ul>
              </div></template
            >
          </div>
          <div id="preview-container" v-if="state.preview?.result.rows?.length">
            <div class="preview-heading">
              <h3>数据预览</h3>
              <span class="muted small">前 10 行 · 标准字段已整理</span>
            </div>
            <div class="table-wrap preview-table">
              <DataTable
                id="preview-table"
                :columns="state.preview.result.columns"
                :rows="state.preview.result.rows"
              />
            </div>
          </div>
        </div>
        <div class="dialog-footer">
          <span class="muted small"
            >上传身份：<b id="upload-identity">{{ state.user?.name }}</b></span
          >
          <div>
            <button
              type="button"
              class="button"
              data-close="upload-dialog"
              @click="actions.closeDialog('upload')"
              :disabled="state.uploading"
            >
              取消</button
            ><button
              id="submit-upload"
              class="button primary"
              type="submit"
              :disabled="!state.valid || state.uploading"
            >
              {{ state.uploading ? '正在保存…' : '确认上传并入库' }}
            </button>
          </div>
        </div>
      </fieldset>
    </form>
  </dialog>
</template>
