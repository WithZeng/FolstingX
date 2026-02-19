<template>
  <n-space vertical :size="16">
    <n-card>
      <n-space>
        <n-input v-model:value="query.level" placeholder="level" style="width: 120px" />
        <n-input v-model:value="query.module" placeholder="module" style="width: 140px" />
        <n-button @click="loadLogs">筛选</n-button>
        <n-button type="error" @click="clearLogs">清空日志</n-button>
      </n-space>
    </n-card>

    <n-card title="日志列表">
      <n-data-table :columns="columns" :data="rows" :pagination="{ pageSize: 20 }" />
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, h } from "vue";
import { NTag, useMessage, type DataTableColumns } from "naive-ui";
import http from "@/api";

interface LogItem {
  id: number;
  level: string;
  module: string;
  message: string;
  created_at: string;
}

const message = useMessage();
const rows = ref<LogItem[]>([]);
const query = reactive({ level: "", module: "" });

const columns: DataTableColumns<LogItem> = [
  { title: "时间", key: "created_at" },
  { title: "级别", key: "level_badge", render: (row) => h(NTag, { type: row.level === "error" ? "error" : row.level === "warn" ? "warning" : "info" }, () => row.level) },
  { title: "模块", key: "module" },
  { title: "内容", key: "message" },
];

const loadLogs = async () => {
  const { data } = await http.get("/logs", { params: query });
  rows.value = data.items || [];
};

const clearLogs = async () => {
  await http.delete("/logs");
  message.success("日志已清空");
  await loadLogs();
};

onMounted(loadLogs);
</script>
