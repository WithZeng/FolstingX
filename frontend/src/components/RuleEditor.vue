<template>
  <n-modal v-model:show="showInner" preset="card" title="规则编辑器" style="width: 800px" :mask-closable="false">
    <n-steps :current="step" size="small" style="margin-bottom: 16px">
      <n-step title="基本信息" />
      <n-step title="入站配置" />
      <n-step title="出站配置" />
      <n-step title="高级设置" />
      <n-step title="确认保存" />
    </n-steps>

    <!-- Step 1: 基本信息 -->
    <div v-if="step === 1">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="规则名"><n-input v-model:value="form.name" /></n-form-item>
        <n-form-item label="转发模式">
          <n-select v-model:value="form.mode" :options="modeOptions" />
        </n-form-item>
        <n-form-item label="入站代理"><n-switch v-model:value="form.inbound_proxy_enabled" /></n-form-item>
        <n-form-item label="入站类型" v-if="form.inbound_proxy_enabled">
          <n-select v-model:value="form.inbound_type" :options="inboundTypeOptions" :disabled="lockInboundType" />
        </n-form-item>
      </n-form>
    </div>

    <!-- Step 2: 入站配置 - 监听节点使用下拉选择 -->
    <div v-else-if="step === 2">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="监听节点">
          <n-select
            v-model:value="form.listen_node_id"
            :options="listenNodeOptions"
            placeholder="选择一个节点作为监听入口"
            filterable
          />
        </n-form-item>
        <n-form-item label="监听端口"><n-input-number v-model:value="form.listen_port" :min="1" :max="65535" style="width: 100%" /></n-form-item>
        <n-form-item label="协议"><n-select v-model:value="form.protocol" :options="protocolOptions" /></n-form-item>
      </n-form>
      <n-alert type="info" :show-icon="true" style="margin-top: 8px">
        任何节点都可以作为监听入口，只要该节点包含 "entry" 角色即可。如果节点没有 entry 角色，这里不会显示。
      </n-alert>
    </div>

    <!-- Step 3: 出站配置 + LB 管理 -->
    <div v-else-if="step === 3">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="目标地址"><n-input v-model:value="form.target_address" placeholder="默认出站目标地址" /></n-form-item>
        <n-form-item label="目标端口"><n-input-number v-model:value="form.target_port" :min="1" :max="65535" style="width: 100%" /></n-form-item>
        <n-form-item label="负载均衡策略">
          <n-select v-model:value="form.lb_strategy" :options="lbOptions" />
        </n-form-item>
      </n-form>

      <n-divider />

      <!-- LB 策略说明 -->
      <n-alert type="info" :show-icon="true" style="margin-bottom: 12px">
        <strong>出站负载均衡(LB)策略说明：</strong><br/>
        <strong>RoundRobin</strong>: 依次轮询分配到各出站目标，流量均匀分布。<br/>
        <strong>WeightedRoundRobin</strong>: 按权重(weight)比例分配流量，权重越高分到的流量越多。<br/>
        <strong>Random</strong>: 随机选择一个健康的出站目标。<br/>
        <strong>LeastConn</strong>: 选择当前活跃连接数最少的目标，适合长连接场景。<br/>
        <strong>Failover</strong>: 始终使用第一个健康目标，只有当它故障时才切到下一个(主备模式)。<br/>
        <br/>
        当只填写上方"目标地址+端口"时为单目标直连，不启用负载均衡。<br/>
        添加下方 LB Targets 后生效。每个 Target 自动进行 TCP 健康检查(30秒一次)，连续失败3次标记为不健康。
      </n-alert>

      <n-card title="LB Targets (出站目标列表)" size="small">
        <template #header-extra>
          <n-button size="small" type="primary" @click="addLBTarget">添加目标</n-button>
        </template>
        <n-empty v-if="lbTargets.length === 0" description="暂无 LB 目标，流量将直接发送到上方目标地址" />
        <div v-for="(t, idx) in lbTargets" :key="idx" style="display:flex;gap:8px;align-items:center;margin-bottom:8px;">
          <n-input v-model:value="t.address" placeholder="地址" style="flex:2" />
          <n-input-number v-model:value="t.port" placeholder="端口" :min="1" :max="65535" style="flex:1" />
          <n-input-number v-model:value="t.weight" placeholder="权重" :min="1" :max="100" style="flex:1" />
          <n-checkbox v-model:checked="t.is_backup">备用</n-checkbox>
          <n-button size="small" type="error" @click="removeLBTarget(idx)">删除</n-button>
        </div>
      </n-card>
    </div>

    <!-- Step 4: 高级设置 -->
    <div v-else-if="step === 4">
      <n-form :model="form" label-placement="left" label-width="120">
        <n-form-item label="带宽限速(B/s)"><n-input-number v-model:value="form.bandwidth_limit" :min="0" style="width: 100%" /></n-form-item>
        <n-form-item label="启用状态"><n-switch v-model:value="form.is_active" /></n-form-item>
      </n-form>
    </div>

    <!-- Step 5: 确认 -->
    <div v-else>
      <n-alert type="info" :show-icon="false">
        将保存规则：{{ form.name }}，监听端口 {{ form.listen_port }}，目标 {{ form.target_address }}:{{ form.target_port }}
        <span v-if="lbTargets.length > 0">  (含 {{ lbTargets.length }} 个LB出站目标, 策略: {{ form.lb_strategy }})</span>
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
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useMessage } from "naive-ui";
import http from "@/api";

