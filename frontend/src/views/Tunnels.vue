<template>
  <n-space vertical :size="16">
    <n-card>
      <n-space justify="space-between">
        <n-space>
          <n-button type="primary" @click="openCreate">新建隧道</n-button>
        </n-space>
        <n-button @click="fetchTunnels">刷新</n-button>
      </n-space>
    </n-card>

    <n-card title="隧道列表">
      <n-data-table :columns="columns" :data="tunnels" :loading="loading" :pagination="{ pageSize: 10 }" :row-key="(row:any)=>row.id" />
    </n-card>
  </n-space>

  <!-- 新建/编辑隧道 -->
  <n-modal v-model:show="showTunnelModal" preset="card" :title="editingTunnel ? '编辑隧道' : '新建隧道'" style="width: 560px">
    <n-form :model="tunnelForm" label-placement="left" label-width="100">
      <n-form-item label="隧道名称"><n-input v-model:value="tunnelForm.name" /></n-form-item>
      <n-form-item label="隧道类型">
        <n-select v-model:value="tunnelForm.type" :options="typeOptions" />
      </n-form-item>
      <n-form-item label="流量倍率"><n-input-number v-model:value="tunnelForm.traffic_ratio" :min="0.1" :max="100" :step="0.1" style="width: 100%" /></n-form-item>
      <n-form-item label="入口IP限制"><n-input v-model:value="tunnelForm.inbound_ip" placeholder="留空不限制" /></n-form-item>
      <n-form-item label="启用"><n-switch v-model:value="tunnelForm.is_active" /></n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showTunnelModal = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="saveTunnel">保存</n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 隧道详情 (链路管理) -->
  <n-modal v-model:show="showDetailModal" preset="card" :title="`隧道详情: ${detailTunnel?.name || ''}`" style="width: 900px">
    <n-tabs type="segment">
      <!-- 链路节点 Tab -->
      <n-tab-pane name="chain" tab="链路节点">
        <n-alert type="info" :show-icon="true" style="margin-bottom: 12px">
          <strong>链路说明：</strong>流量路径按 <strong>入口(Entry) → 中继(Relay) → 出口(Exit)</strong> 顺序经过各节点。<br/>
          端口转发(Type=1) 只需入口节点；链式中转(Type=2) 需要完整链路。<br/>
          <strong>协议说明：</strong> relay=标准中继, ws=WebSocket, wss=WebSocket+TLS, mws=多路复用WS, mwss=多路复用WSS
        </n-alert>

        <n-space style="margin-bottom: 12px">
          <n-button size="small" type="primary" @click="showAddChain = true">添加链路节点</n-button>
        </n-space>

        <n-data-table :columns="chainColumns" :data="detailChains" :row-key="(row:any)=>row.id" size="small" />

        <!-- 链路可视化 -->
        <n-card title="链路拓扑" size="small" style="margin-top: 12px" v-if="detailChains.length > 0">
          <div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap;padding:8px 0;">
            <template v-for="(chain, idx) in sortedChains" :key="chain.id">
              <n-tag :type="chainTagType(chain.chain_type)" round>
                {{ chainTypeName(chain.chain_type) }}: {{ chain.node?.name || `Node#${chain.node_id}` }}
                <template v-if="chain.port"> :{{ chain.port }}</template>
              </n-tag>
              <span v-if="idx < sortedChains.length - 1" style="font-size:20px;color:#999">→</span>
            </template>
          </div>
        </n-card>
      </n-tab-pane>

      <!-- 转发规则 Tab -->
      <n-tab-pane name="forwards" tab="转发规则">
        <n-space style="margin-bottom: 12px">
          <n-button size="small" type="primary" @click="showAddForward = true">添加转发</n-button>
        </n-space>
        <n-data-table :columns="forwardColumns" :data="detailForwards" :row-key="(row:any)=>row.id" size="small" />
      </n-tab-pane>

      <!-- 部署操作 Tab -->
      <n-tab-pane name="deploy" tab="部署操作">
        <n-space vertical :size="12">
          <n-alert type="warning">
            部署操作将通过 WebSocket 向所有链路节点的 gost Agent 下发配置。确保节点已在线。
          </n-alert>
          <n-space>
            <n-button type="primary" @click="deployTunnel" :loading="deploying">部署到节点</n-button>
            <n-button type="error" @click="undeployTunnel" :loading="deploying">取消部署</n-button>
          </n-space>
          <div v-if="deployResult">
            <n-alert :type="deployResult.errors ? 'warning' : 'success'" style="margin-top:8px">
              {{ deployResult.message }}
              <div v-if="deployResult.errors" style="margin-top:4px">
                <div v-for="(e, i) in deployResult.errors" :key="i" style="color:#d03050">{{ e }}</div>
              </div>
            </n-alert>
          </div>
        </n-space>
      </n-tab-pane>
    </n-tabs>
  </n-modal>

  <!-- 添加链路节点 -->
  <n-modal v-model:show="showAddChain" preset="card" title="添加链路节点" style="width: 480px">
    <n-form :model="chainForm" label-placement="left" label-width="80">
      <n-form-item label="节点">
        <n-select v-model:value="chainForm.node_id" :options="nodeOptions" filterable placeholder="选择节点" />
      </n-form-item>
      <n-form-item label="角色">
        <n-select v-model:value="chainForm.chain_type" :options="chainTypeOptions" />
      </n-form-item>
      <n-form-item label="端口"><n-input-number v-model:value="chainForm.port" :min="1" :max="65535" style="width: 100%" placeholder="节点端口" /></n-form-item>
      <n-form-item label="协议">
        <n-select v-model:value="chainForm.protocol" :options="protocolOptions" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showAddChain = false">取消</n-button>
        <n-button type="primary" @click="addChainNode">确定</n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 添加转发 -->
  <n-modal v-model:show="showAddForward" preset="card" title="添加转发规则" style="width: 560px">
    <n-form :model="forwardForm" label-placement="left" label-width="100">
      <n-form-item label="名称"><n-input v-model:value="forwardForm.name" /></n-form-item>
      <n-form-item label="远程地址"><n-input v-model:value="forwardForm.remote_address" placeholder="host:port (目标地址)" /></n-form-item>
      <n-form-item label="监听端口"><n-input-number v-model:value="forwardForm.listen_port" :min="1" :max="65535" style="width: 100%" /></n-form-item>
      <n-form-item label="协议">
        <n-select v-model:value="forwardForm.protocol" :options="[{label:'TCP',value:'tcp'},{label:'UDP',value:'udp'},{label:'TCP+UDP',value:'both'}]" />
      </n-form-item>
      <n-divider />
      <n-form-item label="入站代理"><n-switch v-model:value="forwardForm.inbound_enabled" /></n-form-item>
      <n-form-item label="入站类型" v-if="forwardForm.inbound_enabled">
        <n-select v-model:value="forwardForm.inbound_type" :options="inboundTypeOptions" />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="showAddForward = false">取消</n-button>
        <n-button type="primary" @click="addForward">确定</n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 节点安装命令 -->
  <n-modal v-model:show="showInstallCmd" preset="card" title="节点安装命令" style="width: 640px">
    <n-alert type="info" :show-icon="true" style="margin-bottom: 12px">
      在目标节点服务器上执行以下命令即可自动安装 gost Agent 并连接到面板：
    </n-alert>
    <n-input :value="installCmdText" type="textarea" :rows="3" readonly style="font-family: monospace" />
    <n-space justify="end" style="margin-top: 12px">
      <n-button @click="copyInstallCmd">复制命令</n-button>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from "vue";
