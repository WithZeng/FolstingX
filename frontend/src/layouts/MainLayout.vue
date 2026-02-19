<template>
  <n-layout has-sider class="layout-root">
    <n-layout-sider bordered :width="240" collapse-mode="width">
      <div class="logo">FolstingX</div>
      <n-menu :options="menuOptions" :value="activePath" @update:value="onMenuSelect" />
    </n-layout-sider>

    <n-layout>
      <n-layout-header bordered class="topbar">
        <n-space align="center">
          <span>当前用户：{{ username }}</span>
          <n-button size="small" tertiary @click="logout">退出</n-button>
        </n-space>
      </n-layout-header>
      <n-layout-content content-style="padding: 20px;">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, NMenu, NSpace, NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/stores/auth";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const { user } = storeToRefs(authStore);

onMounted(async () => {
  // 页面刷新后恢复用户资料。
  if (!user.value && authStore.isLoggedIn) {
    try {
      await authStore.fetchProfile();
    } catch {
      // 忽略，401 会由拦截器处理。
    }
  }
});

const menuOptions = [
  { label: () => h("span", "仪表盘"), key: "/dashboard" },
  { label: () => h("span", "隧道管理"), key: "/tunnels" },
  { label: () => h("span", "规则管理"), key: "/rules" },
  { label: () => h("span", "节点管理"), key: "/nodes" },
  { label: () => h("span", "用户管理"), key: "/users" },
  { label: () => h("span", "日志中心"), key: "/logs" },
  { label: () => h("span", "系统设置"), key: "/settings" },
];

const activePath = computed(() => route.path);
const username = computed(() => user.value?.username || "-");

const onMenuSelect = (key: string) => {
  router.push(key);
};

const logout = () => {
  authStore.logout();
  router.push("/login");
};
</script>

<style scoped>
.layout-root {
  min-height: 100vh;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  color: #7dd3fc;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.topbar {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 0 20px;
}
</style>
