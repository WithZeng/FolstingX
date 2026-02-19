<template>
  <n-card>
    <n-space justify="space-between" style="margin-bottom: 16px">
      <n-input v-model:value="keyword" placeholder="搜索规则名/目标" style="max-width: 320px" clearable />
      <n-space>
        <n-button @click="showImportModal = true">导入</n-button>
        <n-button @click="exportSelected" :disabled="selectedKeys.length === 0">批量导出</n-button>
        <n-button type="primary" @click="openCreate">新建规则</n-button>
      </n-space>
    </n-space>

    <n-data-table
      :columns="columns"
      :data="filteredRules"
      :row-key="(row:any)=>row.id"
      v-model:checked-row-keys="selectedKeys"
      :pagination="{ pageSize: 10 }"
    />
  </n-card>

  <rule-editor v-model:show="showEditor" :editing-rule="editingRule" @saved="reload" />

  <!-- 导入弹窗: 支持文件和文本 -->
  <n-modal v-model:show="showImportModal" preset="card" title="导入规则" style="width: 640px">
    <n-tabs type="segment">
      <n-tab-pane name="text" tab="文本导入">
        <n-alert type="info" :show-icon="false" style="margin-bottom: 12px">
          粘贴 JSON 数组格式的规则数据
        </n-alert>
        <n-input v-model:value="importText" type="textarea" :rows="10" placeholder="粘贴 JSON 规则数组..." />
        <n-space justify="end" style="margin-top: 12px">
          <n-button type="primary" @click="doTextImport">导入</n-button>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="file" tab="文件导入">
        <n-upload :default-upload="false" @change="onImportFile" accept=".json">
          <n-button>选择 JSON 文件</n-button>
        </n-upload>
      </n-tab-pane>
    </n-tabs>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { NButton, NTag, NPopconfirm, useMessage, type DataTableColumns, type UploadFileInfo } from "naive-ui";
import http from "@/api";
import RuleEditor from "@/components/RuleEditor.vue";

interface RuleItem {
  id: number;
  name: string;
  mode: string;
  listen_port: number;
  target_address: string;
  target_port: number;
  protocol: string;
  inbound_type: string;
  is_active: boolean;
  connections: number;
  traffic_up: number;
  traffic_down: number;
}

const message = useMessage();
const rules = ref<RuleItem[]>([]);
const showEditor = ref(false);
const showImportModal = ref(false);
const editingRule = ref<RuleItem | null>(null);
const selectedKeys = ref<number[]>([]);
const keyword = ref("");
const importText = ref("");

const filteredRules = computed(() => {
  if (!keyword.value.trim()) return rules.value;
  const k = keyword.value.trim().toLowerCase();
  return rules.value.filter((r) => `${r.name} ${r.target_address}:${r.target_port}`.toLowerCase().includes(k));
});

const columns: DataTableColumns<RuleItem> = [
  { type: "selection" },
  { title: "名称", key: "name" },
  { title: "模式", key: "mode" },
  { title: "端口", key: "listen_port" },
  { title: "目标", key: "target", render: (row) => `${row.target_address}:${row.target_port}` },
  { title: "协议", key: "protocol" },
  { title: "入站", key: "inbound_type" },
  {
    title: "状态",
    key: "status",
    render: (row) => h(NTag, { type: row.is_active ? "success" : "default" }, () => (row.is_active ? "运行" : "停止")),
  },
  { title: "连接数", key: "connections" },
  {
    title: "今日流量",
    key: "traffic_today",
    render: (row) => `${((row.traffic_up + row.traffic_down) / 1024 / 1024).toFixed(2)} MB`,
  },
  {
    title: "操作",
    key: "actions",
    render: (row) =>
      h("div", { style: "display:flex;gap:8px" }, [
        h(NButton, { size: "small", onClick: () => openEdit(row) }, { default: () => "编辑" }),
        h(NButton, { size: "small", onClick: () => toggleRule(row) }, { default: () => (row.is_active ? "停用" : "启用") }),
        h(
          NPopconfirm,
          { onPositiveClick: () => removeRule(row.id) },
          {
            trigger: () => h(NButton, { size: "small", type: "error" }, { default: () => "删除" }),
            default: () => "确认删除该规则？",
          },
        ),
      ]),
  },
];

const reload = async () => {
  const { data } = await http.get<RuleItem[]>("/rules");
  rules.value = data;
};

const openCreate = () => { editingRule.value = null; showEditor.value = true; };
const openEdit = (row: RuleItem) => { editingRule.value = row; showEditor.value = true; };

const removeRule = async (id: number) => {
  await http.delete(`/rules/${id}`);
  message.success("规则已删除");
  await reload();
};

const toggleRule = async (row: RuleItem) => {
  await http.put(`/rules/${row.id}/${row.is_active ? "disable" : "enable"}`);
  message.success("状态已更新");
  await reload();
};

const exportSelected = async () => {
  const { data } = await http.get(`/rules/export?format=json&ids=${selectedKeys.value.join(",")}`);
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url; a.download = "rules-export.json"; a.click();
  URL.revokeObjectURL(url);
};

const onImportFile = async (options: { file: UploadFileInfo }) => {
  const file = options.file.file;
  if (!file) return;
  const formData = new FormData();
  formData.append("file", file);
  formData.append("conflict", "rename");
  await http.post("/rules/import", formData, { headers: { "Content-Type": "multipart/form-data" } });
  message.success("导入成功");
  showImportModal.value = false;
  await reload();
};

const doTextImport = async () => {
  const text = importText.value.trim();
  if (!text) { message.warning("请输入数据"); return; }
  try {
    const arr = JSON.parse(text);
    const { data } = await http.post("/rules/import-text", arr);
    message.success(`成功导入 ${data.imported} 条规则`);
    showImportModal.value = false;
    importText.value = "";
    await reload();
  } catch (e: any) {
    message.error("导入失败: " + (e?.response?.data?.error || e.message));
  }
};

onMounted(reload);
</script>
