import { createFileRoute, redirect } from "@tanstack/react-router";

import { SmartGroupsPage } from "@/features/smart-groups";
import { hasPermission } from "@/lib/admin-permissions";
import { useAuthStore } from "@/stores/auth-store";

export const Route = createFileRoute("/_authenticated/smart-groups/")({
  beforeLoad: () => {
    const user = useAuthStore.getState().auth.user;
    if (!hasPermission(user, "smart_group", "manage")) {
      throw redirect({ to: "/403" });
    }
  },
  component: SmartGroupsPage,
});
