export function statusClassName(status: string) {
  switch (status) {
    case "SUCCEEDED":
      return "status-succeeded";
    case "RUNNING":
      return "status-running";
    case "FAILED":
    case "TIMED_OUT":
      return "status-failed";
    case "QUEUED":
      return "status-queued";
    default:
      return "status-badge";
  }
}