import { NButton, NTag, NPopconfirm, useMessage, type DataTableColumns } from "naive-ui";
import http from "@/api";

interface TunnelItem {
  id: number;
  name: string;
  type: number;
  traffic_ratio: number;
  inbound_ip: string;
  is_active: boolean;
  flow_in: number;
  flow_out: number;
  chain_tunnels?: ChainItem[];
  forwards?: ForwardItem[];
}

interface ChainItem {
  id: number;
  tunnel_id: number;
  chain_type: number;
  node_id: number;
  port: number;
  protocol: string;
  sort_index: number;
  node?: { id: number; name: string; host: string; is_online: boolean };
}

interface ForwardItem {
  id: number;
  tunnel_id: number;
  name: string;
  remote_address: string;
  listen_port: number;
  protocol: string;
  is_active: boolean;
  inbound_enabled: boolean;
  inbound_type: string;
  flow_in: number;
  flow_out: number;
}

const message = useMessage();
const loading = ref(false);
const saving = ref(false);
const deploying = ref(false);

const tunnels = ref<TunnelItem[]>([]);
const allNodes = ref<any[]>([]);

const showTunnelModal = ref(false);
const showDetailModal = ref(false);
const showAddChain = ref(false);
const showAddForward = ref(false);
const showInstallCmd = ref(false);

