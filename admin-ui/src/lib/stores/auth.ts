import { writable, derived } from "svelte/store";
import { browser } from "$app/environment";

export interface User {
  id: string;
  username: string;
  name: string;
  email: string;
  role: string;
  avatar?: string;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  loading: boolean;
}

function createAuthStore() {
  const stored = browser
    ? {
        token: localStorage.getItem("token"),
        refreshToken: localStorage.getItem("refreshToken"),
        user: JSON.parse(localStorage.getItem("user") || "null"),
      }
    : { token: null, refreshToken: null, user: null };

  const { subscribe, set, update } = writable<AuthState>({
    ...stored,
    loading: false,
  });

  return {
    subscribe,

    /** 登录 */
    async login(username: string, password: string) {
      const res = await fetch("/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "登录失败" }));
        throw new Error(err.error || "登录失败");
      }

      const data = await res.json();
      const token = data.token || data.access_token;
      const refreshToken = data.refresh_token;

      const user: User = {
        id: data.user?.id || "",
        username: data.user?.username || username,
        name: data.user?.name || username,
        email: data.user?.email || "",
        role: data.user?.role || "user",
      };

      if (browser) {
        localStorage.setItem("token", token);
        if (refreshToken) localStorage.setItem("refreshToken", refreshToken);
        localStorage.setItem("user", JSON.stringify(user));
      }

      set({ token, refreshToken, user, loading: false });
      return data;
    },

    /** 登出 */
    logout() {
      if (browser) {
        localStorage.removeItem("token");
        localStorage.removeItem("refreshToken");
        localStorage.removeItem("user");
      }
      set({ token: null, refreshToken: null, user: null, loading: false });
    },

    /** 刷新 token */
    async refreshToken() {
      let refreshToken: string | null = null;
      subscribe((state) => {
        refreshToken = state.refreshToken;
      })();

      if (!refreshToken) return false;

      try {
        const res = await fetch("/api/v1/auth/refresh", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
        if (!res.ok) return false;
        const data = await res.json();
        const newToken = data.token || data.access_token;
        if (newToken && browser) {
          localStorage.setItem("token", newToken);
          // 同步刷新用户信息（角色可能已变更）
          if (data.user) {
            const updatedUser: User = {
              id: data.user.id || "",
              username: data.user.username || "",
              name: data.user.name || "",
              email: data.user.email || "",
              role: data.user.role || "user",
            };
            localStorage.setItem("user", JSON.stringify(updatedUser));
            update((s) => ({ ...s, token: newToken, user: updatedUser }));
          } else {
            update((s) => ({ ...s, token: newToken }));
          }
        }
        return true;
      } catch {
        return false;
      }
    },

    /** 强制从 localStorage 同步认证状态 */
    checkAndSync() {
      if (browser) {
        const token = localStorage.getItem("token");
        const refreshToken = localStorage.getItem("refreshToken");
        const user = JSON.parse(localStorage.getItem("user") || "null");
        set({ token, refreshToken, user, loading: false });
      }
    },

    /** 获取当前用户信息 */
    getUser(): User | null {
      if (browser) {
        return JSON.parse(localStorage.getItem("user") || "null");
      }
      return null;
    },
  };
}

export const auth = createAuthStore();

/** 是否已登录（独立导出，可在模板/脚本中直接用 $isAuthenticated） */
export const isAuthenticated = derived(
  auth,
  ($auth) => !!$auth.token && !!$auth.user,
);

/** 是否是管理员（独立导出，可在模板/脚本中直接用 $isAdmin） */
export const isAdmin = derived(auth, ($auth) => $auth.user?.role === "admin");
