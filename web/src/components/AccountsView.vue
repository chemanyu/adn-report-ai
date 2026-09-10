<script setup>
import { inject } from 'vue'
const { state, ...actions } = inject('workspace')
</script>

<template>
  <section id="accounts-view" class="page" v-show="state.view === 'accounts'">
    <div class="page-heading">
      <div>
        <span class="eyebrow">ADN ACCOUNTS</span>
        <h1>ADN 账户</h1>
        <p class="muted">维护结算文件关联的账户，上传时直接选择。</p>
      </div>
      <button id="open-account" class="button primary" @click="actions.openAccount">
        ＋ 新增账户
      </button>
    </div>
    <div id="accounts-grid" class="account-grid">
      <article v-for="account in state.accounts" :key="account.id" class="account-card">
        <div class="account-mark">▦</div>
        <h3>{{ account.name }}</h3>
        <p>{{ account.code }}</p>
        <small>● 可用于结算关联</small>
      </article>
      <div v-if="!state.accounts.length" class="empty">
        <h3>还没有 ADN 账户</h3>
        <p>点击右上角“新增账户”，填写实际账户标识。</p>
      </div>
    </div>
    <p class="page-note">账户目录供平台成员选择；上传记录和结算明细仍按登录身份隔离。</p>
  </section>
</template>
