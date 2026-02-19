<template>
  <div class="page">
    <n-h1>FolstingX 登录</n-h1>
    <n-card style="max-width: 420px; width: 100%">
      <n-form ref="formRef" :model="form" :rules="rules" @submit.prevent="onSubmit">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="form.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="请输入密码" />
        </n-form-item>
        <n-button type="primary" block :loading="loading" attr-type="submit">登录</n-button>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { useMessage, type FormInst, type FormRules } from "naive-ui";
import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const message = useMessage();
const authStore = useAuthStore();

const formRef = ref<FormInst | null>(null);
const loading = ref(false);

const form = reactive({
  username: "admin",
  password: "admin123",
});

const rules: FormRules = {
  username: [{ required: true, message: "请输入用户名", trigger: ["blur", "input"] }],
  password: [{ required: true, message: "请输入密码", trigger: ["blur", "input"] }],
};

const onSubmit = async () => {
  await formRef.value?.validate();
  try {
    loading.value = true;
    await authStore.login(form.username, form.password);
    message.success("登录成功");
    await router.push("/dashboard");
  } catch (error: any) {
    const errMsg = error?.response?.data?.error || "登录失败";
    message.error(errMsg);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 16px;
  padding: 16px;
}
</style>
