<template>
  <n-modal v-model:show="showInner" preset="card" title="规则编辑器" style="width: 760px" :mask-closable="false">
    <n-steps :current="step" size="small" style="margin-bottom: 16px">
      <n-step title="基本信息" />
      <n-step title="入站配置" />
      <n-step title="出站配置" />
      <n-step title="高级设置" />
      <n-step title="确认保存" />
    </n-steps>

    <div v-if="step === 1">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="规则名"><n-input v-model:value="form.name" /></n-form-item>
        <n-form-item label="转发模式">
          <n-select v-model:value="form.mode" :options="modeOptions" />
        </n-form-item>
        <n-form-item label="入站代理"><n-switch v-model:value="form.inbound_proxy_enabled" /></n-form-item>
        <n-form-item label="入站类型">
          <n-select v-model:value="form.inbound_type" :options="inboundTypeOptions" :disabled="lockInboundType" />
        </n-form-item>
      </n-form>
    </div>

    <div v-else-if="step === 2">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="监听节点ID"><n-input-number v-model:value="form.listen_node_id" :min="1" style="width: 100%" /></n-form-item>
        <n-form-item label="监听端口"><n-input-number v-model:value="form.listen_port" :min="1" :max="65535" style="width: 100%" /></n-form-item>
        <n-form-item label="协议"><n-select v-model:value="form.protocol" :options="protocolOptions" /></n-form-item>
      </n-form>
    </div>

    <div v-else-if="step === 3">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="目标地址"><n-input v-model:value="form.target_address" /></n-form-item>
        <n-form-item label="目标端口"><n-input-number v-model:value="form.target_port" :min="1" :max="65535" style="width: 100%" /></n-form-item>
        <n-form-item label="LB策略"><n-select v-model:value="form.lb_strategy" :options="lbOptions" /></n-form-item>
      </n-form>
    </div>

    <div v-else-if="step === 4">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="带宽限速(B/s)"><n-input-number v-model:value="form.bandwidth_limit" :min="0" style="width: 100%" /></n-form-item>
        <n-form-item label="启用状态"><n-switch v-model:value="form.is_active" /></n-form-item>
      </n-form>
    </div>

    <div v-else>
      <n-alert type="info" :show-icon="false">
        将保存规则：{{ form.name }}，监听端口 {{ form.listen_port }}，目标 {{ form.target_address }}:{{ form.target_port }}
      </n-alert>
      <n-alert v-if="form.inbound_proxy_enabled && form.inbound_type === 'vless_reality'" type="success" style="margin-top: 10px">
        保存后可在后端生成 vless:// 分享链接。
      </n-alert>
    </div>

    <template #footer>
      <n-space justify="space-between">
        <n-button @click="prevStep" :disabled="step === 1">上一步</n-button>
        <n-space>
          <n-button @click="showInner = false">取消</n-button>
          <n-button v-if="step < 5" type="primary" @click="nextStep">下一步</n-button>
          <n-button v-else type="primary" :loading="saving" @click="saveRule">保存</n-button>
        </n-space>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useMessage } from "naive-ui";
import http from "@/api";

const props = defineProps<{ show: boolean; editingRule: any | null }>();
const emit = defineEmits<{ (e: "update:show", val: boolean): void; (e: "saved"): void }>();

const message = useMessage();
const step = ref(1);
const saving = ref(false);
const showInner = computed({
  get: () => props.show,
  set: (v: boolean) => emit("update:show", v),
});

const modeOptions = [
  { label: "海外直连", value: "direct" },
  { label: "中转", value: "relay" },
  { label: "IX", value: "ix" },
  { label: "链式", value: "chain" },
];
const protocolOptions = [
  { label: "TCP", value: "tcp" },
  { label: "UDP", value: "udp" },
  { label: "TCP+UDP", value: "both" },
];
const inboundTypeOptions = [
  { label: "VLESS Reality", value: "vless_reality" },
  { label: "Shadowsocks", value: "shadowsocks" },
];
const lbOptions = [
  { label: "RoundRobin", value: "round_robin" },
  { label: "WeightedRoundRobin", value: "weighted_round_robin" },
  { label: "Random", value: "random" },
  { label: "LeastConn", value: "least_conn" },
  { label: "Failover", value: "failover" },
];

const form = reactive<any>({
  name: "",
  mode: "direct",
  listen_node_id: 1,
  listen_port: 10000,
  protocol: "tcp",
  inbound_proxy_enabled: false,
  inbound_type: "vless_reality",
  target_address: "127.0.0.1",
  target_port: 80,
  lb_strategy: "round_robin",
  lb_targets: [],
  bandwidth_limit: 0,
  is_active: true,
});

const lockInboundType = computed(() => form.mode === "direct" && form.inbound_proxy_enabled);

watch(
  () => [form.mode, form.inbound_proxy_enabled],
  () => {
    if (form.mode === "direct" && form.inbound_proxy_enabled) {
      form.inbound_type = "vless_reality";
    }
    if (form.mode !== "direct" && !form.inbound_proxy_enabled) {
      form.inbound_type = "";
    }
  },
  { immediate: true },
);

watch(
  () => props.editingRule,
  (val) => {
    step.value = 1;
    if (val) {
      Object.assign(form, val);
    } else {
      Object.assign(form, {
        name: "",
        mode: "direct",
        listen_node_id: 1,
        listen_port: 10000,
        protocol: "tcp",
        inbound_proxy_enabled: false,
        inbound_type: "vless_reality",
        target_address: "127.0.0.1",
        target_port: 80,
        lb_strategy: "round_robin",
        lb_targets: [],
        bandwidth_limit: 0,
        is_active: true,
      });
    }
  },
  { immediate: true },
);

const nextStep = () => {
  if (step.value < 5) step.value += 1;
};

const prevStep = () => {
  if (step.value > 1) step.value -= 1;
};

const saveRule = async () => {
  saving.value = true;
  try {
    if (props.editingRule?.id) {
      await http.put(`/rules/${props.editingRule.id}`, form);
    } else {
      await http.post("/rules", form);
    }
    message.success("规则保存成功");
    showInner.value = false;
    emit("saved");
  } catch (error: any) {
    message.error(error?.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};
</script>
