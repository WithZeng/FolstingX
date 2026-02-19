<template>
  <n-space vertical :size="16">
    <n-card>
      <n-space justify="space-between">
        <n-space>
          <n-button type="primary" @click="openCreate">新增节点</n-button>
          <n-button @click="showBatchImport = true">批量导入</n-button>
          <n-button @click="exportAll">批量导出</n-button>
          <n-button @click="showTextExport = true">文本导出</n-button>
        </n-space>
        <n-button @click="fetchNodes">刷新</n-button>
      </n-space>
    </n-card>

    <n-card title="节点列表">
      <n-data-table :columns="columns" :data="nodes" :loading="loading" :pagination="{ pageSize: 10 }" />
    </n-card>
  </n-space>

  <!-- 新增/编辑节点 -->
  <n-modal v-model:show="showModal" preset="card" :title="editingNode ? '编辑节点' : '新增节点'" style="width: 560px">
    <n-form ref="formRef" :model="form" :rules="formRules" label-placement="left" label-width="100">
      <n-form-item label="名称" path="name"><n-input v-model:value="form.name" /></n-form-item>
      <n-form-item label="地址" path="host"><n-input v-model:value="form.host" /></n-form-item>
      <n-form-item label="SSH端口" path="ssh_port"><n-input-number v-model:value="form.ssh_port" :min="1" :max="65535" style="width: 100%" /></n-form-item>
      <n-form-item label="SSH用户" path="ssh_user">
        <n-input v-model:value="form.ssh_user" placeholder="建议使用 folstingx (非root)" />
      </n-form-item>
      <n-form-item label="SSH密钥" path="ssh_key"><n-input v-model:value="form.ssh_key" type="textarea" /></n-form-item>
      <n-form-item label="位置" path="location"><n-input v-model:value="form.location" /></n-form-item>
      <n-form-item label="角色" path="roles">
        <n-checkbox-group v-model:value="form.roles">
          <n-space>
            <n-checkbox value="entry" label="入口(entry)" />
            <n-checkbox value="relay" label="中继(relay)" />
            <n-checkbox value="exit" label="出口(exit)" />
          </n-space>
        </n-checkbox-group>
      </n-form-item>
      <n-form-item label="启用" path="is_active"><n-switch v-model:value="form.is_active" /></n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showModal = false">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="submitForm">保存</n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 批量导入 -->
  <n-modal v-model:show="showBatchImport" preset="card" title="批量导入节点" style="width: 640px">
    <n-tabs type="segment">
      <n-tab-pane name="text" tab="文本导入">
        <n-alert type="info" :show-icon="false" style="margin-bottom: 12px">
          支持两种格式：<br/>
          1. JSON 数组: [{"name":"节点1","host":"1.2.3.4","ssh_port":22,"ssh_user":"folstingx","location":"HK","roles":["entry","relay","exit"]}]<br/>
          2. 管道分隔文本 (每行一条): name|host|ssh_port|ssh_user|location|roles(逗号分隔)
        </n-alert>
        <n-input v-model:value="importText" type="textarea" :rows="10" placeholder="粘贴节点数据..." />
        <n-space justify="end" style="margin-top: 12px">
          <n-button type="primary" @click="doTextImport">导入</n-button>
        </n-space>
      </n-tab-pane>
      <n-tab-pane name="file" tab="文件导入">
        <n-upload :default-upload="false" @change="onImportFile" accept=".json,.txt">
          <n-button>选择 JSON 文件</n-button>
        </n-upload>
      </n-tab-pane>
    </n-tabs>
  </n-modal>

  <!-- 文本导出 -->
  <n-modal v-model:show="showTextExport" preset="card" title="文本导出节点" style="width: 640px">
    <n-alert type="info" :show-icon="false" style="margin-bottom: 12px">
      可直接复制下面文本用于批量导入到其他系统
    </n-alert>
    <n-input :value="exportTextData" type="textarea" :rows="12" readonly />
    <n-space justify="end" style="margin-top: 12px">
      <n-button @click="copyExportText">复制</n-button>
      <n-button type="primary" @click="downloadExportText">下载</n-button>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref, watch } from "vue";
