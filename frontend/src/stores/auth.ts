import { defineStore } from "pinia";
import http from "@/api";

export interface AuthUser {
  id: number;
  username: string;
  role: string;
  api_key: string;
  bandwidth_limit: number;
  traffic_limit: number;
  traffic_used: number;
  is_active: boolean;
  expire_at: string;
}

export const useAuthStore = defineStore("auth", {
  state: () => ({
    accessToken: localStorage.getItem("folstingx_access_token") || "",
    refreshToken: localStorage.getItem("folstingx_refresh_token") || "",
    user: null as AuthUser | null,
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.accessToken),
  },
  actions: {
    async login(username: string, password: string) {
      const { data } = await http.post("/auth/login", { username, password });
      this.accessToken = data.access_token;
      this.refreshToken = data.refresh_token;
      localStorage.setItem("folstingx_access_token", this.accessToken);
      localStorage.setItem("folstingx_refresh_token", this.refreshToken);
      await this.fetchProfile();
    },
    async fetchProfile() {
      const { data } = await http.get<AuthUser>("/auth/profile");
      this.user = data;
    },
    async refresh() {
      if (!this.refreshToken) return;
      const { data } = await http.post("/auth/refresh", {
        refresh_token: this.refreshToken,
      });
      this.accessToken = data.access_token;
      this.refreshToken = data.refresh_token;
      localStorage.setItem("folstingx_access_token", this.accessToken);
      localStorage.setItem("folstingx_refresh_token", this.refreshToken);
    },
    logout() {
      this.accessToken = "";
      this.refreshToken = "";
      this.user = null;
      localStorage.removeItem("folstingx_access_token");
      localStorage.removeItem("folstingx_refresh_token");
    },
  },
});
