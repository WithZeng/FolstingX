<template>
  <n-space vertical :size="16">
    <n-card>
      <n-space justify="space-between">
        <n-button type="primary" @click="openCreate">新增节点</n-button>
        <n-button @click="fetchNodes">刷新</n-button>
      </n-space>
    </n-card>

    <n-card title="节点列表">
      <n-data-table :columns="columns" :data="nodes" :loading="loading" :pagination="{ pageSize: 10 }" />
    </n-card>
  </n-space>

  <n-modal v-model:show="showModal" preset="card" :title="editingNode ? '编辑节点' : '新增节点'" style="width: 560px">
    <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="100">
      <n-form-item label="名称" path="name"><n-input v-model:value="form.name" /></n-form-item>
      <n-form-item label="地址" path="host"><n-input v-model:value="form.host" /></n-form-item>
      <n-form-item label="SSH端口" path="ssh_port"><n-input-number v-model:value="form.ssh_port" :min="1" :max="65535" style="width: 100%" /></n-form-item>
      <n-form-item label="SSH用户" path="ssh_user"><n-input v-model:value="form.ssh_user" /></n-form-item>
      <n-form-item label="SSH密钥" path="ssh_key"><n-input v-model:value="form.ssh_key" type="textarea" /></n-form-item>
      <n-form-item label="位置" path="location"><n-input v-model:value="form.location" /></n-form-item>
      <n-form-item label="类型" path="node_type">
        <n-select v-model:value="form.node_type" :options="nodeTypeOptions" />
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
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref } from "vue";
import { useMessage, type FormInst, type FormRules, type DataTableColumns, NTag, NButton, NPopconfirm } from "naive-ui";
import http from "@/api";

interface NodeItem {
  id: number;
  name: string;
  host: string;
  ssh_port: number;
  ssh_user: string;
  ssh_key: string;
  location: string;
  node_type: "entry" | "relay" | "exit";
  is_active: boolean;
  latency_ms: number;
}

const message = useMessage();
const loading = ref(false);
const submitting = ref(false);
const showModal = ref(false);
const formRef = ref<FormInst | null>(null);
const nodes = ref<NodeItem[]>([]);
const editingNode = ref<NodeItem | null>(null);

const nodeTypeOptions = [
  { label: "入口(entry)", value: "entry" },
  { label: "中继(relay)", value: "relay" },
  { label: "出口(exit)", value: "exit" },
];

const form = reactive<NodeItem>({
  id: 0,
  name: "",
  host: "",
  ssh_port: 22,
  ssh_user: "root",
  ssh_key: "",
  location: "",
  node_type: "relay",
  is_active: true,
  latency_ms: -1,
});

const rules: FormRules = {
  name: [{ required: true, message: "请输入名称", trigger: ["blur", "input"] }],
  host: [{ required: true, message: "请输入地址", trigger: ["blur", "input"] }],
  ssh_user: [{ required: true, message: "请输入 SSH 用户", trigger: ["blur", "input"] }],
};

const latencyColor = (latency: number): "success" | "warning" | "error" => {
  if (latency < 0) return "error";
  if (latency < 100) return "success";
  if (latency <= 300) return "warning";
  return "error";
};

const columns: DataTableColumns<NodeItem> = [
  { title: "名称", key: "name" },
  { title: "地址", key: "host" },
  { title: "类型", key: "node_type" },
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
    id: 0,
    name: "",
    host: "",
    ssh_port: 22,
    ssh_user: "root",
    ssh_key: "",
    location: "",
    node_type: "relay",
    is_active: true,
    latency_ms: -1,
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

const openCreate = () => {
  editingNode.value = null;
  resetForm();
  showModal.value = true;
};

const openEdit = (row: NodeItem) => {
  editingNode.value = row;
  Object.assign(form, row);
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

onMounted(fetchNodes);
</script>
