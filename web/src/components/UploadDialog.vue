<script setup>
import { inject } from 'vue'
import DataTable from './DataTable.vue'
const { state, ...actions } = inject('workspace')
function selectSheets(all) {
  state.selectedSheets = all
    ? [
        ...state.selectedSheets,
        ...state.sheets.filter((sheet) => !state.selectedSheets.includes(sheet)),
      ]
    : []
  actions.preview()
}
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
            必须包含：<b>代理商、广告主、日期、任务名称、结算数、结算单价、结算金额</b>。结算三列允许为
            0 或空，广告主直接作为业务账户，其他列原样保留，首行为表头。
          </div>
          <div class="notice">
            仅在本人的数据内，按代理商、广告主、日期、任务名称及可选的 fix
            列匹配更新，未匹配则新增；文件名不参与匹配，本次文件未包含的旧记录保留。若前四项相同的多行需分别保留，请添加小写
            fix 列并填写不同的稳定标识。
          </div>
          <section class="sheet-picker" v-if="state.sheets.length" aria-labelledby="sheet-label">
            <div class="sheet-toolbar">
              <div>
                <strong id="sheet-label">选择工作表</strong>
                <span class="sheet-count"
                  >已选 {{ state.selectedSheets.length }} / {{ state.sheets.length }}</span
                >
              </div>
              <div class="sheet-tools">
                <button
                  type="button"
                  @click="selectSheets(true)"
                  :disabled="state.selectedSheets.length === state.sheets.length"
                >
                  全选
                </button>
                <button
                  type="button"
                  @click="selectSheets(false)"
                  :disabled="!state.selectedSheets.length"
                >
                  清空
                </button>
              </div>
            </div>
            <div class="sheet-list">
              <label
                class="sheet-option"
                :class="{ selected: state.selectedSheets.includes(sheet) }"
                v-for="sheet in state.sheets"
                :key="sheet"
              >
                <input
                  type="checkbox"
                  v-model="state.selectedSheets"
                  :value="sheet"
                  @change="actions.preview"
                />
                <span class="sheet-name">{{ sheet }}</span>
              </label>
            </div>
            <small class="sheet-hint muted"
              >可多选，按勾选顺序入库；相同业务键以后上传的表为准。请勿同时选择明细和汇总表。</small
            >
          </section>
          <div class="form-row">
            <label
              >运营人员<input
                id="operator"
                required
                placeholder="填写负责此次结算的运营人员"
                v-model="state.operator"
            /></label>
          </div>
          <div id="validation" role="status" aria-live="polite">
            <div v-if="state.previewLoading" class="notice">正在读取并校验文件，请稍候…</div>
            <div v-if="state.uploadError" class="notice error">{{ state.uploadError }}</div>
            <template v-for="preview in state.previews" :key="preview.result.sheet"
              ><div v-if="preview.valid" class="notice success">
                ✓ {{ preview.result.sheet }} 校验通过 ·
                {{ actions.number(preview.row_count) }} 行数据 · 结算金额
                {{ actions.money(preview.result.total_amount) }} 元
              </div>
              <div v-else class="notice error">
                <b>「{{ preview.result.sheet }}」校验未通过，请修改 Excel 后重新选择文件。</b>
                <ul>
                  <li v-for="(error, index) in preview.result.errors" :key="index">
                    {{ error }}
                  </li>
                </ul>
              </div></template
            >
          </div>
          <div v-for="preview in state.previews" :key="preview.result.sheet">
            <div class="preview-heading">
              <h3>{{ preview.result.sheet }} · 数据预览</h3>
              <span class="muted small">前 10 行 · 标准字段已整理</span>
            </div>
            <div class="table-wrap preview-table">
              <DataTable
                :id="`preview-table-${preview.result.sheet}`"
                :columns="preview.result.columns"
                :rows="preview.result.rows"
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

<style scoped>
.sheet-picker {
  margin: 18px 0;
  border: 1px solid var(--line);
  border-radius: 10px;
  overflow: hidden;
}
.sheet-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  background: var(--bg);
}
.sheet-toolbar strong {
  font-size: 13px;
}
.sheet-count {
  margin-left: 10px;
  font-size: 12px;
  color: var(--muted);
  white-space: nowrap;
}
.sheet-tools {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
}
.sheet-tools button {
  padding: 2px;
  border: 0;
  background: transparent;
  color: var(--green);
  font-size: 12px;
}
.sheet-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  max-height: 260px;
  overflow-y: auto;
  padding: 12px;
}
.sheet-list .sheet-option {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 10px;
  margin: 0;
  padding: 10px 12px;
  border: 1px solid var(--line);
  border-radius: 7px;
  cursor: pointer;
  min-width: 0;
  background: white;
}
.sheet-list .sheet-option:hover {
  border-color: #9ccbbb;
}
.sheet-list .sheet-option.selected {
  border-color: #9ccbbb;
  background: var(--soft);
}
.sheet-list .sheet-option:focus-within {
  outline: 2px solid var(--green);
  outline-offset: -2px;
}
.sheet-option input[type='checkbox'] {
  width: 16px;
  height: 16px;
  flex: 0 0 16px;
  margin: 0;
  padding: 0;
  accent-color: var(--green);
}
.sheet-name {
  overflow-wrap: anywhere;
  line-height: 1.5;
}
.sheet-hint {
  display: block;
  padding: 0 14px 12px;
  font-size: 12px;
}
@media (max-width: 600px) {
  .sheet-list {
    grid-template-columns: 1fr;
  }
  .sheet-count {
    display: block;
    margin-left: 0;
  }
}
</style>
