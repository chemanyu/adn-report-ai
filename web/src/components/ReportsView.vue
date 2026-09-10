<script setup>
import { inject } from 'vue'
const { state, ...actions } = inject('workspace')
</script>

<template>
  <section id="reports-view" class="page" v-show="state.view === 'reports'">
    <div class="page-heading">
      <div>
        <span class="eyebrow">SETTLEMENT REPORTS</span>
        <h1>结算文件</h1>
        <p class="muted">上传、整理并追溯你的每一份结算数据。</p>
      </div>
      <div class="actions">
        <a class="button" :href="actions.sitePath('/api/template')">↓ 下载模板</a
        ><button id="open-upload" class="button primary" @click="actions.openUpload">
          ＋ 上传结算文件
        </button>
      </div>
    </div>
    <div class="stats">
      <article>
        <span>结算文件</span
        ><strong id="stat-files">{{ actions.number(state.records.total) }}</strong
        ><small>份已归档文件</small>
      </article>
      <article>
        <span>结算数据</span
        ><strong id="stat-rows">{{ actions.number(state.records.row_count) }}</strong
        ><small>条明细记录</small>
      </article>
      <article>
        <span>结算金额</span
        ><strong id="stat-amount">{{ actions.money(state.records.total_amount) }}</strong
        ><small>当前筛选范围合计 · 元</small>
      </article>
      <article>
        <span>关联 ADN 账户</span
        ><strong id="stat-accounts">{{ actions.number(state.records.account_count) }}</strong
        ><small>个账户有结算记录</small>
      </article>
    </div>
    <div class="card">
      <div class="card-toolbar">
        <div>
          <h3>
            上传记录 <span id="record-count" class="count">{{ state.records.total }}</span>
          </h3>
          <p id="record-hint" class="muted small">
            {{
              state.user.admin
                ? '全部成员的上传记录，按最近上传时间排序'
                : '我的上传记录，按最近上传时间排序'
            }}
          </p>
        </div>
        <form id="filter-form" class="filters" @submit.prevent="actions.filterRecords">
          <input
            id="search"
            placeholder="搜索文件、运营或上传人员"
            aria-label="搜索记录"
            v-model="state.filters.q"
          /><select
            id="filter-account"
            aria-label="按账户筛选"
            v-model="state.filters.account_id"
            @change="actions.filterRecords"
          >
            <option value="">全部 ADN 账户</option>
            <option v-for="account in state.accounts" :key="account.id" :value="String(account.id)">
              {{ account.name }} · {{ account.code }}
            </option></select
          ><button class="button" type="submit">筛选</button>
        </form>
      </div>
      <div id="list-error" class="notice error" v-if="state.listError">{{ state.listError }}</div>
      <div class="table-wrap">
        <table class="records-table">
          <thead>
            <tr>
              <th>结算文件</th>
              <th>ADN 账户</th>
              <th>运营 / 上传人员</th>
              <th>数据行数</th>
              <th>结算金额（元）</th>
              <th>上传时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody id="records" :class="{ loading: state.listLoading }">
            <tr v-for="record in state.records.items" :key="record.id">
              <td>
                <div class="file-cell">
                  <span class="file-icon">XLSX</span>
                  <div>
                    <b :title="record.filename">{{ record.filename }}</b
                    ><small
                      >{{ record.sheet }} · {{ record.date_from
                      }}{{
                        record.date_to !== record.date_from ? ' 至 ' + record.date_to : ''
                      }}</small
                    >
                  </div>
                </div>
              </td>
              <td>
                {{ record.account }}<small>{{ record.account_code }}</small>
              </td>
              <td>
                {{ record.operator }}<small>上传：{{ record.uploader }}</small>
              </td>
              <td>{{ actions.number(record.row_count) }}</td>
              <td class="money">{{ actions.money(record.total_amount) }}</td>
              <td>{{ actions.datetime(record.created_at) }}</td>
              <td>
                <div class="row-actions">
                  <button
                    class="link-button"
                    :data-detail="record.id"
                    @click="actions.openDetail(record.id)"
                  >
                    查看明细</button
                  ><a class="link-button" :href="actions.sitePath(`/api/uploads/${record.id}/csv`)"
                    >CSV ↓</a
                  >
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div
        id="empty-records"
        class="empty"
        v-if="!state.listLoading && !state.records.items.length"
      >
        <div class="empty-icon">▤</div>
        <h3>
          {{
            state.filters.q || state.filters.account_id
              ? '没有匹配的上传记录'
              : '从第一份结算文件开始'
          }}
        </h3>
        <p>
          {{
            state.filters.q || state.filters.account_id
              ? '试试其他关键词，或切换账户筛选。'
              : '上传 Excel，校验后自动保存 CSV 和结算明细。'
          }}
        </p>
        <button id="empty-upload" class="button primary" @click="actions.openUpload">
          ＋ 上传结算文件
        </button>
      </div>
      <footer class="pagination">
        <span id="list-page-label"
          >共 {{ state.records.total }} 份文件 · 第 {{ state.page }} /
          {{ Math.max(1, Math.ceil(state.records.total / 20)) }} 页</span
        >
        <div>
          <button
            id="list-prev"
            class="button"
            :disabled="state.page <= 1"
            @click="actions.listPage(-1)"
          >
            上一页</button
          ><button
            id="list-next"
            class="button"
            :disabled="state.page * 20 >= state.records.total"
            @click="actions.listPage(1)"
          >
            下一页
          </button>
        </div>
      </footer>
    </div>
    <p class="page-note">
      <span class="status-dot"></span>
      标准字段统一整理，其他列完整保留。统计仅汇总已成功入库的数据。
    </p>
  </section>
</template>
