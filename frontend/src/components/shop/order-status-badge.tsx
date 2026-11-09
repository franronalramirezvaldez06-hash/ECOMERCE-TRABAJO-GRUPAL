import type { OrderStatus } from "@/lib/api/types";
import { STATUS_LABELS, STATUS_TONES } from "@/lib/format";
import { Badge } from "@/components/ui/misc";

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  return <Badge tone={STATUS_TONES[status] ?? "gray"}>{STATUS_LABELS[status] ?? status}</Badge>;
}
