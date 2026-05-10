import type { AISummaryResult, Product, ProductGroup, Run, RunsResponse } from "./types";

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
export const API_BASE = API_BASE_URL;

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
    cache: "no-store",
  });

  if (!res.ok) {
    let detail = "";
    try {
      const body = await res.json();
      detail = body?.message ? `: ${body.message}` : "";
    } catch {}
    throw new Error(`API error ${res.status}${detail}`);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  request,
  getRuns: (limit = 50, offset = 0) => request<RunsResponse>(`/v1/runs?limit=${limit}&offset=${offset}`),
  getRun: (id: string) => request<Run & { result?: Product[] }>(`/v1/runs/${id}`),
  submitSearch: (body: object, marketplace = "tokopedia") =>
    request<{ run_id: string; status: string; marketplace: string }>(`/v1/scrape/${marketplace}/search`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  getNormalized: (id: string) => request<ProductGroup[]>(`/v1/runs/${id}/normalized`),
  triggerNormalize: (id: string) => request<{ group_cnt: number }>(`/v1/runs/${id}/normalize`, { method: "POST" }),
  getAISummary: (id: string) => request<AISummaryResult>(`/v1/runs/${id}/ai-summary`),
  triggerAISummary: (id: string, prompt?: string) =>
    request<AISummaryResult>(`/v1/runs/${id}/ai-summary`, {
      method: "POST",
      body: JSON.stringify({ prompt: prompt ?? "" }),
    }),
  getAISettings: () => request<AISettings>("/v1/ai/settings"),
  getAIStatus: () => request<AIStatus>("/v1/ai/status"),
  updateAISettings: (body: AISettingsInput) =>
    request<AISettings>("/v1/ai/settings", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  testAISettings: () => request<{ status: string }>("/v1/ai/test", { method: "POST" }),
  getMarketplaceSettings: () => request<MarketplaceSettings>("/v1/marketplace/settings"),
  updateMarketplaceSettings: (body: MarketplaceSettingsInput) =>
    request<MarketplaceSettings>("/v1/marketplace/settings", {
      method: "PUT",
      body: JSON.stringify(body),
    }),
};

export interface AISettings {
  provider: string;
  api_key?: string;
  model: string;
  timeout_sec: number;
  max_retries: number;
  configured: boolean;
}

export interface AISettingsInput {
  provider: string;
  api_key?: string;
  model: string;
  timeout_sec: number;
  max_retries: number;
}

export interface AIStatus {
  ready: boolean;
  message: string;
  provider?: string;
  model?: string;
}


export interface MarketplaceSettings {
  shopee_cookie_header?: string;
  lazada_cookie_header?: string;
}

export interface MarketplaceSettingsInput {
  shopee_cookie_header?: string;
  lazada_cookie_header?: string;
}
