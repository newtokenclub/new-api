import { Globe2, Pencil, Plus, Search, Trash2 } from "lucide-react";
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

type SmartRoute = { values: string[]; group: string; presence?: PresenceRoute };
type PresenceRoute = {
  parameter: string;
  present_group: string;
  absent_group: string;
};
type SmartGroup = {
  description: string;
  parameter: string;
  routes: SmartRoute[];
  default_group: string;
  default_presence?: PresenceRoute;
  user_selectable?: boolean;
  user_ids?: number[];
};
type RouteDraft = { id: string; value: string; group: string; presence?: PresenceRoute };
type SmartGroupDraft = Omit<SmartGroup, "routes" | "user_selectable" | "user_ids"> & {
  id: string;
  name: string;
  routes: RouteDraft[];
  default_presence?: PresenceRoute;
  user_selectable: boolean;
  user_ids: number[];
};
type SmartGroupUser = { id: number; username: string; display_name: string };

function isValidParameterPath(path: string) {
  return /^[A-Za-z0-9_-]+(?:\[\])?(?:\.[A-Za-z0-9_-]+(?:\[\])?)*$/.test(path.trim());
}

type UserAssignmentProps = {
  userIds: number[];
  userDetails: Record<number, SmartGroupUser>;
  onAdd: (user: SmartGroupUser) => void;
  onRemove: (userId: number) => void;
};

