"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { api } from "@/lib/api";

export async function normalizeRun(formData: FormData) {
  const id = String(formData.get("id") ?? "");
  if (!id) return;
  try {
    await api.request(`/v1/runs/${id}/normalize`, { method: "POST" });
  } catch {}
  revalidatePath(`/runs/${id}`);
  redirect(`/runs/${id}`);
}

export async function generateSummary(formData: FormData) {
  const id = String(formData.get("id") ?? "");
  const prompt = String(formData.get("prompt") ?? "");
  if (!id) return;
  try {
    await api.request(`/v1/runs/${id}/ai-summary`, {
      method: "POST",
      body: JSON.stringify({ prompt }),
    });
  } catch {}
  revalidatePath(`/runs/${id}`);
  redirect(`/runs/${id}`);
}