const editingTunnel = ref<TunnelItem | null>(null);
const detailTunnel = ref<TunnelItem | null>(null);
const detailChains = ref<ChainItem[]>([]);
const detailForwards = ref<ForwardItem[]>([]);
const deployResult = ref<any>(null);
const installCmdText = ref("");

const tunnelForm = reactive({
  name: "",
  type: 1,
  traffic_ratio: 1.0,
  inbound_ip: "",
  is_active: true,
});

const chainForm = reactive({
  node_id: null as number | null,
  chain_type: 1,
  port: 10000,
  protocol: "relay",
});

const forwardForm = reactive({
  name: "",
  remote_address: "",
  listen_port: 10000,
  protocol: "tcp",
  inbound_enabled: false,
  inbound_type: "vless_reality",
});

const typeOptions = [
  { label: "端口转发 (直连)", value: 1 },
  { label: "链式中转 (多跳)", value: 2 },
];

const chainTypeOptions = [
  { label: "入口 (Entry)", value: 1 },
  { label: "中继 (Relay)", value: 2 },
  { label: "出口 (Exit)", value: 3 },
];

const protocolOptions = [
  { label: "relay (标准中继)", value: "relay" },
  { label: "ws (WebSocket)", value: "ws" },
  { label: "wss (WebSocket+TLS)", value: "wss" },
  { label: "mws (多路复用WS)", value: "mws" },
  { label: "mwss (多路复用WSS)", value: "mwss" },
  { label: "tcp", value: "tcp" },
];

const inboundTypeOptions = [
  { label: "VLESS Reality", value: "vless_reality" },
  { label: "Shadowsocks", value: "shadowsocks" },
  { label: "Trojan", value: "trojan" },
];

const nodeOptions = computed(() =>
  allNodes.value.map((n: any) => ({
    label: `${n.name} (${n.host}) ${n.is_online ? '🟢' : '⚪'}`,
    value: n.id,
  }))
);

const sortedChains = computed(() =>
  [...detailChains.value].sort((a, b) => a.chain_type - b.chain_type || a.sort_index - b.sort_index)
);

const chainTypeName = (ct: number) => {
  switch (ct) {
    case 1: return "入口";
    case 2: return "中继";
    case 3: return "出口";
    default: return "未知";
  }
};

const chainTagType = (ct: number): "success" | "warning" | "info" => {
  switch (ct) {
    case 1: return "success";
    case 2: return "warning";
    case 3: return "info";
    default: return "info";
  }
};

