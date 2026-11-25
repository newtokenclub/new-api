import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { SectionPageLayout } from "@/components/layout";
import { Button } from "@/components/ui/button";
import { SmartGroupsEditor } from "@/features/system-settings/models/smart-groups-editor";
import { api } from "@/lib/api";

type SmartGroupSettingsResponse = {
  success: boolean;
  data?: {
    smart_groups: string;
    pricing_groups: string[];
  };
  message?: string;
};

export function SmartGroupsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<string | null>(null);
  const [isInvalid, setIsInvalid] = useState(true);
  const settings = useQuery({
    queryKey: ["smart-group-settings"],
    queryFn: async () =>
      (await api.get<SmartGroupSettingsResponse>("/api/group/smart/settings"))
        .data,
    staleTime: 0,
  });

  useEffect(() => {
    if (settings.data?.data?.smart_groups !== undefined) {
      setDraft(settings.data.data.smart_groups);
    }
  }, [settings.data?.data?.smart_groups]);

  const save = useMutation({
    mutationFn: async (value: string) =>
      (await api.put<SmartGroupSettingsResponse>("/api/group/smart/settings", { value }))
        .data,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t("Failed to update setting"));
        return;
      }
      toast.success(t("Smart groups saved successfully"));
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["smart-group-settings"] }),
        queryClient.invalidateQueries({ queryKey: ["system-options"] }),
        queryClient.invalidateQueries({ queryKey: ["groups"] }),
        queryClient.invalidateQueries({ queryKey: ["user-groups"] }),
      ]);
    },
    onError: (error: Error) => {
      toast.error(error.message || t("Failed to update setting"));
    },
  });

  const value = draft ?? settings.data?.data?.smart_groups ?? "{}";
  const groupNames = settings.data?.data?.pricing_groups ?? [];

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t("Smart groups")}</SectionPageLayout.Title>
      {!settings.isLoading && !settings.isError && (
        <SectionPageLayout.Actions>
          <Button
            type="button"
            size="sm"
            onClick={() => save.mutate(value)}
            disabled={save.isPending || isInvalid}
          >
            {save.isPending ? t("Saving...") : t("Save smart groups")}
          </Button>
        </SectionPageLayout.Actions>
      )}
      <SectionPageLayout.Content>
        {settings.isLoading ? (
          <div className="text-muted-foreground flex min-h-40 items-center justify-center text-sm">
            {t("Loading settings...")}
          </div>
        ) : settings.isError ? (
          <div className="text-destructive flex min-h-40 items-center justify-center text-sm">
            {t("Failed to load smart group settings")}
          </div>
        ) : (
          <SmartGroupsEditor
            value={value}
            groupNames={groupNames}
            onChange={setDraft}
            onInvalidChange={setIsInvalid}
          />
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  );
}
