export type RunStatus = "QUEUED" | "RUNNING" | "SUCCEEDED" | "FAILED" | "TIMED_OUT";
export type Marketplace = "tokopedia" | "shopee" | "blibli" | "lazada" | string;

export interface Run {
  id: string;
  status: RunStatus;
  marketplace: Marketplace;
  keyword?: string;
  item_count: number;
  created_at: string;
  started_at?: string | null;
  finished_at?: string | null;
  input?: Record<string, unknown>;
  result?: Product[] | null;
  normalized?: ProductGroup[] | null;
  ai_summary?: AISummaryResult | null;
  error_message?: string | null;
}

export interface Product {
  id: string;
  name: string;
  price: number;
  original_price: number;
  discount_percent: number;
  rating: number;
  count_review: number;
  sold: number;
  url: string;
  image_url: string;
  shop_name: string;
  shop_city: string;
  shop_url?: string;
  is_official_store: boolean;
  marketplace: Marketplace;
  category?: string;
  badge?: string;
}

export interface GroupedItem {
  product_id: string;
  marketplace: Marketplace;
  name: string;
  price: number;
  original_price: number;
  discount_percent: number;
  rating: number;
  count_review: number;
  sold: number;
  shop_name: string;
  shop_city: string;
  is_official_store: boolean;
  url: string;
  image_url: string;
}

export interface ProductGroup {
  group_id: string;
  canonical_name: string;
  brand: string;
  model: string;
  variant: string;
  category_path: string;
  important_specs: string[];
  items: GroupedItem[];
  min_price: number;
  max_price: number;
  avg_price: number;
  best_price_id: string;
}

export interface RecommendedItem {
  group_id: string;
  product_id: string;
  reason: string;
}

export interface AISummaryResult {
  summary_text: string;
  recommended_items: RecommendedItem[];
}

export interface RunsResponse {
  runs: Run[];
  total: number;
  limit: number;
  offset: number;
}