interface LBTargetItem {
  address: string;
  port: number;
  weight: number;
  is_backup: boolean;
}

interface NodeOption {
  label: string;
  value: number;
}

const props = defineProps<{ show: boolean; editingRule: any | null }>();
const emit = defineEmits<{ (e: "update:show", val: boolean): void; (e: "saved"): void }>();

const message = useMessage();
const step = ref(1);
const saving = ref(false);
const showInner = computed({
  get: () => props.show,
  set: (v: boolean) => emit("update:show", v),
});

// 节点列表，用于入站监听节点下拉
const allNodes = ref<any[]>([]);
const listenNodeOptions = computed<NodeOption[]>(() => {
  return allNodes.value
    .filter((n: any) => n.is_active && (n.roles || []).includes("entry"))
    .map((n: any) => ({
      label: `${n.name} (${n.host}) [${(n.roles || []).join("/")}]`,
      value: n.id,
    }));
});

const loadNodes = async () => {
  try {
    const { data } = await http.get("/nodes");
    allNodes.value = data;
  } catch {
    allNodes.value = [];
  }
};

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
  { label: "RoundRobin (轮询)", value: "round_robin" },
  { label: "WeightedRoundRobin (加权轮询)", value: "weighted_round_robin" },
  { label: "Random (随机)", value: "random" },
  { label: "LeastConn (最少连接)", value: "least_conn" },
  { label: "Failover (主备故障转移)", value: "failover" },
];

const form = reactive<any>({
  name: "",
  mode: "direct",
  listen_node_id: null as number | null,
  listen_port: 10000,
  protocol: "tcp",
  inbound_proxy_enabled: false,
  inbound_type: "vless_reality",
  target_address: "",
  target_port: 80,
  lb_strategy: "round_robin",
  lb_targets: [] as string[],
  bandwidth_limit: 0,
  is_active: true,
});

const lbTargets = reactive<LBTargetItem[]>([]);

const lockInboundType = computed(() => form.mode === "direct" && form.inbound_proxy_enabled);

const addLBTarget = () => {
  lbTargets.push({ address: "", port: 80, weight: 1, is_backup: false });
};
const removeLBTarget = (idx: number) => {
  lbTargets.splice(idx, 1);
};

// 同步 lbTargets 到 form.lb_targets (JSON string array)
const syncLBTargets = () => {
  form.lb_targets = lbTargets
    .filter((t) => t.address)
    .map((t) => JSON.stringify({ address: t.address, port: t.port, weight: t.weight, is_backup: t.is_backup }));
};

// 从 form.lb_targets 解析到 lbTargets
const parseLBTargets = () => {
  lbTargets.splice(0, lbTargets.length);
  if (form.lb_targets && form.lb_targets.length > 0) {
    for (const item of form.lb_targets) {
      try {
        const parsed = typeof item === "string" ? JSON.parse(item) : item;
        lbTargets.push({
          address: parsed.address || "",
          port: parsed.port || 80,
          weight: parsed.weight || 1,
          is_backup: parsed.is_backup || false,
        });
      } catch {
        // skip invalid
      }
    }
  }
};

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
        listen_node_id: null,
        listen_port: 10000,
        protocol: "tcp",
        inbound_proxy_enabled: false,
        inbound_type: "vless_reality",
        target_address: "",
        target_port: 80,
        lb_strategy: "round_robin",
        lb_targets: [],
        bandwidth_limit: 0,
        is_active: true,
      });
    }
    parseLBTargets();
  },
  { immediate: true },
);

watch(
  () => props.show,
  (v) => {
    if (v) loadNodes();
  },
);

const nextStep = () => {
  if (step.value < 5) step.value += 1;
};

const prevStep = () => {
  if (step.value > 1) step.value -= 1;
};

const saveRule = async () => {
  syncLBTargets();
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

onMounted(loadNodes);
</script>
