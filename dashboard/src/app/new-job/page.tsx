"use client";

import { FormEvent, useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { request } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

type SubmitResponse = {
  run_id: string;
  status: string;
  message: string;
};

export default function NewJobPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    const formData = new FormData(event.currentTarget);
    const payload = {
      keyword: String(formData.get("keyword") ?? ""),
      max_items: Number(formData.get("max_items") ?? 30),
      sort_by: String(formData.get("sort_by") ?? "relevancy"),
      min_price: Number(formData.get("min_price") ?? 0),
      max_price: Number(formData.get("max_price") ?? 0),
    };

    startTransition(async () => {
      try {
        const res = await request<SubmitResponse>("/v1/scrape/tokopedia/search", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        router.push(`/runs/${res.run_id}`);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to submit job");
      }
    });
  }

  return (
    <div className="mx-auto max-w-2xl space-y-5">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">New Tokopedia Job</h1>
        <p className="text-sm text-muted">Submit an async scraping run and jump straight into the detail page.</p>
      </div>

      <Card className="p-5">
        <form onSubmit={onSubmit} className="space-y-4">
          <label className="block space-y-2">
            <span className="text-sm font-medium text-muted">Keyword</span>
            <Input name="keyword" placeholder="iphone 15" required maxLength={100} />
          </label>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block space-y-2">
              <span className="text-sm font-medium text-muted">Max Items</span>
              <Input name="max_items" type="number" min={1} max={200} defaultValue={30} />
            </label>
            <label className="block space-y-2">
              <span className="text-sm font-medium text-muted">Sort By</span>
              <select
                name="sort_by"
                defaultValue="relevancy"
                className="field px-3 py-2 text-sm"
              >
                <option value="relevancy">Relevancy</option>
                <option value="price_asc">Price Asc</option>
                <option value="price_desc">Price Desc</option>
                <option value="latest">Latest</option>
              </select>
            </label>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block space-y-2">
              <span className="text-sm font-medium text-muted">Min Price</span>
              <Input name="min_price" type="number" min={0} defaultValue={0} />
            </label>
            <label className="block space-y-2">
              <span className="text-sm font-medium text-muted">Max Price</span>
              <Input name="max_price" type="number" min={0} defaultValue={0} />
            </label>
          </div>

          {error ? <div className="rounded-md border p-3 text-sm text-[var(--danger)] soft-panel">{error}</div> : null}

          <Button type="submit" disabled={isPending}>
            {isPending ? "Submitting..." : "Submit Job"}
          </Button>
        </form>
      </Card>
    </div>
  );
}
