<script setup>
import { inject } from 'vue'
const { state, ...actions } = inject('workspace')
</script>

<template>
  <div id="login-view" class="login-shell" v-if="state.ready && !state.user">
    <section class="login-story">
      <a class="brand" :href="actions.sitePath('/')"
        ><span class="brand-mark">a</span> ADN <span class="brand-sub">REPORTS</span></a
      >
      <div>
        <span class="eyebrow">SETTLEMENT WORKSPACE</span>
        <h1>让每一笔结算，<br />清晰可见。</h1>
        <p>汇集多渠道结算文件，连接账户与运营人员，<br />让数据整理、核对与追溯在一个地方完成。</p>
        <div class="story-grid">
          <span>01 <b>上传与校验</b></span
          ><span>02 <b>统一归档</b></span
          ><span>03 <b>查看与追溯</b></span>
        </div>
      </div>
      <small>ADN 结算数据平台 · V1.0</small>
    </section>
    <section class="login-panel">
      <div class="login-card">
        <span class="eyebrow">WELCOME BACK</span>
        <h2>登录工作台</h2>
        <p class="muted">使用钉钉身份，进入你的结算工作空间。</p>
        <div id="login-error" class="notice error" v-if="state.loginError">
          {{ state.loginError }}
        </div>
        <a
          id="ding-login"
          class="button primary login-button"
          :href="actions.sitePath('/auth/login')"
          v-if="state.auth.dingtalk"
          >钉钉扫码登录 <span>↗</span></a
        >
        <p id="ding-hint" class="notice" v-if="!state.auth.dingtalk">
          {{
            state.auth.local_admin
              ? '钉钉应用尚未配置，可先使用本地管理员登录。'
              : '登录方式尚未配置，请联系管理员配置钉钉应用。'
          }}
        </p>
        <details id="local-section" v-if="state.auth.local_admin">
          <summary>本地管理员登录</summary>
          <form id="login-form" @submit.prevent="actions.login">
            <label
              >管理员账号<input
                id="username"
                autocomplete="username"
                value="admin"
                required
                v-model="state.login.username" /></label
            ><label
              >密码<input
                id="password"
                type="password"
                autocomplete="current-password"
                required
                v-model="state.login.password" /></label
            ><button class="button dark" type="submit" :disabled="state.loggingIn">登录</button>
          </form>
        </details>
        <p class="login-foot">个人账号仅可查看自己的上传记录<br />管理员可查看全部账号的数据</p>
      </div>
    </section>
  </div>
</template>
