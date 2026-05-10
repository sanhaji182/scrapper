import { humanStatus } from "@/lib/format";

export function StatusBadge({ status }: { status: string }) {
  return (
    <span className={`status-badge status-${status.toLowerCase()}`}>
      <span className="dot" />
      {humanStatus(status)}
    </span>
  );
}