import { useMessage, type FormInst, type FormRules, type DataTableColumns, NTag, NButton, NPopconfirm, type UploadFileInfo } from "naive-ui";
import http from "@/api";

interface NodeItem {
  id: number;
  name: string;
  host: string;
  ssh_port: number;
  ssh_user: string;
  ssh_key: string;
  location: string;
  roles: string[];
  is_active: boolean;
  latency_ms: number;
}

const message = useMessage();
const loading = ref(false);
const submitting = ref(false);
const showModal = ref(false);
const showBatchImport = ref(false);
const showTextExport = ref(false);
const formRef = ref<FormInst | null>(null);
const nodes = ref<NodeItem[]>([]);
const editingNode = ref<NodeItem | null>(null);
const importText = ref("");
const exportTextData = ref("");

const form = reactive<NodeItem>({
  id: 0,
  name: "",
  host: "",
  ssh_port: 22,
  ssh_user: "folstingx",
  ssh_key: "",
  location: "",
  roles: ["entry", "relay", "exit"],
  is_active: true,
  latency_ms: -1,
});

const formRules: FormRules = {
  name: [{ required: true, message: "请输入名称", trigger: ["blur", "input"] }],
  host: [{ required: true, message: "请输入地址", trigger: ["blur", "input"] }],
  ssh_user: [{ required: true, message: "请输入 SSH 用户", trigger: ["blur", "input"] }],
};

const roleTagMap: Record<string, string> = { entry: "入口", relay: "中继", exit: "出口" };

const latencyColor = (latency: number): "success" | "warning" | "error" => {
  if (latency < 0) return "error";
  if (latency < 100) return "success";
  if (latency <= 300) return "warning";
  return "error";
};

const columns: DataTableColumns<NodeItem> = [
  { title: "名称", key: "name" },
  { title: "地址", key: "host" },
  {
    title: "角色",
    key: "roles",
    render: (row) =>
      h("div", { style: "display:flex;gap:4px;flex-wrap:wrap;" },
        (row.roles || []).map((r: string) => h(NTag, { size: "small", type: "info" }, () => roleTagMap[r] || r))
      ),
  },
  { title: "位置", key: "location" },
  {
    title: "延迟",
    key: "latency_ms",
    render: (row) => h(NTag, { type: latencyColor(row.latency_ms) }, () => (row.latency_ms < 0 ? "离线" : `${row.latency_ms} ms`)),
  },
  {
    title: "状态",
    key: "is_active",
    render: (row) => h(NTag, { type: row.is_active ? "success" : "default" }, () => (row.is_active ? "启用" : "停用")),
  },
  {
    title: "操作",
    key: "actions",
    render: (row) =>
      h("div", { style: "display:flex;gap:8px;" }, [
        h(NButton, { size: "small", onClick: () => openEdit(row) }, { default: () => "编辑" }),
        h(NButton, { size: "small", type: "warning", onClick: () => checkNode(row.id) }, { default: () => "检查" }),
        h(
          NPopconfirm,
          { onPositiveClick: () => removeNode(row.id) },
          {
            trigger: () => h(NButton, { size: "small", type: "error" }, { default: () => "删除" }),
            default: () => "确认删除该节点？",
          },
        ),
      ]),
  },
];

const resetForm = () => {
  Object.assign(form, {
    id: 0, name: "", host: "", ssh_port: 22, ssh_user: "folstingx",
    ssh_key: "", location: "", roles: ["entry", "relay", "exit"], is_active: true, latency_ms: -1,
  });
};

const fetchNodes = async () => {
  loading.value = true;
  try {
    const { data } = await http.get<NodeItem[]>("/nodes");
    nodes.value = data;
  } catch (error: any) {
    message.error(error?.response?.data?.error || "获取节点失败");
  } finally {
    loading.value = false;
  }
};

