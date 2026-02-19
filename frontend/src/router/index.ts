import { createRouter, createWebHistory } from "vue-router";
import MainLayout from "@/layouts/MainLayout.vue";
import LoginView from "@/views/Login.vue";
import DashboardView from "@/views/Dashboard.vue";
import RulesView from "@/views/Rules.vue";
import NodesView from "@/views/Nodes.vue";
import UsersView from "@/views/Users.vue";
import LogsView from "@/views/Logs.vue";
import SettingsView from "@/views/Settings.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginView },
    {
      path: "/",
      component: MainLayout,
      redirect: "/dashboard",
      children: [
        { path: "/dashboard", name: "dashboard", component: DashboardView },
        { path: "/rules", name: "rules", component: RulesView },
        { path: "/nodes", name: "nodes", component: NodesView },
        { path: "/users", name: "users", component: UsersView },
        { path: "/logs", name: "logs", component: LogsView },
        { path: "/settings", name: "settings", component: SettingsView },
      ],
    },
  ],
});

router.beforeEach((to) => {
  const token = localStorage.getItem("folstingx_access_token");
  if (to.path !== "/login" && !token) {
    return "/login";
  }
  if (to.path === "/login" && token) {
    return "/dashboard";
  }
  return true;
});

export default router;