// ========== Table Columns ==========
const columns: DataTableColumns<TunnelItem> = [
  { title: "ID", key: "id", width: 60 },
  { title: "名称", key: "name" },
  {
    title: "类型", key: "type",
    render: (row) => h(NTag, { type: row.type === 1 ? "info" : "warning", size: "small" }, () => row.type === 1 ? "端口转发" : "链式中转"),
  },
  { title: "倍率", key: "traffic_ratio" },
  {
    title: "链路", key: "chains",
    render: (row) => `${(row.chain_tunnels || []).length} 节点`,
  },
  {
    title: "状态", key: "is_active",
    render: (row) => h(NTag, { type: row.is_active ? "success" : "default", size: "small" }, () => row.is_active ? "启用" : "停用"),
  },
  {
    title: "流量", key: "flow",
    render: (row) => `↑${formatBytes(row.flow_in)} ↓${formatBytes(row.flow_out)}`,
  },
  {
    title: "操作", key: "actions",
    render: (row) => h("div", { style: "display:flex;gap:6px" }, [
      h(NButton, { size: "small", type: "info", onClick: () => openDetail(row) }, { default: () => "详情" }),
      h(NButton, { size: "small", onClick: () => openEdit(row) }, { default: () => "编辑" }),
      h(NButton, { size: "small", onClick: () => toggleTunnel(row) }, { default: () => row.is_active ? "停用" : "启用" }),
      h(NPopconfirm, { onPositiveClick: () => removeTunnel(row.id) }, {
        trigger: () => h(NButton, { size: "small", type: "error" }, { default: () => "删除" }),
        default: () => "确认删除该隧道？",
      }),
    ]),
  },
];

const chainColumns: DataTableColumns<ChainItem> = [
  { title: "ID", key: "id", width: 50 },
  {
    title: "角色", key: "chain_type",
    render: (row) => h(NTag, { type: chainTagType(row.chain_type), size: "small" }, () => chainTypeName(row.chain_type)),
  },
  {
    title: "节点", key: "node",
    render: (row) => row.node ? `${row.node.name} (${row.node.host})` : `Node#${row.node_id}`,
  },
  {
    title: "在线", key: "online",
    render: (row) => h(NTag, { type: row.node?.is_online ? "success" : "default", size: "small" }, () => row.node?.is_online ? "在线" : "离线"),
  },
  { title: "端口", key: "port" },
  { title: "协议", key: "protocol" },
  {
    title: "操作", key: "actions",
    render: (row) => h("div", { style: "display:flex;gap:6px" }, [
      h(NButton, { size: "small", type: "info", onClick: () => showNodeInstall(row.node_id) }, { default: () => "安装命令" }),
      h(NPopconfirm, { onPositiveClick: () => removeChainNode(row.id) }, {
        trigger: () => h(NButton, { size: "small", type: "error" }, { default: () => "移除" }),
        default: () => "确认移除该链路节点？",
      }),
    ]),
  },
];

const forwardColumns: DataTableColumns<ForwardItem> = [
  { title: "名称", key: "name" },
  { title: "远程地址", key: "remote_address" },
  { title: "监听端口", key: "listen_port" },
  { title: "协议", key: "protocol" },
  {
    title: "入站代理", key: "inbound",
    render: (row) => row.inbound_enabled ? h(NTag, { type: "warning", size: "small" }, () => row.inbound_type) : "-",
  },
  {
    title: "流量", key: "flow",
    render: (row) => `↑${formatBytes(row.flow_in)} ↓${formatBytes(row.flow_out)}`,
  },
  {
    title: "操作", key: "actions",
    render: (row) => h(NPopconfirm, { onPositiveClick: () => removeForward(row.id) }, {
      trigger: () => h(NButton, { size: "small", type: "error" }, { default: () => "删除" }),
      default: () => "确认删除？",
    }),
  },
];

