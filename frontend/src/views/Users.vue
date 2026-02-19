<template>
  <n-space vertical :size="16">
    <n-card>
      <n-space justify="space-between">
        <n-button type="primary" @click="openCreate">创建用户</n-button>
        <n-button @click="loadUsers">刷新</n-button>
      </n-space>
    </n-card>

    <n-card title="用户列表">
      <n-data-table :columns="columns" :data="users" :pagination="{ pageSize: 10 }" />
    </n-card>
  </n-space>

  <n-modal v-model:show="showModal" preset="card" :title="editing ? '编辑用户' : '创建用户'" style="width: 560px">
    <n-form :model="form" label-placement="left" label-width="120">
      <n-form-item label="用户名"><n-input v-model:value="form.username" /></n-form-item>
      <n-form-item v-if="!editing" label="密码"><n-input v-model:value="form.password" type="password" /></n-form-item>
      <n-form-item label="角色">
        <n-select v-model:value="form.role" :options="roleOptions" />
      </n-form-item>
      <n-form-item label="带宽限制"><n-input-number v-model:value="form.bandwidth_limit" :min="0" style="width: 100%" /></n-form-item>
      <n-form-item label="流量上限"><n-input-number v-model:value="form.traffic_limit" :min="0" style="width: 100%" /></n-form-item>
      <n-form-item label="过期时间"><n-date-picker v-model:value="expireAtMs" type="datetime" style="width: 100%" /></n-form-item>
      <n-form-item label="启用"><n-switch v-model:value="form.is_active" /></n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showModal = false">取消</n-button>
        <n-button type="primary" @click="saveUser">保存</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from "vue";
import { NButton, NPopconfirm, NTag, useMessage, type DataTableColumns } from "naive-ui";
import http from "@/api";

interface UserItem {
  id: number;
  username: string;
  role: string;
  traffic_used: number;
  traffic_limit: number;
  expire_at: string;
  is_active: boolean;
  bandwidth_limit: number;
}

const message = useMessage();
const users = ref<UserItem[]>([]);
const showModal = ref(false);
const editing = ref<UserItem | null>(null);

const form = reactive<any>({
  username: "",
  password: "",
  role: "user",
  bandwidth_limit: 0,
  traffic_limit: 0,
  expire_at: "",
  is_active: true,
});

const expireAtMs = computed({
  get: () => (form.expire_at ? new Date(form.expire_at).getTime() : null),
  set: (val: number | null) => {
    form.expire_at = val ? new Date(val).toISOString() : "";
  },
});

const roleOptions = [
  { label: "super_admin", value: "super_admin" },
  { label: "admin", value: "admin" },
  { label: "user", value: "user" },
];

const columns: DataTableColumns<UserItem> = [
  { title: "用户名", key: "username" },
  { title: "角色", key: "role_badge", render: (row) => h(NTag, { type: row.role === "super_admin" ? "error" : row.role === "admin" ? "warning" : "info" }, () => row.role) },
  { title: "流量", key: "traffic", render: (row) => `${(row.traffic_used / 1024 / 1024).toFixed(2)} / ${(row.traffic_limit / 1024 / 1024).toFixed(2)} MB` },
  { title: "过期时间", key: "expire_at" },
  { title: "状态", key: "status", render: (row) => h(NTag, { type: row.is_active ? "success" : "default" }, () => (row.is_active ? "启用" : "停用")) },
  {
    title: "操作",
    key: "actions",
    render: (row) =>
      h("div", { style: "display:flex;gap:8px;" }, [
        h(NButton, { size: "small", onClick: () => openEdit(row) }, { default: () => "编辑" }),
        h(NButton, { size: "small", type: "warning", onClick: () => resetUserTraffic(row.id) }, { default: () => "重置流量" }),
        h(
          NPopconfirm,
          { onPositiveClick: () => removeUser(row.id) },
          { trigger: () => h(NButton, { size: "small", type: "error" }, { default: () => "删除" }), default: () => "确认删除？" },
        ),
      ]),
  },
];

const loadUsers = async () => {
  const { data } = await http.get<UserItem[]>("/users");
  users.value = data;
};

const openCreate = () => {
  editing.value = null;
  Object.assign(form, { username: "", password: "", role: "user", bandwidth_limit: 0, traffic_limit: 0, expire_at: "", is_active: true });
  showModal.value = true;
};

const openEdit = (row: UserItem) => {
  editing.value = row;
  Object.assign(form, row);
  showModal.value = true;
};

const saveUser = async () => {
  if (editing.value) {
    await http.put(`/users/${editing.value.id}`, form);
    message.success("更新成功");
  } else {
    await http.post("/users", form);
    message.success("创建成功");
  }
  showModal.value = false;
  await loadUsers();
};

const resetUserTraffic = async (id: number) => {
  await http.post(`/users/${id}/reset-traffic`);
  message.success("流量已重置");
  await loadUsers();
};

const removeUser = async (id: number) => {
  await http.delete(`/users/${id}`);
  message.success("用户已删除");
  await loadUsers();
};

onMounted(loadUsers);
</script>
