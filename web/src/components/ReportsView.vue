<script setup>
import { inject } from 'vue'
const { state, ...actions } = inject('workspace')
</script>

<template>
  <section id="reports-view" class="page">
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
        <span>广告主</span
        ><strong id="stat-advertisers">{{ actions.number(state.records.advertiser_count) }}</strong
        ><small>个广告主有有效记录</small>
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
                ? '全部成员的上传记录；行数、金额和 CSV 为当前有效数据'
                : '我的上传记录；行数、金额和 CSV 为当前有效数据'
            }}
          </p>
        </div>
        <form id="filter-form" class="filters" @submit.prevent="actions.filterRecords">
          <input
            id="search"
            placeholder="搜索文件、广告主、代理商或人员"
            aria-label="搜索记录"
            v-model="state.filters.q"
          />
          <button class="button" type="submit">筛选</button>
        </form>
      </div>
      <div id="list-error" class="notice error" v-if="state.listError">{{ state.listError }}</div>
      <div class="table-wrap">
        <table class="records-table">
          <thead>
            <tr>
              <th>结算文件</th>
              <th>广告主</th>
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
                {{ record.advertisers?.join('、') || '—' }}
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
                    >当前 CSV ↓</a
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
            state.filters.q
              ? '没有匹配的上传记录'
              : '从第一份结算文件开始'
          }}
        </h3>
        <p>
          {{
            state.filters.q
              ? '试试其他文件名、广告主或人员关键词。'
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
