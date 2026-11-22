<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import {
  getLoginHistory, getLoginNotificationSettings,
  saveLoginNotificationSettings, testLoginNotification,
  type LoginHistory,
} from "@/api/user/security";

const history = ref<LoginHistory>();
const loading = ref(false);
const saving = ref(false);
const testing = ref(false);
const ready = ref(false);
const tokenConfigured = ref(false);
const form = reactive({ enabled: false, botToken: "", chatId: "", proxyUrl: "" });

function formatTime(value?: string) {
  if (!value || value.startsWith("0001-")) return "—";
  return new Date(value).toLocaleString("zh-CN", { hour12: false });
}

async function refreshHistory() {
  loading.value = true;
  try { history.value = (await getLoginHistory()).data; }
  catch { /* The shared request handler displays errors. */ }
  finally { loading.value = false; }
}

async function loadSettings() {
  try {
    const { data } = await getLoginNotificationSettings();
    form.enabled = data.enabled;
    form.chatId = data.chatId;
    form.proxyUrl = data.proxyUrl;
    tokenConfigured.value = data.botTokenConfigured;
    ready.value = true;
  } catch { /* Do not allow saving empty settings after a failed load. */ }
}

async function saveSettings() {
  if (!ready.value || saving.value) return;
  saving.value = true;
  try {
    const { data } = await saveLoginNotificationSettings({ ...form });
    tokenConfigured.value = data.botTokenConfigured;
    form.botToken = "";
    ElMessage.success("登录提醒设置已保存");
  } catch { /* The shared request handler displays errors. */ }
  finally { saving.value = false; }
}

async function sendTest() {
  testing.value = true;
  try {
    await testLoginNotification();
    ElMessage.success("测试提醒已排队，请查看 Telegram；发送状态可点击刷新查看");
  } catch { /* The shared request handler displays errors. */ }
  finally { testing.value = false; }
}

onMounted(() => { refreshHistory(); loadSettings(); });
</script>

<template>
  <el-card class="login-security-card">
    <template #header>
      <div class="login-security-header">
        <span>登录记录</span>
        <el-button :loading="loading" @click="refreshHistory">刷新</el-button>
      </div>
    </template>
    <p class="login-security-note">仅保留最近 20 条登录成功和失败记录，新记录覆盖最旧记录。记录保存在内存中，服务重启后清空。</p>
    <p v-if="history" class="login-security-note">本次记录开始时间：{{ formatTime(history.startedAt) }}</p>
    <el-table v-loading="loading" :data="history?.list ?? []" empty-text="暂无登录记录">
      <el-table-column label="登录时间" min-width="170">
        <template #default="{ row }">{{ formatTime(row.time) }}</template>
      </el-table-column>
      <el-table-column prop="username" label="账号" min-width="100" show-overflow-tooltip />
      <el-table-column prop="ip" label="来源 IP" min-width="180" show-overflow-tooltip />
      <el-table-column label="结果" width="90">
        <template #default="{ row }"><el-tag :type="row.success ? 'success' : 'danger'">{{ row.success ? '成功' : '失败' }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="reason" label="说明" min-width="140" />
      <el-table-column prop="ipSource" label="IP 识别方式" min-width="150" />
      <el-table-column prop="userAgent" label="设备 / 浏览器" min-width="200" show-overflow-tooltip />
    </el-table>
  </el-card>

  <el-card class="login-security-card">
    <template #header>Telegram 登录提醒</template>
    <p class="login-security-note">成功登录时立即提醒；5 分钟内累计失败 5 次时合并提醒，失败提醒最多每 5 分钟一条。保存设置后可发送测试消息。</p>
    <el-form :model="form" label-width="110px" :disabled="!ready || saving" @submit.prevent="saveSettings">
      <el-form-item label="启用提醒"><el-switch v-model="form.enabled" /></el-form-item>
      <el-form-item label="Bot Token">
        <el-input v-model="form.botToken" type="password" show-password autocomplete="new-password" :placeholder="tokenConfigured ? '已配置，留空保留原 Token' : '请输入机器人 Token'" />
      </el-form-item>
      <el-form-item label="Chat ID"><el-input v-model="form.chatId" placeholder="接收提醒的 Chat ID" /></el-form-item>
      <el-form-item label="HTTP 代理"><el-input v-model="form.proxyUrl" placeholder="可选，例如 http://host.docker.internal:7890" /></el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="saving" @click="saveSettings">保存设置</el-button>
        <el-button :loading="testing" @click="sendTest">测试已保存配置</el-button>
      </el-form-item>
    </el-form>
    <p v-if="history" class="login-security-note">
      本次运行发送成功 {{ history.notifications.sent }} 次，失败 {{ history.notifications.failed }} 次，队列满跳过 {{ history.notifications.dropped }} 次。
      配置变更取消 {{ history.notifications.canceled }} 次。
      最近发送：{{ formatTime(history.notifications.lastSentAt) }}。
    </p>
    <el-alert v-if="history?.notifications.lastError" :title="history.notifications.lastError" type="warning" :closable="false" />
  </el-card>
</template>

<style scoped>
.login-security-card { margin: 10px; }
.login-security-header { display: flex; align-items: center; justify-content: space-between; }
.login-security-note { color: var(--el-text-color-secondary); font-size: 13px; line-height: 1.7; }
.el-form { max-width: 700px; margin-top: 20px; }
</style>
