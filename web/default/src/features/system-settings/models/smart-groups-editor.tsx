import { ChevronDown, ChevronUp, Pencil, Plus, Save, Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";

type SmartRoute = { values: string[]; group: string };
type SmartGroup = {
  description: string;
  parameter: string;
  routes: SmartRoute[];
  default_group: string;
  user_selectable?: boolean;
  user_ids?: number[];
};
type RouteDraft = { id: string; value: string; group: string };
type SmartGroupDraft = Omit<SmartGroup, "routes" | "user_selectable" | "user_ids"> & {
  id: string;
  name: string;
  routes: RouteDraft[];
  user_selectable: boolean;
  user_ids: number[];
};
type SmartGroupUser = { id: number; username: string; display_name: string };

type Props = {
  value: string;
  groupNames: string[];
  onChange: (value: string) => void;
  onInvalidChange: (invalid: boolean) => void;
  onSave: () => void;
  isSaving: boolean;
};

function parseGroups(value: string): SmartGroupDraft[] {
  try {
    const source = JSON.parse(value) as Record<string, SmartGroup>;
    return Object.entries(source).map(([name, group]) => ({
      id: crypto.randomUUID(),
      name,
      description: group.description || "",
      parameter: group.parameter || "",
      routes: (group.routes || []).flatMap((route) =>
        route.values.map((routeValue) => ({
          id: crypto.randomUUID(),
          value: routeValue,
          group: route.group,
        })),
      ),
      default_group: group.default_group || "",
      user_selectable: group.user_selectable ?? true,
      user_ids: group.user_ids || [],
    }));
  } catch {
    return [];
  }
}

export function SmartGroupsEditor({
  value,
  groupNames,
  onChange,
  onInvalidChange,
  onSave,
  isSaving,
}: Props) {
  const { t } = useTranslation();
  const [rows, setRows] = useState(() => parseGroups(value));
  const [sourceValue, setSourceValue] = useState(value);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [expandedUserAssignmentsFor, setExpandedUserAssignmentsFor] = useState<string | null>(null);
  const [userSearch, setUserSearch] = useState("");
  const [userSearchResults, setUserSearchResults] = useState<SmartGroupUser[]>([]);
  const [isSearchingUsers, setIsSearchingUsers] = useState(false);
  const [userDetails, setUserDetails] = useState<Record<number, SmartGroupUser>>({});

  const assignedUserIds = useMemo(
    () => [...new Set(rows.flatMap((row) => row.user_ids))].sort((a, b) => a - b),
    [rows],
  );
  const assignedUserIdsKey = assignedUserIds.join(",");

  useEffect(() => {
    if (!assignedUserIdsKey) return;
    let cancelled = false;
    const ids = assignedUserIdsKey.split(",").map(Number);
    const chunks = Array.from({ length: Math.ceil(ids.length / 200) }, (_, index) =>
      ids.slice(index * 200, (index + 1) * 200),
    );
    Promise.all(
      chunks.map((chunk) =>
        api.get<{ success: boolean; data?: SmartGroupUser[] }>("/api/group/smart/users", {
          params: { ids: chunk.join(",") },
        }),
      ),
    ).then((responses) => {
      if (cancelled) return;
      const users = responses.flatMap((response) => response.data.data || []);
      setUserDetails((current) => ({
        ...current,
        ...Object.fromEntries(users.map((user) => [user.id, user])),
      }));
    }).catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [assignedUserIdsKey]);

  useEffect(() => {
    if (value === sourceValue) return;
    setRows(parseGroups(value));
    setEditingId(null);
    setSourceValue(value);
  }, [sourceValue, value]);

  const duplicateNames = useMemo(() => {
    const names = rows.map((row) => row.name.trim()).filter(Boolean);
    return names.filter((name, index) => names.indexOf(name) !== index);
  }, [rows]);

  const invalid = useMemo(() => {
    if (duplicateNames.length > 0) return true;
    return rows.some((row) => {
      const values = row.routes.map((route) => route.value.trim());
      return (
        !row.name.trim() ||
        row.name.trim() === "auto" ||
        row.name.trim() === "default" ||
        groupNames.includes(row.name.trim()) ||
        !row.parameter.trim() ||
        !groupNames.includes(row.default_group) ||
        values.some((routeValue) => !routeValue) ||
        new Set(values).size !== values.length ||
        row.routes.some((route) => !groupNames.includes(route.group))
      );
    });
  }, [duplicateNames.length, groupNames, rows]);

  useEffect(() => onInvalidChange(invalid), [invalid, onInvalidChange]);

  const emit = useCallback(
    (nextRows: SmartGroupDraft[]) => {
      setRows(nextRows);
      const config: Record<string, SmartGroup> = {};
      for (const row of nextRows) {
        const name = row.name.trim();
        if (!name) continue;
        config[name] = {
          description: row.description,
          parameter: row.parameter.trim(),
          routes: row.routes
            .filter((route) => route.value.trim() && route.group)
            .map((route) => ({
              values: [route.value.trim()],
              group: route.group,
            })),
          default_group: row.default_group,
          user_selectable: row.user_selectable,
          user_ids: row.user_ids,
        };
      }
      const serialized = JSON.stringify(config, null, 2);
      setSourceValue(serialized);
      onChange(serialized);
    },
    [onChange],
  );

  const addGroup = () => {
    let index = 1;
    let name = `smart_group_${index}`;
    while (rows.some((row) => row.name === name) || groupNames.includes(name)) {
      name = `smart_group_${++index}`;
    }
    const id = crypto.randomUUID();
    emit([
      ...rows,
      {
        id,
        name,
        description: "",
        parameter: "resolution",
        routes: [],
        default_group: groupNames[0] || "",
        user_selectable: true,
        user_ids: [],
      },
    ]);
    setEditingId(id);
  };

  const updateRow = (index: number, patch: Partial<SmartGroupDraft>) => {
    emit(rows.map((row, rowIndex) => (rowIndex === index ? { ...row, ...patch } : row)));
  };

  const searchUsers = async () => {
    const keyword = userSearch.trim();
    if (!keyword) {
      setUserSearchResults([]);
      return;
    }
    setIsSearchingUsers(true);
    try {
      const response = await api.get<{ success: boolean; data?: SmartGroupUser[] }>(
        "/api/group/smart/users",
        { params: { keyword } },
      );
      setUserSearchResults(response.data.data || []);
    } catch {
      setUserSearchResults([]);
    } finally {
      setIsSearchingUsers(false);
    }
  };

  const addUser = (index: number, user: SmartGroupUser) => {
    const row = rows[index];
    if (!row || row.user_ids.includes(user.id)) return;
    setUserDetails((current) => ({ ...current, [user.id]: user }));
    updateRow(index, { user_ids: [...row.user_ids, user.id] });
    setUserSearchResults((current) => current.filter((candidate) => candidate.id !== user.id));
  };

  return (
    <Card className="rounded-md">
      <CardHeader className="bg-muted/20 border-b">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle>{t("Smart groups")}</CardTitle>
            <CardDescription>
              {t(
                "Route requests to pricing groups using a request parameter. Smart groups do not have a multiplier.",
              )}
            </CardDescription>
          </div>
          <Button type="button" size="sm" onClick={addGroup} disabled={groupNames.length === 0}>
            <Plus className="mr-2 h-4 w-4" />
            {t("Add smart group")}
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={onSave}
            disabled={isSaving || invalid}
          >
            <Save className="mr-2 h-4 w-4" />
            {isSaving ? t("Saving...") : t("Save smart groups")}
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 pt-4">
        {rows.length === 0 && (
          <p className="text-muted-foreground text-sm">{t("No smart groups configured.")}</p>
        )}
        {groupNames.length === 0 && (
          <p className="text-destructive text-sm">
            {t("Create a pricing group before configuring smart groups.")}
          </p>
        )}
        {rows.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium">{t("Configured smart groups")}</h3>
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {rows.map((row) => (
                <div
                  key={row.id}
                  className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] gap-3 rounded-md border p-3"
                >
                  <div className="min-w-0">
                    <p className="break-words text-sm font-medium">{row.name}</p>
                    <p className="text-muted-foreground text-xs">
                      {row.description || row.parameter} · {t("{{count}} routing rules", { count: row.routes.length })}
                    </p>
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant={editingId === row.id ? "secondary" : "outline"}
                    onClick={() => setEditingId(editingId === row.id ? null : row.id)}
                  >
                    <Pencil className="mr-2 h-4 w-4" />
                    {editingId === row.id ? t("Editing") : t("Edit")}
                  </Button>
                  <div className="col-span-2 min-w-0 border-t pt-2">
                    <div className="mb-1 flex items-center justify-between gap-2">
                      <span className="text-muted-foreground text-xs font-medium">
                        {t("Assigned users")}
                      </span>
                      <span className="text-muted-foreground text-xs tabular-nums">
                        {row.user_ids.length}
                      </span>
                    </div>
                    {row.user_ids.length > 0 ? (
                      <div className="flex max-h-20 flex-wrap gap-1 overflow-y-auto">
                        {row.user_ids.map((userId) => {
                          const user = userDetails[userId];
                          return (
                            <span
                              key={userId}
                              className="bg-muted/60 inline-flex max-w-full items-center rounded px-1.5 py-1 text-xs"
                              title={user ? `${user.display_name || user.username} (${user.username}, #${user.id})` : `#${userId}`}
                            >
                              <span className="truncate">
                                {user
                                  ? `${user.display_name || user.username} (${user.username})`
                                  : `#${userId}`}
                              </span>
                            </span>
                          );
                        })}
                      </div>
                    ) : (
                      <p className="text-muted-foreground text-xs">{t("No users assigned.")}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
        {rows.map((row, index) => {
          if (row.id !== editingId) return null;
          const badName =
            !row.name.trim() ||
            ["auto", "default"].includes(row.name.trim()) ||
            duplicateNames.includes(row.name.trim()) ||
            groupNames.includes(row.name.trim());
          const values = row.routes.map((route) => route.value.trim()).filter(Boolean);
          return (
            <div key={row.id} className="space-y-3 rounded-md border p-3">
              <div className="grid gap-3 md:grid-cols-3">
                <label className="space-y-1 text-sm">
                  <span>{t("Group name")}</span>
                  <Input value={row.name} aria-invalid={badName} onChange={(event) => updateRow(index, { name: event.target.value })} />
                </label>
                <label className="space-y-1 text-sm">
                  <span>{t("Description")}</span>
                  <Input value={row.description} onChange={(event) => updateRow(index, { description: event.target.value })} />
                </label>
                <label className="space-y-1 text-sm">
                  <span>{t("Request parameter")}</span>
                  <Input value={row.parameter} placeholder="resolution" onChange={(event) => updateRow(index, { parameter: event.target.value })} />
                </label>
              </div>
              <div className="space-y-3 rounded-md border p-3">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <label className="flex items-start gap-3 text-sm">
                    <Checkbox
                      checked={row.user_selectable}
                      onCheckedChange={(checked) => {
                        const userSelectable = checked === true;
                        updateRow(index, { user_selectable: userSelectable });
                        if (!userSelectable) setExpandedUserAssignmentsFor(null);
                      }}
                    />
                    <span className="space-y-1">
                      <span className="block font-medium">{t("User selectable")}</span>
                      <span className="text-muted-foreground block text-xs">
                        {row.user_selectable
                          ? t("All users can select this smart group.")
                          : t("Only assigned users can select this smart group.")}
                      </span>
                    </span>
                  </label>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    aria-expanded={expandedUserAssignmentsFor === row.id}
                    onClick={() => setExpandedUserAssignmentsFor(expandedUserAssignmentsFor === row.id ? null : row.id)}
                  >
                    {t("Assigned users")}
                    <span className="bg-muted ml-1 rounded px-1.5 py-0.5 text-xs">{row.user_ids.length}</span>
                    {expandedUserAssignmentsFor === row.id
                      ? <ChevronUp className="ml-1 size-4" />
                      : <ChevronDown className="ml-1 size-4" />}
                  </Button>
                </div>
                {expandedUserAssignmentsFor === row.id && (
                  <div className="space-y-2 border-t pt-3">
                    <label className="block space-y-1 text-sm">
                      <span className="text-muted-foreground block text-xs">
                        {row.user_selectable
                          ? t("Assigned members can use protected pricing groups routed by this smart group.")
                          : t("Search by user ID, username, or display name, then add users to this smart group.")}
                      </span>
                    </label>
                    <div className="flex gap-2">
                      <Input
                        value={userSearch}
                        onChange={(event) => setUserSearch(event.target.value)}
                        onKeyDown={(event) => event.key === "Enter" && void searchUsers()}
                        placeholder={t("Search users to add")}
                      />
                      <Button type="button" variant="outline" onClick={() => void searchUsers()} disabled={isSearchingUsers}>
                        {isSearchingUsers ? t("Searching...") : t("Search")}
                      </Button>
                    </div>
                    {userSearchResults.length > 0 && (
                      <div className="flex max-h-48 flex-wrap gap-2 overflow-auto rounded-md border p-2">
                        {userSearchResults.map((user) => (
                          <button
                            key={user.id}
                            type="button"
                            className="hover:bg-muted inline-flex max-w-full items-center gap-2 rounded border px-2 py-1 text-left text-xs"
                            disabled={row.user_ids.includes(user.id)}
                            onClick={() => addUser(index, user)}
                          >
                            <span className="max-w-64 truncate">
                              {user.display_name || user.username} ({user.username}, #{user.id})
                            </span>
                            <span className="text-muted-foreground shrink-0">{t("Add")}</span>
                          </button>
                        ))}
                      </div>
                    )}
                    {row.user_ids.length > 0 ? (
                      <div className="flex flex-wrap gap-2">
                        {row.user_ids.map((userId) => {
                          const user = userDetails[userId];
                          return (
                            <span key={userId} className="inline-flex max-w-full items-center gap-1 rounded border px-2 py-1 text-xs">
                              <span className="max-w-64 truncate">
                                {user
                                  ? `${user.display_name || user.username} (${user.username}, #${user.id})`
                                  : `#${userId}`}
                              </span>
                              <Button
                                type="button"
                                size="icon"
                                variant="ghost"
                                className="size-6 shrink-0"
                                title={t("Remove")}
                                aria-label={t("Remove")}
                                onClick={() => updateRow(index, { user_ids: row.user_ids.filter((id) => id !== userId) })}
                              >
                                <Trash2 className="size-3" />
                              </Button>
                            </span>
                          );
                        })}
                      </div>
                    ) : (
                      <p className="text-muted-foreground text-sm">{t("No users assigned.")}</p>
                    )}
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium">{t("Routing rules")}</span>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => updateRow(index, { routes: [...row.routes, { id: crypto.randomUUID(), value: "", group: groupNames[0] || "" }] })}
                  >
                    <Plus className="mr-2 h-4 w-4" />
                    {t("Add routing rule")}
                  </Button>
                </div>
                {row.routes.map((route) => (
                  <div key={route.id} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(12rem,0.7fr)_2.5rem] md:items-end">
                    <label className="space-y-1 text-sm">
                      <span>{t("Parameter value")}</span>
                      <Input
                        value={route.value}
                        aria-invalid={!route.value.trim() || values.filter((item) => item === route.value.trim()).length > 1}
                        placeholder={t("Enter a value, for example 720p")}
                        onChange={(event) => updateRow(index, { routes: row.routes.map((item) => item.id === route.id ? { ...item, value: event.target.value } : item) })}
                      />
                    </label>
                    <label className="space-y-1 text-sm">
                      <span>{t("Pricing group")}</span>
                      <select
                        className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
                        value={route.group}
                        onChange={(event) => updateRow(index, { routes: row.routes.map((item) => item.id === route.id ? { ...item, group: event.target.value } : item) })}
                      >
                        <option value="">{t("Select a pricing group")}</option>
                        {groupNames.map((name) => <option key={name} value={name}>{name}</option>)}
                      </select>
                    </label>
                    <Button type="button" variant="ghost" size="icon" aria-label={t("Delete")} onClick={() => updateRow(index, { routes: row.routes.filter((item) => item.id !== route.id) })}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <p className="text-muted-foreground text-xs">
                  {t("Unmatched or missing parameter values use the default pricing group.")}
                </p>
              </div>
              <div className="grid gap-3 md:grid-cols-[minmax(12rem,0.7fr)_auto] md:items-end">
                <label className="space-y-1 text-sm">
                  <span>{t("Default pricing group")}</span>
                  <select className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm" value={row.default_group} onChange={(event) => updateRow(index, { default_group: event.target.value })}>
                    <option value="">{t("Select a pricing group")}</option>
                    {groupNames.map((name) => <option key={name} value={name}>{name}</option>)}
                  </select>
                </label>
                <Button type="button" variant="ghost" size="icon" aria-label={t("Delete")} onClick={() => { emit(rows.filter((_, rowIndex) => rowIndex !== index)); setEditingId(null); }}>
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          );
        })}
        {(duplicateNames.length > 0 || rows.some((row) => groupNames.includes(row.name.trim()) || ["auto", "default"].includes(row.name.trim()))) && (
          <p className="text-destructive text-sm">{t("Smart group names must be unique and cannot match a pricing group.")}</p>
        )}
        {rows.some((row) => row.routes.some((route) => !route.value.trim()) || new Set(row.routes.map((route) => route.value.trim()).filter(Boolean)).size !== row.routes.map((route) => route.value.trim()).filter(Boolean).length) && (
          <p className="text-destructive text-sm">{t("Routing parameter values must not be empty or duplicated within a smart group.")}</p>
        )}
        {rows.some((row) => row.routes.some((route) => !groupNames.includes(route.group)) || !groupNames.includes(row.default_group)) && (
          <p className="text-destructive text-sm">{t("Every route and default must target an existing pricing group.")}</p>
        )}
      </CardContent>
    </Card>
  );
}