const formatBytes = (b: number) => {
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(2)} MB`;
  return `${(b / 1024 / 1024 / 1024).toFixed(2)} GB`;
};

// ========== API Calls ==========
const fetchTunnels = async () => {
  loading.value = true;
  try {
    const { data } = await http.get<TunnelItem[]>("/tunnels");
    tunnels.value = data;
  } catch (e: any) {
    message.error(e?.response?.data?.error || "获取隧道失败");
  } finally {
    loading.value = false;
  }
};

const fetchNodes = async () => {
  try {
    const { data } = await http.get("/nodes");
    allNodes.value = data;
  } catch { allNodes.value = []; }
};

const openCreate = () => {
  editingTunnel.value = null;
  Object.assign(tunnelForm, { name: "", type: 1, traffic_ratio: 1.0, inbound_ip: "", is_active: true });
  showTunnelModal.value = true;
};

const openEdit = (row: TunnelItem) => {
  editingTunnel.value = row;
  Object.assign(tunnelForm, { name: row.name, type: row.type, traffic_ratio: row.traffic_ratio, inbound_ip: row.inbound_ip, is_active: row.is_active });
  showTunnelModal.value = true;
};

const saveTunnel = async () => {
  saving.value = true;
  try {
    if (editingTunnel.value) {
      await http.put(`/tunnels/${editingTunnel.value.id}`, tunnelForm);
      message.success("隧道已更新");
    } else {
      await http.post("/tunnels", tunnelForm);
      message.success("隧道已创建");
    }
    showTunnelModal.value = false;
    await fetchTunnels();
  } catch (e: any) {
    message.error(e?.response?.data?.error || "保存失败");
  } finally {
    saving.value = false;
  }
};

const removeTunnel = async (id: number) => {
  await http.delete(`/tunnels/${id}`);
  message.success("隧道已删除");
  await fetchTunnels();
};

const toggleTunnel = async (row: TunnelItem) => {
  await http.put(`/tunnels/${row.id}/toggle`);
  message.success("状态已更新");
  await fetchTunnels();
};

const openDetail = async (row: TunnelItem) => {
  detailTunnel.value = row;
  deployResult.value = null;
  try {
    const { data } = await http.get<TunnelItem>(`/tunnels/${row.id}`);
    detailTunnel.value = data;
    detailChains.value = data.chain_tunnels || [];
    detailForwards.value = data.forwards || [];
  } catch { detailChains.value = []; detailForwards.value = []; }
  showDetailModal.value = true;
};

const addChainNode = async () => {
  if (!chainForm.node_id || !detailTunnel.value) return;
  try {
    await http.post(`/tunnels/${detailTunnel.value.id}/chain`, chainForm);
    message.success("链路节点已添加");
    showAddChain.value = false;
    await openDetail(detailTunnel.value);
  } catch (e: any) { message.error(e?.response?.data?.error || "添加失败"); }
};

const removeChainNode = async (chainId: number) => {
  if (!detailTunnel.value) return;
  await http.delete(`/tunnels/${detailTunnel.value.id}/chain/${chainId}`);
  message.success("已移除");
  await openDetail(detailTunnel.value);
};

const addForward = async () => {
  if (!detailTunnel.value) return;
  try {
    await http.post(`/tunnels/${detailTunnel.value.id}/forwards`, forwardForm);
    message.success("转发已添加");
    showAddForward.value = false;
    await openDetail(detailTunnel.value);
  } catch (e: any) { message.error(e?.response?.data?.error || "添加失败"); }
};

const removeForward = async (fwdId: number) => {
  if (!detailTunnel.value) return;
  await http.delete(`/tunnels/${detailTunnel.value.id}/forwards/${fwdId}`);
  message.success("已删除");
  await openDetail(detailTunnel.value);
};

const deployTunnel = async () => {
  if (!detailTunnel.value) return;
  deploying.value = true;
  try {
    const { data } = await http.post(`/tunnels/${detailTunnel.value.id}/deploy`);
    deployResult.value = data;
    message.success(data.message);
  } catch (e: any) {
    message.error(e?.response?.data?.error || "部署失败");
  } finally {
    deploying.value = false;
  }
};

const undeployTunnel = async () => {
  if (!detailTunnel.value) return;
  deploying.value = true;
  try {
    const { data } = await http.post(`/tunnels/${detailTunnel.value.id}/undeploy`);
    deployResult.value = data;
    message.success("已取消部署");
  } catch (e: any) {
    message.error(e?.response?.data?.error || "操作失败");
  } finally {
    deploying.value = false;
  }
};

const showNodeInstall = async (nodeId: number) => {
  try {
    const { data } = await http.get(`/nodes/${nodeId}/install-command`);
    installCmdText.value = data.install_command;
    showInstallCmd.value = true;
  } catch (e: any) {
    message.error(e?.response?.data?.error || "获取安装命令失败");
  }
};

const copyInstallCmd = () => {
  navigator.clipboard.writeText(installCmdText.value);
  message.success("已复制到剪贴板");
};

onMounted(() => {
  fetchTunnels();
  fetchNodes();
});
</script>
