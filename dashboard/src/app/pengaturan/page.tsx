import { AISettingsPanel } from "@/components/settings/AISettingsPanel";
import { MarketplaceSettingsPanel } from "@/components/settings/MarketplaceSettingsPanel";
import { api } from "@/lib/api";

export const dynamic = "force-dynamic";

export default async function PengaturanPage() {
  const settings = await api.getAISettings().catch(() => null);
  const marketplaceSettings = await api.getMarketplaceSettings().catch(() => null);
  return (
    <>
      <AISettingsPanel initialSettings={settings} />
      <MarketplaceSettingsPanel initialSettings={marketplaceSettings} />
    </>
  );
}