const openCreate = () => { editingNode.value = null; resetForm(); showModal.value = true; };
const openEdit = (row: NodeItem) => {
  editingNode.value = row;
  Object.assign(form, { ...row, roles: row.roles || ["entry", "relay", "exit"] });
  showModal.value = true;
};

const submitForm = async () => {
  await formRef.value?.validate();
  submitting.value = true;
  try {
    if (editingNode.value) {
      await http.put(`/nodes/${editingNode.value.id}`, form);
      message.success("节点更新成功");
    } else {
      await http.post("/nodes", form);
      message.success("节点创建成功");
    }
    showModal.value = false;
    await fetchNodes();
  } catch (error: any) {
    message.error(error?.response?.data?.error || "保存失败");
  } finally {
    submitting.value = false;
  }
};

const checkNode = async (id: number) => {
  try {
    await http.post(`/nodes/${id}/check`);
    message.success("健康检查已完成");
    await fetchNodes();
  } catch (error: any) {
    message.error(error?.response?.data?.error || "健康检查失败");
  }
};

const removeNode = async (id: number) => {
  try {
    await http.delete(`/nodes/${id}`);
    message.success("删除成功");
    await fetchNodes();
  } catch (error: any) {
    message.error(error?.response?.data?.error || "删除失败");
  }
};

// ========= 批量导入 =========
const doTextImport = async () => {
  const text = importText.value.trim();
  if (!text) { message.warning("请输入数据"); return; }
  if (text.startsWith("[")) {
    try {
      const arr = JSON.parse(text);
      const { data } = await http.post("/nodes/import-text", arr);
      message.success(`成功导入 ${data.imported} 个节点`);
      showBatchImport.value = false; importText.value = ""; await fetchNodes();
    } catch (e: any) { message.error("JSON 解析失败: " + e.message); }
    return;
  }
  const lines = text.split("\n").filter((l: string) => l.trim() && !l.startsWith("#"));
  const arr = lines.map((line: string) => {
    const p = line.split("|");
    return {
      name: p[0]?.trim() || "", host: p[1]?.trim() || "",
      ssh_port: parseInt(p[2]?.trim() ?? "", 10) || 22, ssh_user: p[3]?.trim() || "folstingx",
      location: p[4]?.trim() || "", roles: (p[5]?.trim() || "entry,relay,exit").split(",").map((s: string) => s.trim()),
    };
  }).filter((n: any) => n.host);
  try {
    const { data } = await http.post("/nodes/import-text", arr);
    message.success(`成功导入 ${data.imported} 个节点`);
    showBatchImport.value = false; importText.value = ""; await fetchNodes();
  } catch (e: any) { message.error("导入失败: " + (e?.response?.data?.error || e.message)); }
};

const onImportFile = async (options: { file: UploadFileInfo }) => {
  const file = options.file.file;
  if (!file) return;
  const formData = new FormData();
  formData.append("file", file);
  try {
    const { data } = await http.post("/nodes/import", formData, { headers: { "Content-Type": "multipart/form-data" } });
    message.success(`导入成功 ${data.imported} 个节点`);
    showBatchImport.value = false; await fetchNodes();
  } catch (e: any) { message.error("导入失败: " + (e?.response?.data?.error || e.message)); }
};

// ========= 批量导出 =========
const exportAll = async () => {
  try {
    const { data } = await http.get("/nodes/export?format=json");
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = "nodes-export.json"; a.click();
    URL.revokeObjectURL(url); message.success("导出成功");
  } catch { message.error("导出失败"); }
};

const loadExportText = async () => {
  try { const { data } = await http.get("/nodes/export-text"); exportTextData.value = data; }
  catch { exportTextData.value = "加载失败"; }
};

const copyExportText = () => { navigator.clipboard.writeText(exportTextData.value); message.success("已复制到剪贴板"); };

const downloadExportText = () => {
  const blob = new Blob([exportTextData.value], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a"); a.href = url; a.download = "nodes-export.txt"; a.click();
  URL.revokeObjectURL(url);
};

watch(() => showTextExport.value, (v) => { if (v) loadExportText(); });

onMounted(fetchNodes);
</script>