function SmartGroupUserAssignment({
  userIds,
  userDetails,
  onAdd,
  onRemove,
}: UserAssignmentProps) {
  const { t } = useTranslation();
  const [keyword, setKeyword] = useState("");
  const [results, setResults] = useState<SmartGroupUser[]>([]);
  const [isSearching, setIsSearching] = useState(false);

  const searchUsers = async () => {
    const query = keyword.trim();
    if (!query) {
      setResults([]);
      return;
    }
    setIsSearching(true);
    try {
      const response = await api.get<{ success: boolean; data?: SmartGroupUser[] }>(
        "/api/group/smart/users",
        { params: { keyword: query } },
      );
      setResults(response.data.data || []);
    } catch {
      setResults([]);
    } finally {
      setIsSearching(false);
    }
  };

  const addUser = (user: SmartGroupUser) => {
    onAdd(user);
    setResults((current) => current.filter((candidate) => candidate.id !== user.id));
  };

  return (
    <div className="col-span-2 min-w-0 border-t pt-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <span className="text-muted-foreground text-xs font-medium">
          {t("Assigned users")}
        </span>
        <span className="text-muted-foreground text-xs tabular-nums">
          {userIds.length}
        </span>
      </div>
      <p className="text-muted-foreground mb-2 text-xs">
        {t("Only assigned users can select this smart group.")}
      </p>
      <div className="flex gap-2">
        <Input
          className="min-w-0"
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          onKeyDown={(event) => event.key === "Enter" && void searchUsers()}
          placeholder={t("Search users to add")}
          aria-label={t("Search users to add")}
        />
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="shrink-0"
          onClick={() => void searchUsers()}
          disabled={isSearching || !keyword.trim()}
        >
          <Search className="size-4 sm:mr-1.5" />
          <span className="hidden sm:inline">
            {isSearching ? t("Searching...") : t("Search")}
          </span>
        </Button>
      </div>
      {results.length > 0 && (
        <div className="mt-2 flex max-h-28 flex-wrap gap-1.5 overflow-y-auto rounded-md border p-2">
          {results.map((user) => (
            <button
              key={user.id}
              type="button"
              className="hover:bg-muted inline-flex max-w-full items-center gap-1.5 rounded border px-2 py-1 text-left text-xs"
              disabled={userIds.includes(user.id)}
              onClick={() => addUser(user)}
            >
              <span className="max-w-48 truncate">
                {user.display_name || user.username} ({user.username}, #{user.id})
              </span>
              <span className="text-muted-foreground shrink-0">{t("Add")}</span>
            </button>
          ))}
        </div>
      )}
      <div className="mt-2 flex max-h-24 flex-wrap gap-1.5 overflow-y-auto">
        {userIds.map((userId) => {
          const user = userDetails[userId];
          return (
            <span
              key={userId}
              className="bg-muted/60 inline-flex max-w-full items-center gap-1 rounded px-1.5 py-1 text-xs"
              title={user ? `${user.display_name || user.username} (${user.username}, #${user.id})` : `#${userId}`}
            >
              <span className="max-w-48 truncate">
                {user
                  ? `${user.display_name || user.username} (${user.username})`
                  : `#${userId}`}
              </span>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                className="size-5 shrink-0"
                title={t("Remove")}
                aria-label={`${t("Remove")} ${user?.username || userId}`}
                onClick={() => onRemove(userId)}
              >
                <Trash2 className="size-3" />
              </Button>
            </span>
          );
        })}
        {userIds.length === 0 && (
          <p className="text-muted-foreground text-xs">{t("No users assigned.")}</p>
        )}
      </div>
    </div>
  );
}

type SmartGroupTargetEditorProps = {
  group: string;
  presence?: PresenceRoute;
  groupNames: string[];
  onChange: (group: string, presence?: PresenceRoute) => void;
};

function SmartGroupTargetEditor({
  group,
  presence,
  groupNames,
  onChange,
}: SmartGroupTargetEditorProps) {
  const { t } = useTranslation();
  const setPresenceEnabled = (enabled: boolean) => {
    if (enabled) {
      onChange("", {
        parameter: "",
        present_group: groupNames[0] || "",
        absent_group: groupNames[0] || "",
      });
    } else {
      onChange(presence?.present_group || groupNames[0] || "", undefined);
    }
  };

  return (
    <div className="min-w-0 space-y-2">
      {!presence ? (
        <label className="block space-y-1 text-sm">
          <span>{t("Pricing group")}</span>
          <select
            className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
            value={group}
            onChange={(event) => onChange(event.target.value)}
          >
            <option value="">{t("Select a pricing group")}</option>
            {groupNames.map((name) => <option key={name} value={name}>{name}</option>)}
          </select>
        </label>
      ) : (
        <>
          <label className="block space-y-1 text-sm">
            <span>{t("Second-level parameter path")}</span>
            <Input
              value={presence.parameter}
              aria-invalid={!isValidParameterPath(presence.parameter)}
              placeholder="content[].video_url"
              onChange={(event) => onChange(group, { ...presence, parameter: event.target.value })}
            />
            <span className="text-muted-foreground block text-xs">
              {t("Use dots for nested fields and [] for array items, for example content[].video_url.")}
            </span>
          </label>
          <div className="grid gap-2 sm:grid-cols-2">
            <label className="block space-y-1 text-sm">
              <span>{t("When the parameter has a value")}</span>
              <select
                className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
                value={presence.present_group}
                onChange={(event) => onChange(group, { ...presence, present_group: event.target.value })}
              >
                {groupNames.map((name) => <option key={name} value={name}>{name}</option>)}
              </select>
            </label>
            <label className="block space-y-1 text-sm">
              <span>{t("When the parameter has no value")}</span>
              <select
                className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm"
                value={presence.absent_group}
                onChange={(event) => onChange(group, { ...presence, absent_group: event.target.value })}
              >
                {groupNames.map((name) => <option key={name} value={name}>{name}</option>)}
              </select>
            </label>
          </div>
          <p className="text-muted-foreground text-xs">
            {t("Null, blank strings, and empty arrays or objects count as no value. Zero and false count as values.")}
          </p>
        </>
      )}
      <label className="flex items-center gap-2 text-xs">
        <Checkbox checked={!!presence} onCheckedChange={(checked) => setPresenceEnabled(checked === true)} />
        <span>{t("Continue with second-level routing when this branch is selected")}</span>
      </label>
    </div>
  );
}

type Props = {
  value: string;
  groupNames: string[];
  onChange: (value: string) => void;
  onInvalidChange?: (invalid: boolean) => void;
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
          presence: route.presence,
        })),
      ),
      default_group: group.default_group || "",
      default_presence: group.default_presence,
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
}: Props) {
  const { t } = useTranslation();
  const [rows, setRows] = useState(() => parseGroups(value));
  const [sourceValue, setSourceValue] = useState(value);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [userDetails, setUserDetails] = useState<Record<number, SmartGroupUser>>({});
  const orderedRows = useMemo(
    () =>
      rows
        .map((row, index) => ({ row, index }))
        .sort(
          (a, b) =>
            Number(a.row.user_selectable) - Number(b.row.user_selectable) ||
            a.index - b.index,
        ),
    [rows],
  );

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
        (row.default_presence
          ? !isValidParameterPath(row.default_presence.parameter) ||
            !groupNames.includes(row.default_presence.present_group) ||
            !groupNames.includes(row.default_presence.absent_group)
          : !groupNames.includes(row.default_group)) ||
        values.some((routeValue) => !routeValue) ||
        new Set(values).size !== values.length ||
        row.routes.some((route) =>
          route.presence
            ? !isValidParameterPath(route.presence.parameter) ||
              !groupNames.includes(route.presence.present_group) ||
              !groupNames.includes(route.presence.absent_group)
            : !groupNames.includes(route.group),
        )
      );
    });
  }, [duplicateNames.length, groupNames, rows]);

  useEffect(() => onInvalidChange?.(invalid), [invalid, onInvalidChange]);

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
            .filter((route) => route.value.trim() && (route.presence || route.group))
            .map((route) => {
              const serializedRoute: SmartRoute & { presence?: PresenceRoute } = {
                values: [route.value.trim()],
                group: route.presence ? "" : route.group,
                ...(route.presence ? { presence: route.presence } : {}),
              };
              return serializedRoute;
            }),
          default_group: row.default_group,
          user_selectable: row.user_selectable,
          user_ids: row.user_ids,
        };
        if (row.default_presence) {
          config[name].default_presence = row.default_presence;
          config[name].default_group = "";
        }
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
        default_presence: undefined,
        user_selectable: true,
        user_ids: [],
      },
    ]);
    setEditingId(id);
  };

  const updateRow = (index: number, patch: Partial<SmartGroupDraft>) => {
    emit(rows.map((row, rowIndex) => (rowIndex === index ? { ...row, ...patch } : row)));
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
              {orderedRows.map(({ row, index }) => (
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
                  {row.user_selectable ? (
                    <div className="bg-muted/30 text-muted-foreground col-span-2 flex min-h-12 items-center gap-2 rounded-md border px-3 py-2 text-sm">
                      <Globe2 className="text-primary size-4 shrink-0" />
                      <span className="text-foreground font-medium">{t("Public group")}</span>
                      <span>{t("Available to all users")}</span>
                    </div>
                  ) : (
                    <SmartGroupUserAssignment
                      userIds={row.user_ids}
                      userDetails={userDetails}
                      onAdd={(user) => {
                        setUserDetails((current) => ({ ...current, [user.id]: user }));
                        updateRow(index, { user_ids: [...row.user_ids, user.id] });
                      }}
                      onRemove={(userId) =>
                        updateRow(index, { user_ids: row.user_ids.filter((id) => id !== userId) })
                      }
                    />
                  )}
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
                </div>
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
                  <div key={route.id} className="grid gap-2 rounded-md border p-3 md:grid-cols-[minmax(0,1fr)_minmax(14rem,0.8fr)_2.5rem] md:items-end">
                    <label className="space-y-1 text-sm">
                      <span>{t("Parameter value")}</span>
                      <Input
                        value={route.value}
                        aria-invalid={!route.value.trim() || values.filter((item) => item === route.value.trim()).length > 1}
                        placeholder={t("Enter a value, for example 720p")}
                        onChange={(event) => updateRow(index, { routes: row.routes.map((item) => item.id === route.id ? { ...item, value: event.target.value } : item) })}
                      />
                    </label>
                    <SmartGroupTargetEditor
                      group={route.group}
                      presence={route.presence}
                      groupNames={groupNames}
                      onChange={(group, presence) => updateRow(index, {
                        routes: row.routes.map((item) => item.id === route.id ? { ...item, group, presence } : item),
                      })}
                    />
                    <Button type="button" variant="ghost" size="icon" aria-label={t("Delete")} onClick={() => updateRow(index, { routes: row.routes.filter((item) => item.id !== route.id) })}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <p className="text-muted-foreground text-xs">
                  {t("Unmatched or missing parameter values use the default pricing group.")}
                </p>
              </div>
              <div className="grid gap-3 md:grid-cols-[minmax(12rem,1fr)_auto] md:items-end">
                <div>
                  <p className="mb-1 text-sm font-medium">{t("Default pricing group")}</p>
                  <SmartGroupTargetEditor
                    group={row.default_group}
                    presence={row.default_presence}
                    groupNames={groupNames}
                    onChange={(defaultGroup, defaultPresence) => updateRow(index, {
                      default_group: defaultGroup,
                      default_presence: defaultPresence,
                    })}
                  />
                </div>
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
        {rows.some((row) =>
          row.routes.some((route) => route.presence
            ? !isValidParameterPath(route.presence.parameter) ||
              !groupNames.includes(route.presence.present_group) ||
              !groupNames.includes(route.presence.absent_group)
            : !groupNames.includes(route.group)) ||
          (row.default_presence
            ? !isValidParameterPath(row.default_presence.parameter) ||
              !groupNames.includes(row.default_presence.present_group) ||
              !groupNames.includes(row.default_presence.absent_group)
            : !groupNames.includes(row.default_group))
        ) && (
          <p className="text-destructive text-sm">{t("Every route and default must target an existing pricing group.")}</p>
        )}
      </CardContent>
    </Card>
  );
}
