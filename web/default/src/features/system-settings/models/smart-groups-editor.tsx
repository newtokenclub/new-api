import { Plus, Trash2 } from "lucide-react";
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
import { Input } from "@/components/ui/input";

type SmartRoute = { values: string[]; group: string };
type SmartGroup = {
  description: string;
  parameter: string;
  routes: SmartRoute[];
  default_group: string;
};
type RouteDraft = { id: string; value: string; group: string };
type SmartGroupDraft = Omit<SmartGroup, "routes"> & {
  id: string;
  name: string;
  routes: RouteDraft[];
};

type Props = {
  value: string;
  groupNames: string[];
  onChange: (value: string) => void;
  onInvalidChange: (invalid: boolean) => void;
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

  useEffect(() => {
    if (value === sourceValue) return;
    setRows(parseGroups(value));
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
    emit([
      ...rows,
      {
        id: crypto.randomUUID(),
        name,
        description: "",
        parameter: "resolution",
        routes: [],
        default_group: groupNames[0] || "",
      },
    ]);
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
          <Button size="sm" onClick={addGroup} disabled={groupNames.length === 0}>
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
        {rows.map((row, index) => {
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
                <Button type="button" variant="ghost" size="icon" aria-label={t("Delete")} onClick={() => emit(rows.filter((_, rowIndex) => rowIndex !== index))}>
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
