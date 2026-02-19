<template>
  <n-space vertical :size="16">
    <n-grid :cols="4" x-gap="12">
      <n-grid-item><n-card title="总上行">{{ formatBytes(stats.total_up) }}</n-card></n-grid-item>
      <n-grid-item><n-card title="总下行">{{ formatBytes(stats.total_down) }}</n-card></n-grid-item>
      <n-grid-item><n-card title="活跃规则">{{ stats.active_rules }}</n-card></n-grid-item>
      <n-grid-item><n-card title="当前连接">{{ stats.connections }}</n-card></n-grid-item>
    </n-grid>

    <n-card title="最近 60 秒流量">
      <v-chart :option="lineOption" autoresize style="height: 320px" />
    </n-card>

    <n-grid :cols="2" x-gap="12">
      <n-grid-item><n-card title="CPU">{{ stats.cpu_percent.toFixed(2) }}%</n-card></n-grid-item>
      <n-grid-item><n-card title="内存">{{ stats.memory_percent.toFixed(2) }}%</n-card></n-grid-item>
    </n-grid>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import VChart from "vue-echarts";
import { use } from "echarts/core";
import { LineChart } from "echarts/charts";
import { GridComponent, TooltipComponent, LegendComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import http from "@/api";

use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const stats = reactive({
  total_up: 0,
  total_down: 0,
  active_rules: 0,
  connections: 0,
  cpu_percent: 0,
  memory_percent: 0,
});

const xData = ref<string[]>([]);
const upSeries = ref<number[]>([]);
const downSeries = ref<number[]>([]);
let ws: WebSocket | null = null;
let reconnectTimer: number | null = null;

const lineOption = computed(() => ({
  tooltip: { trigger: "axis" },
  legend: { data: ["上行", "下行"] },
  xAxis: { type: "category", data: xData.value },
  yAxis: { type: "value" },
  series: [
    { name: "上行", type: "line", data: upSeries.value, smooth: true },
    { name: "下行", type: "line", data: downSeries.value, smooth: true },
  ],
}));

const formatBytes = (n: number) => {
  if (!n) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let idx = 0;
  let v = n;
  while (v >= 1024 && idx < units.length - 1) {
    v /= 1024;
    idx++;
  }
  return `${v.toFixed(2)} ${units[idx]}`;
};

const pushPoint = (ts: number, up: number, down: number) => {
  const label = new Date(ts * 1000).toLocaleTimeString();
  xData.value.push(label);
  upSeries.value.push(up);
  downSeries.value.push(down);
  if (xData.value.length > 60) {
    xData.value.shift();
    upSeries.value.shift();
    downSeries.value.shift();
  }
};

const connectWS = () => {
  const protocol = location.protocol === "https:" ? "wss" : "ws";
  ws = new WebSocket(`${protocol}://${location.host}/ws/monitor`);
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    stats.total_up = data.total_up || 0;
    stats.total_down = data.total_down || 0;
    stats.active_rules = data.active_rules || 0;
    stats.connections = data.total_conn || 0;
    stats.cpu_percent = data.cpu_percent || 0;
    stats.memory_percent = data.mem_percent || 0;
    pushPoint(data.timestamp || Math.floor(Date.now() / 1000), data.total_up || 0, data.total_down || 0);
  };
  ws.onclose = () => {
    reconnectTimer = window.setTimeout(connectWS, 3000);
  };
};

onMounted(async () => {
  try {
    const { data } = await http.get("/monitor/overview");
    Object.assign(stats, data);
  } catch {
    // ignore
  }
  connectWS();
});

onBeforeUnmount(() => {
  if (ws) ws.close();
  if (reconnectTimer) window.clearTimeout(reconnectTimer);
});
</script>
