<script setup>
import { inject } from 'vue'
const { state, ...actions } = inject('workspace')
</script>

<template>
  <dialog
    id="account-dialog"
    class="small-dialog"
    :ref="(element) => actions.dialogRef('account', element)"
  >
    <div class="dialog-header">
      <h2>新增 ADN 账户</h2>
      <button
        class="icon-button"
        data-close="account-dialog"
        @click="actions.closeDialog('account')"
        aria-label="关闭"
      >
        ×
      </button>
    </div>
    <form id="account-form" @submit.prevent="actions.saveAccount">
      <div class="dialog-content">
        <label
          >账户名称<input
            id="account-name"
            required
            maxlength="191"
            placeholder="例如：美数科技 · 主账户"
            v-model="state.account.name" /></label
        ><label
          >ADN 账户标识<input
            id="account-code"
            required
            maxlength="191"
            placeholder="填写实际 ADN 账号或 ID"
            v-model="state.account.code"
        /></label>
        <div id="account-error" class="notice error" v-if="state.accountError">
          {{ state.accountError }}
        </div>
      </div>
      <div class="dialog-footer">
        <span></span
        ><button class="button primary" type="submit" :disabled="state.savingAccount">
          保存账户
        </button>
      </div>
    </form>
  </dialog>
</template>
