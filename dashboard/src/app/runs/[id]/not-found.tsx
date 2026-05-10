import Link from "next/link";
import { Card } from "@/components/ui/card";

export default function RunNotFound() {
  return (
    <Card className="mx-auto max-w-xl p-8 text-center">
      <p className="text-xs font-medium uppercase tracking-[0.24em] text-faint">Run not found</p>
      <h1 className="mt-3 text-2xl font-semibold tracking-tight">This run is no longer available</h1>
      <p className="mt-2 text-sm text-muted">The ID may be stale or removed. Go back to the runs list and open a current run.</p>
      <Link href="/runs" className="mt-5 inline-flex rounded-md primary-button px-4 py-2 text-sm">
        Back to runs
      </Link>
    </Card>
  );
}
