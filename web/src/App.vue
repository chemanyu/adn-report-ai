<script setup>
import { provide } from 'vue'
import { useWorkspace } from './composables/useWorkspace'
import LoginView from './components/LoginView.vue'
import ReportsView from './components/ReportsView.vue'
import UploadDialog from './components/UploadDialog.vue'
import DetailDialog from './components/DetailDialog.vue'

const workspace = useWorkspace()
provide('workspace', workspace)
const { state, ...actions } = workspace
</script>

<template>
  <p v-if="!state.ready" class="notice">正在加载工作台…</p>
  <LoginView />
  <div id="app-view" v-if="state.user">
    <aside class="sidebar">
      <a class="brand" :href="actions.sitePath('/')"
        ><span class="brand-mark">a</span> ADN <span class="brand-sub">REPORTS</span></a
      >
      <div class="nav-label">工作空间</div>
      <nav>
        <button
          id="nav-reports"
          class="nav-item active"
        >
          <span>▤</span> 结算文件</button
        ><button id="logout-mobile" class="nav-item mobile-only" @click="actions.logout">
          退出
        </button>
      </nav>
      <div class="sidebar-bottom">
        <div class="version"><span class="status-dot"></span> 结算数据平台 <span>V1.0</span></div>
        <div class="user-card">
          <span id="avatar" class="avatar">{{ state.user.name.slice(0, 1) }}</span>
          <div>
            <strong id="user-name">{{ state.user.name }}</strong
            ><small id="user-role">{{ state.user.admin ? '管理员' : '运营成员' }}</small>
          </div>
          <button id="logout" title="退出登录" aria-label="退出登录" @click="actions.logout">
            ↗
          </button>
        </div>
        <details class="identity">
          <summary>查看登录身份</summary>
          <small id="user-identity">{{ state.user.identity }}</small>
        </details>
      </div>
    </aside>
    <main>
      <header class="topbar">
        <span
          >工作空间 <span class="slash">/</span>
          <b id="breadcrumb">结算文件</b></span
        ><span id="scope-label" class="scope">{{
          state.user.admin ? '管理员 · 全部上传记录' : '个人空间 · 仅自己的记录'
        }}</span>
      </header>
      <ReportsView />
    </main>
  </div>
  <UploadDialog />
  <DetailDialog />
  <div id="toast" class="toast" role="status" v-if="state.toast">{{ state.toast }}</div>
</template>
