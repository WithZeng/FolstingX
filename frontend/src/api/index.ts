import axios from "axios";

const http = axios.create({
  baseURL: "/api/v1",
  timeout: 10000,
});

http.interceptors.request.use((config) => {
  const token = localStorage.getItem("folstingx_access_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401 && !location.pathname.startsWith("/login")) {
      localStorage.removeItem("folstingx_access_token");
      localStorage.removeItem("folstingx_refresh_token");
      window.location.href = "/login";
    }
    return Promise.reject(error);
  },
);

export default http;
