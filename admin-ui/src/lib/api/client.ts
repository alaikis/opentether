import { auth } from "$lib/stores/auth";
import { get } from "svelte/store";
import { browser } from "$app/environment";

const BASE_URL = "/api/v1";

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  params?: Record<string, string>;
};

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public data?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {},
): Promise<T> {
  const { body, params, ...fetchOptions } = options;

  // Build URL with query params
  let url = `${BASE_URL}${endpoint}`;
  if (params) {
    const searchParams = new URLSearchParams(params);
    url += `?${searchParams.toString()}`;
  }

  // Auth header
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((fetchOptions.headers as Record<string, string>) || {}),
  };

  const authState = get(auth);
  if (authState.token) {
    headers["Authorization"] = `Bearer ${authState.token}`;
  }

  const res = await fetch(url, {
    ...fetchOptions,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  // Token expired or role changed - try refresh
  if ((res.status === 401 || res.status === 403) && authState.refreshToken) {
    const refreshed = await auth.refreshToken();
    if (refreshed) {
      // Retry with new token
      const newState = get(auth);
      headers["Authorization"] = `Bearer ${newState.token}`;
      const retryRes = await fetch(url, {
        ...fetchOptions,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!retryRes.ok) {
        const err = await retryRes.json().catch(() => ({}));
        throw new ApiError(
          retryRes.status,
          err.error || err.message || "请求失败",
          err,
        );
      }
      return retryRes.json();
    }
    // Refresh failed, logout
    auth.logout();
    if (browser) window.location.href = "/admin/login";
    throw new ApiError(401, "登录已过期，请重新登录");
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new ApiError(res.status, err.error || err.message || "请求失败", err);
  }

  // Handle empty responses (204, etc)
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text);
}

export const api = {
  get<T>(endpoint: string, options?: RequestOptions) {
    return request<T>(endpoint, { ...options, method: "GET" });
  },
  post<T>(endpoint: string, body?: unknown, options?: RequestOptions) {
    return request<T>(endpoint, { ...options, method: "POST", body });
  },
  put<T>(endpoint: string, body?: unknown, options?: RequestOptions) {
    return request<T>(endpoint, { ...options, method: "PUT", body });
  },
  delete<T>(endpoint: string, options?: RequestOptions) {
    return request<T>(endpoint, { ...options, method: "DELETE" });
  },
};

export { ApiError };
