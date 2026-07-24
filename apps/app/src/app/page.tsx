"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  Archive,
  ArchiveRestore,
  ChevronDown,
  FolderOpen,
  Languages,
  LayoutGrid,
  Link2,
  List as ListIcon,
  Moon,
  MoreHorizontal,
  Pencil,
  Pin,
  Plus,
  Search,
  Sun,
  Trash2,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useI18n } from "@/lib/i18n";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const PAGE_SIZE = 24;

interface WorkspaceSummary {
  id: string;
  slug: string;
  name: string | null;
  description: string | null;
  status: string;
  isPinned: boolean;
  lastActiveAt: string;
  createdAt: string;
}

type SortKey = "lastActive" | "created" | "name";
type StatusFilter = "ALL" | "WARM" | "ARCHIVED";
type ViewMode = "grid" | "list";
type MenuAction =
  | "rename"
  | "editNote"
  | "togglePin"
  | "copyLink"
  | "toggleArchive"
  | "delete";

export default function WorkspaceListPage() {
  const router = useRouter();
  const { t, locale, setLocale } = useI18n();
  const { setTheme, resolvedTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  const [workspaces, setWorkspaces] = useState<WorkspaceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  const [query, setQuery] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("lastActive");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("ALL");
  const [viewMode, setViewMode] = useState<ViewMode>("grid");

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);
  const [toast, setToast] = useState<string | null>(null);
  const [confirmState, setConfirmState] = useState<{
    title: string;
    message: string;
    onConfirm: () => void;
  } | null>(null);

  const [langOpen, setLangOpen] = useState(false);
  const langRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setMounted(true);
    const saved = window.localStorage.getItem("riffpad.viewMode");
    if (saved === "grid" || saved === "list") setViewMode(saved);
  }, []);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (langRef.current && !langRef.current.contains(e.target as Node)) {
        setLangOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const fetchWorkspaces = useCallback(async () => {
    try {
      const res = await fetch(`${API_URL}/api/v1/workspaces`);
      if (res.ok) {
        const data = await res.json();
        setWorkspaces(data ?? []);
      }
    } catch (err) {
      console.error("Failed to fetch workspaces:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchWorkspaces();
  }, [fetchWorkspaces]);

  const showToast = useCallback((text: string) => {
    setToast(text);
    if (toastTimer.current) clearTimeout(toastTimer.current);
    toastTimer.current = setTimeout(() => setToast(null), 1600);
  }, []);

  const askConfirm = useCallback(
    (req: { title: string; message: string; onConfirm: () => void }) => {
      setConfirmState(req);
    },
    []
  );

  const createWorkspace = useCallback(async () => {
    setCreating(true);
    try {
      const res = await fetch(`${API_URL}/api/v1/workspaces`, { method: "POST" });
      if (!res.ok) throw new Error("Failed to create workspace");
      const data = await res.json();
      router.push(`/w/${data.id}`);
    } catch (err) {
      console.error(err);
      setCreating(false);
    }
  }, [router]);

  const patchWorkspace = useCallback(
    async (id: string, fields: Record<string, unknown>) => {
      try {
        const res = await fetch(`${API_URL}/api/v1/workspaces/${id}`, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(fields),
        });
        if (res.ok) {
          const updated: WorkspaceSummary = await res.json();
          setWorkspaces((prev) =>
            prev.map((w) => (w.id === id ? updated : w))
          );
        }
      } catch (err) {
        console.error("Failed to update workspace:", err);
      }
    },
    []
  );

  const doDeleteWorkspace = useCallback(async (id: string) => {
    try {
      const res = await fetch(`${API_URL}/api/v1/workspaces/${id}`, {
        method: "DELETE",
      });
      if (res.ok) {
        setWorkspaces((prev) => prev.filter((w) => w.id !== id));
        setSelected((prev) => {
          const next = new Set(prev);
          next.delete(id);
          return next;
        });
      }
    } catch (err) {
      console.error("Failed to delete workspace:", err);
    }
  }, []);

  const deleteWorkspace = useCallback(
    (id: string) => {
      askConfirm({
        title: t("workspaces.delete"),
        message: t("workspaces.deleteConfirm"),
        onConfirm: () => void doDeleteWorkspace(id),
      });
    },
    [t, askConfirm, doDeleteWorkspace]
  );

  const doDeleteSelected = useCallback(async (ids: string[]) => {
    const results = await Promise.all(
      ids.map((id) =>
        fetch(`${API_URL}/api/v1/workspaces/${id}`, { method: "DELETE" })
      )
    );
    const deleted = new Set(
      ids.filter((_, i) => results[i]?.ok)
    );
    setWorkspaces((prev) => prev.filter((w) => !deleted.has(w.id)));
    setSelected(new Set());
  }, []);

  const deleteSelected = useCallback(() => {
    const ids = Array.from(selected);
    if (ids.length === 0) return;
    askConfirm({
      title: t("workspaces.deleteSelected").replace("{n}", String(ids.length)),
      message: t("workspaces.deleteBatchConfirm").replace(
        "{n}",
        String(ids.length)
      ),
      onConfirm: () => void doDeleteSelected(ids),
    });
  }, [selected, t, askConfirm, doDeleteSelected]);

  // --- Derived list: filter -> sort (pinned first) -> paginate ---

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return workspaces.filter((w) => {
      if (statusFilter !== "ALL" && w.status !== statusFilter) return false;
      if (!q) return true;
      return (
        (w.name ?? "").toLowerCase().includes(q) ||
        w.slug.toLowerCase().includes(q) ||
        (w.description ?? "").toLowerCase().includes(q)
      );
    });
  }, [workspaces, query, statusFilter]);

  const sorted = useMemo(() => {
    const arr = [...filtered];
    arr.sort((a, b) => {
      if (a.isPinned !== b.isPinned) return a.isPinned ? -1 : 1;
      if (sortKey === "lastActive") {
        return b.lastActiveAt.localeCompare(a.lastActiveAt);
      }
      if (sortKey === "created") {
        return b.createdAt.localeCompare(a.createdAt);
      }
      return (a.name || a.slug).localeCompare(b.name || b.slug);
    });
    return arr;
  }, [filtered, sortKey]);

  const visible = sorted.slice(0, visibleCount);
  const hasMore = sorted.length > visible.length;

  // Reset pagination when filters change.
  useEffect(() => {
    setVisibleCount(PAGE_SIZE);
  }, [query, statusFilter, sortKey]);

  // Infinite scroll sentinel.
  useEffect(() => {
    if (!sentinelRef.current || !hasMore) return;
    const observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting) {
        setVisibleCount((n) => n + PAGE_SIZE);
      }
    });
    observer.observe(sentinelRef.current);
    return () => observer.disconnect();
  }, [hasMore, sorted.length]);

  const toggleSelect = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const allVisibleSelected =
    visible.length > 0 && visible.every((w) => selected.has(w.id));

  const toggleSelectAll = useCallback(() => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allVisibleSelected) {
        for (const w of visible) next.delete(w.id);
      } else {
        for (const w of visible) next.add(w.id);
      }
      return next;
    });
  }, [allVisibleSelected, visible]);

  const toggleTheme = () => {
    setTheme(resolvedTheme === "dark" ? "light" : "dark");
  };
  const isDark = resolvedTheme === "dark";

  const switchView = (mode: ViewMode) => {
    setViewMode(mode);
    window.localStorage.setItem("riffpad.viewMode", mode);
  };

  const selectionActive = selected.size > 0;

  return (
    <div className="min-h-dvh bg-canvas text-body flex flex-col">
      {/* Header */}
      <header className="border-b border-hairline bg-card/80 backdrop-blur px-4 lg:px-6 h-14 flex items-center justify-between sticky top-0 z-40">
        <div className="flex items-center gap-3">
          <Image
            src="/logo.png"
            alt="Riffpad"
            width={32}
            height={32}
            className="riffpad-logo rounded-md"
          />
          <div>
            <h1 className="text-[15px] font-bold text-ink leading-tight">
              {t("app.name")}
            </h1>
            <p className="text-[11px] text-mute hidden sm:block">
              {t("app.tagline")}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          <div ref={langRef} className="relative">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setLangOpen((v) => !v)}
              className="text-xs font-semibold h-8 px-2 gap-1"
              aria-expanded={langOpen}
              aria-haspopup="listbox"
            >
              <Languages className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t(`lang.${locale}`)}</span>
              <ChevronDown
                className={`h-3.5 w-3.5 transition-transform ${
                  langOpen ? "rotate-180" : ""
                }`}
              />
            </Button>
            {langOpen && (
              <div
                className="absolute right-0 top-full z-50 mt-1 min-w-[120px] overflow-hidden rounded-md border border-hairline bg-card shadow-lg"
                role="listbox"
              >
                {(["zh", "en"] as const).map((code) => (
                  <button
                    key={code}
                    onClick={() => {
                      setLocale(code);
                      setLangOpen(false);
                    }}
                    className={`flex min-h-[36px] w-full items-center justify-between px-3 py-2 text-left text-sm transition ${
                      locale === code
                        ? "bg-card-soft font-semibold text-ink"
                        : "text-body hover:bg-card-soft hover:text-ink"
                    }`}
                    role="option"
                    aria-selected={locale === code}
                  >
                    {t(`lang.${code}`)}
                    {locale === code && (
                      <span className="h-1.5 w-1.5 rounded-full bg-primary" />
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>

          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            className="h-8 w-8"
            aria-label="toggle theme"
          >
            {mounted && isDark ? (
              <Moon className="h-4 w-4" />
            ) : (
              <Sun className="h-4 w-4" />
            )}
          </Button>
        </div>
      </header>

      {/* Body */}
      <main className="flex-1 w-full max-w-6xl mx-auto px-4 lg:px-6 py-6">
        <div className="flex items-center justify-between mb-4 gap-3">
          <h2 className="text-xl font-extrabold text-ink tracking-tight shrink-0">
            {t("workspaces.title")}
          </h2>
          <Button
            onClick={() => void createWorkspace()}
            disabled={creating}
            size="sm"
            className="bg-primary text-primary-foreground hover:bg-primary-pressed h-8 text-xs font-bold rounded-md gap-1.5 shrink-0"
          >
            <Plus className="h-3.5 w-3.5" />
            {creating ? t("workspaces.creating") : t("workspace.new")}
          </Button>
        </div>

        {/* Toolbar: search / sort / filter / view */}
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <label className="flex items-center gap-2 h-8 flex-1 min-w-[180px] max-w-sm rounded-md border border-hairline bg-card px-2.5 focus-within:ring-2 focus-within:ring-primary/40">
            <Search className="h-3.5 w-3.5 shrink-0 text-mute" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t("workspaces.searchPlaceholder")}
              className="w-full bg-transparent text-sm text-ink placeholder:text-mute focus:outline-none"
              aria-label={t("workspaces.searchPlaceholder")}
            />
          </label>

          <select
            value={sortKey}
            onChange={(e) => setSortKey(e.target.value as SortKey)}
            className="h-8 rounded-md border border-hairline bg-card px-2 text-xs font-semibold text-body focus:outline-none focus:ring-2 focus:ring-primary/40"
            aria-label={t("workspaces.sortLabel")}
          >
            <option value="lastActive">{t("workspaces.sort.lastActive")}</option>
            <option value="created">{t("workspaces.sort.created")}</option>
            <option value="name">{t("workspaces.sort.name")}</option>
          </select>

          <div
            className="flex items-center rounded-md border border-hairline bg-card p-0.5"
            role="group"
            aria-label={t("workspaces.filterLabel")}
          >
            {(["ALL", "WARM", "ARCHIVED"] as const).map((s) => (
              <button
                key={s}
                onClick={() => setStatusFilter(s)}
                className={`h-7 rounded px-2.5 text-xs font-semibold transition ${
                  statusFilter === s
                    ? "bg-card-soft text-ink shadow-sm"
                    : "text-mute hover:text-ink"
                }`}
                aria-pressed={statusFilter === s}
              >
                {t(`workspaces.filter.${s.toLowerCase()}`)}
              </button>
            ))}
          </div>

          <div
            className="flex items-center rounded-md border border-hairline bg-card p-0.5"
            role="group"
            aria-label={t("workspaces.viewLabel")}
          >
            <button
              onClick={() => switchView("grid")}
              className={`h-7 w-7 rounded inline-flex items-center justify-center transition ${
                viewMode === "grid"
                  ? "bg-card-soft text-ink shadow-sm"
                  : "text-mute hover:text-ink"
              }`}
              aria-label={t("workspaces.view.grid")}
              aria-pressed={viewMode === "grid"}
            >
              <LayoutGrid className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => switchView("list")}
              className={`h-7 w-7 rounded inline-flex items-center justify-center transition ${
                viewMode === "list"
                  ? "bg-card-soft text-ink shadow-sm"
                  : "text-mute hover:text-ink"
              }`}
              aria-label={t("workspaces.view.list")}
              aria-pressed={viewMode === "list"}
            >
              <ListIcon className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        {/* Batch bar */}
        {selectionActive && (
          <div className="mb-4 flex items-center gap-3 rounded-lg border border-primary/30 bg-primary/5 px-3 py-2">
            <span className="text-xs font-semibold text-ink">
              {t("workspaces.selected").replace("{n}", String(selected.size))}
            </span>
            <div className="flex-1" />
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSelected(new Set())}
              className="h-7 text-xs font-semibold"
            >
              {t("workspaces.cancel")}
            </Button>
            <Button
              size="sm"
              onClick={() => void deleteSelected()}
              className="h-7 text-xs font-bold bg-accent-red text-white hover:opacity-90 gap-1.5"
            >
              <Trash2 className="h-3.5 w-3.5" />
              {t("workspaces.deleteSelected").replace(
                "{n}",
                String(selected.size)
              )}
            </Button>
          </div>
        )}

        {loading ? (
          <p className="text-sm text-mute">{t("workspace.loading")}</p>
        ) : workspaces.length === 0 ? (
          /* Empty state */
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <div className="relative mb-8">
              <div className="relative w-24 h-24 bg-card border border-hairline rounded-3xl shadow-xl flex items-center justify-center">
                <Image
                  src="/logo.png"
                  alt="Riffpad"
                  width={64}
                  height={64}
                  className="riffpad-logo rounded-lg"
                />
              </div>
            </div>

            <h2 className="text-2xl sm:text-3xl font-extrabold text-ink mb-3 tracking-tight">
              {t("app.name")}
            </h2>
            <p className="text-body max-w-md mb-8 leading-relaxed text-base">
              {t("app.tagline")}
            </p>

            <Button
              onClick={() => void createWorkspace()}
              disabled={creating}
              className="bg-primary text-primary-foreground hover:bg-primary-pressed font-bold h-11 px-8 rounded-lg shadow-lg shadow-primary/20 transition hover:shadow-primary/30 hover:-translate-y-0.5"
            >
              {creating ? t("workspaces.creating") : t("workspace.new")}
            </Button>

            <p className="mt-4 text-xs text-mute">{t("workspace.hint")}</p>
          </div>
        ) : sorted.length === 0 ? (
          <p className="text-sm text-mute py-8 text-center">
            {t("workspaces.noMatch")}
          </p>
        ) : (
          <>
            {/* Select all */}
            <label className="mb-2 inline-flex items-center gap-2 text-xs text-mute cursor-pointer select-none">
              <input
                type="checkbox"
                checked={allVisibleSelected}
                onChange={toggleSelectAll}
                className="h-3.5 w-3.5 rounded border-hairline accent-[var(--primary)]"
                aria-label={t("workspaces.selectAll")}
              />
              {t("workspaces.selectAll")}
            </label>

            {viewMode === "grid" ? (
              <ul className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {visible.map((ws) => (
                  <li key={ws.id}>
                    <WorkspaceCard
                      ws={ws}
                      selected={selected.has(ws.id)}
                      selectionActive={selectionActive}
                      onOpen={() => router.push(`/w/${ws.id}`)}
                      onToggleSelect={() => toggleSelect(ws.id)}
                      onPatch={(fields) => void patchWorkspace(ws.id, fields)}
                      onDelete={() => void deleteWorkspace(ws.id)}
                      onCopyLink={() => {
                        void navigator.clipboard
                          .writeText(`${window.location.origin}/w/${ws.id}`)
                          .then(() => showToast(t("workspaces.linkCopied")));
                      }}
                    />
                  </li>
                ))}
              </ul>
            ) : (
              <ul className="overflow-hidden rounded-xl border border-hairline bg-card divide-y divide-[var(--hairline)]">
                {visible.map((ws) => (
                  <li key={ws.id}>
                    <WorkspaceRow
                      ws={ws}
                      selected={selected.has(ws.id)}
                      selectionActive={selectionActive}
                      onOpen={() => router.push(`/w/${ws.id}`)}
                      onToggleSelect={() => toggleSelect(ws.id)}
                      onPatch={(fields) => void patchWorkspace(ws.id, fields)}
                      onDelete={() => void deleteWorkspace(ws.id)}
                      onCopyLink={() => {
                        void navigator.clipboard
                          .writeText(`${window.location.origin}/w/${ws.id}`)
                          .then(() => showToast(t("workspaces.linkCopied")));
                      }}
                    />
                  </li>
                ))}
              </ul>
            )}

            {hasMore && (
              <div
                ref={sentinelRef}
                className="py-6 text-center text-xs text-mute"
              >
                {t("workspaces.loadingMore")}
              </div>
            )}
          </>
        )}
      </main>

      {/* Toast */}
      {toast && (
        <div className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2 rounded-md bg-ink px-3 py-1.5 text-xs font-semibold text-canvas shadow-lg">
          {toast}
        </div>
      )}

      {/* Confirm dialog */}
      <ConfirmDialog
        open={confirmState !== null}
        title={confirmState?.title ?? ""}
        message={confirmState?.message ?? ""}
        confirmLabel={t("common.delete")}
        cancelLabel={t("common.cancel")}
        onConfirm={() => {
          const action = confirmState?.onConfirm;
          setConfirmState(null);
          action?.();
        }}
        onCancel={() => setConfirmState(null)}
      />
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/* Workspace item (shared card/row logic)                                      */
/* -------------------------------------------------------------------------- */

interface WorkspaceItemProps {
  ws: WorkspaceSummary;
  selected: boolean;
  selectionActive: boolean;
  onOpen: () => void;
  onToggleSelect: () => void;
  onPatch: (fields: Record<string, unknown>) => void;
  onDelete: () => void;
  onCopyLink: () => void;
}

function useWorkspaceItem(props: WorkspaceItemProps) {
  const { t } = useI18n();
  const [menuOpen, setMenuOpen] = useState(false);
  const [editing, setEditing] = useState<"name" | "description" | null>(null);
  const [draft, setDraft] = useState("");
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [menuOpen]);

  const startEdit = (field: "name" | "description") => {
    setEditing(field);
    setDraft(
      field === "name"
        ? props.ws.name ?? ""
        : props.ws.description ?? ""
    );
  };

  const commitEdit = () => {
    if (!editing) return;
    const value = draft.trim();
    if (editing === "name") {
      props.onPatch({ name: value });
    } else {
      props.onPatch({ description: value });
    }
    setEditing(null);
  };

  const handleAction = (action: MenuAction) => {
    setMenuOpen(false);
    switch (action) {
      case "rename":
        startEdit("name");
        break;
      case "editNote":
        startEdit("description");
        break;
      case "togglePin":
        props.onPatch({ isPinned: !props.ws.isPinned });
        break;
      case "copyLink":
        props.onCopyLink();
        break;
      case "toggleArchive":
        props.onPatch({
          status: props.ws.status === "ARCHIVED" ? "WARM" : "ARCHIVED",
        });
        break;
      case "delete":
        props.onDelete();
        break;
    }
  };

  const archived = props.ws.status === "ARCHIVED";

  const formatTime = (iso: string, locale: string) => {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "";
    return d.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const editInput = (
    <input
      autoFocus
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === "Enter") commitEdit();
        if (e.key === "Escape") setEditing(null);
      }}
      onBlur={commitEdit}
      onClick={(e) => e.stopPropagation()}
      className="w-full rounded border border-primary/50 bg-canvas px-1.5 py-0.5 text-sm text-ink focus:outline-none"
      aria-label={editing === "name" ? t("workspaces.rename") : t("workspaces.editNote")}
    />
  );

  return {
    t,
    menuOpen,
    setMenuOpen,
    editing,
    editInput,
    handleAction,
    menuRef,
    archived,
    formatTime,
  };
}

function PinButton({
  pinned,
  selectionActive,
  onToggle,
  label,
}: {
  pinned: boolean;
  selectionActive: boolean;
  onToggle: () => void;
  label: string;
}) {
  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onToggle();
      }}
      className={`rounded-md p-1.5 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 ${
        pinned
          ? "text-primary opacity-100"
          : selectionActive
            ? "text-mute opacity-100"
            : "text-mute opacity-0 group-hover:opacity-100 hover:text-ink"
      }`}
      aria-label={label}
      aria-pressed={pinned}
      title={label}
    >
      <Pin className={`h-4 w-4 ${pinned ? "fill-primary" : ""}`} />
    </button>
  );
}

function SelectCheckbox({
  selected,
  selectionActive,
  onToggle,
  label,
}: {
  selected: boolean;
  selectionActive: boolean;
  onToggle: () => void;
  label: string;
}) {
  return (
    <input
      type="checkbox"
      checked={selected}
      onChange={onToggle}
      onClick={(e) => e.stopPropagation()}
      className={`h-4 w-4 rounded border-hairline accent-[var(--primary)] transition ${
        selected || selectionActive ? "opacity-100" : "opacity-0 group-hover:opacity-100"
      }`}
      aria-label={label}
    />
  );
}

function CardMenu({
  menuRef,
  open,
  onToggle,
  onAction,
  archived,
  pinned,
  labels,
}: {
  menuRef: React.MutableRefObject<HTMLDivElement | null>;
  open: boolean;
  onToggle: () => void;
  onAction: (action: MenuAction) => void;
  archived: boolean;
  pinned: boolean;
  labels: Record<string, string>;
}) {
  const items: { action: MenuAction; label: string; icon: React.ReactNode; danger?: boolean }[] = [
    { action: "rename", label: labels.rename, icon: <Pencil className="h-3.5 w-3.5" /> },
    { action: "editNote", label: labels.editNote, icon: <Pencil className="h-3.5 w-3.5" /> },
    { action: "togglePin", label: pinned ? labels.unpin : labels.pin, icon: <Pin className="h-3.5 w-3.5" /> },
    { action: "copyLink", label: labels.copyLink, icon: <Link2 className="h-3.5 w-3.5" /> },
    {
      action: "toggleArchive",
      label: archived ? labels.unarchive : labels.archive,
      icon: archived ? <ArchiveRestore className="h-3.5 w-3.5" /> : <Archive className="h-3.5 w-3.5" />,
    },
    { action: "delete", label: labels.delete, icon: <Trash2 className="h-3.5 w-3.5" />, danger: true },
  ];

  return (
    <div ref={menuRef} className="relative">
      <button
        onClick={(e) => {
          e.stopPropagation();
          onToggle();
        }}
        className="rounded-md p-1.5 text-mute opacity-0 transition group-hover:opacity-100 hover:bg-card-soft hover:text-ink focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
        aria-label={labels.actions}
        aria-expanded={open}
        aria-haspopup="menu"
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>
      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full z-50 mt-1 min-w-[160px] overflow-hidden rounded-md border border-hairline bg-card shadow-lg"
          onClick={(e) => e.stopPropagation()}
        >
          {items.map((item) => (
            <button
              key={item.action}
              role="menuitem"
              onClick={() => onAction(item.action)}
              className={`flex min-h-[36px] w-full items-center gap-2 px-3 py-2 text-left text-sm transition ${
                item.danger
                  ? "text-accent-red hover:bg-card-soft"
                  : "text-body hover:bg-card-soft hover:text-ink"
              }`}
            >
              {item.icon}
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function useMenuLabels() {
  const { t } = useI18n();
  return {
    actions: t("workspaces.actions"),
    rename: t("workspaces.rename"),
    editNote: t("workspaces.editNote"),
    pin: t("workspaces.pin"),
    unpin: t("workspaces.unpin"),
    copyLink: t("workspaces.copyLink"),
    archive: t("workspaces.archive"),
    unarchive: t("workspaces.unarchive"),
    delete: t("workspaces.delete"),
  };
}

function StatusDot({ status }: { status: string }) {
  return (
    <span
      className={`inline-block h-1.5 w-1.5 rounded-full ${
        status === "WARM" ? "bg-accent-green" : "bg-mute"
      }`}
    />
  );
}

/* ------------------------------- Grid card -------------------------------- */

function WorkspaceCard(props: WorkspaceItemProps) {
  const { ws } = props;
  const { t, locale } = useI18n();
  const item = useWorkspaceItem(props);
  const labels = useMenuLabels();

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={props.onOpen}
      onKeyDown={(e) => {
        if (e.key === "Enter") props.onOpen();
      }}
      className="group relative h-full cursor-pointer rounded-xl border border-hairline bg-card p-4 shadow-sm transition hover:shadow-md hover:border-primary/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
    >
      <div className="absolute left-3 top-3">
        <SelectCheckbox
          selected={props.selected}
          selectionActive={props.selectionActive}
          onToggle={props.onToggleSelect}
          label={t("workspaces.selectOne")}
        />
      </div>

      <div className="absolute right-2 top-2 flex items-center">
        <PinButton
          pinned={ws.isPinned}
          selectionActive={props.selectionActive}
          onToggle={() => props.onPatch({ isPinned: !ws.isPinned })}
          label={ws.isPinned ? t("workspaces.unpin") : t("workspaces.pin")}
        />
        <CardMenu
          menuRef={item.menuRef}
          open={item.menuOpen}
          onToggle={() => item.setMenuOpen(!item.menuOpen)}
          onAction={item.handleAction}
          archived={item.archived}
          pinned={ws.isPinned}
          labels={labels}
        />
      </div>

      <div className="pl-6 pr-12">
        <div className="flex items-center gap-2 min-h-[24px]">
          <FolderOpen className="h-4 w-4 shrink-0 text-mute" />
          {item.editing === "name" ? (
            item.editInput
          ) : (
            <h3 className="truncate text-sm font-bold text-ink">
              {ws.name || ws.slug}
            </h3>
          )}
        </div>
        {ws.name && (
          <p className="mt-0.5 truncate text-[11px] text-mute">{ws.slug}</p>
        )}

        <div className="mt-1.5 min-h-[18px]">
          {item.editing === "description" ? (
            item.editInput
          ) : ws.description ? (
            <p className="truncate text-xs text-body/80">{ws.description}</p>
          ) : null}
        </div>

        <p className="mt-2 flex items-center gap-1.5 text-xs text-mute">
          <StatusDot status={ws.status} />
          {item.archived && (
            <span className="rounded bg-card-soft px-1 py-0.5 text-[10px] font-semibold text-mute">
              {t("workspaces.filter.archived")}
            </span>
          )}
          {t("workspaces.lastActive")} · {item.formatTime(ws.lastActiveAt, locale)}
        </p>
      </div>
    </div>
  );
}

/* -------------------------------- List row -------------------------------- */

function WorkspaceRow(props: WorkspaceItemProps) {
  const { ws } = props;
  const { t, locale } = useI18n();
  const item = useWorkspaceItem(props);
  const labels = useMenuLabels();

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={props.onOpen}
      onKeyDown={(e) => {
        if (e.key === "Enter") props.onOpen();
      }}
      className="group flex cursor-pointer items-center gap-3 px-3 py-2.5 transition hover:bg-card-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary/50"
    >
      <SelectCheckbox
        selected={props.selected}
        selectionActive={props.selectionActive}
        onToggle={props.onToggleSelect}
        label={t("workspaces.selectOne")}
      />
      <PinButton
        pinned={ws.isPinned}
        selectionActive={props.selectionActive}
        onToggle={() => props.onPatch({ isPinned: !ws.isPinned })}
        label={ws.isPinned ? t("workspaces.unpin") : t("workspaces.pin")}
      />

      <div className="min-w-0 w-48 shrink-0">
        {item.editing === "name" ? (
          item.editInput
        ) : (
          <>
            <h3 className="truncate text-sm font-bold text-ink">
              {ws.name || ws.slug}
            </h3>
            {ws.name && (
              <p className="truncate text-[11px] text-mute">{ws.slug}</p>
            )}
          </>
        )}
      </div>

      <div className="hidden sm:block min-w-0 flex-1">
        {item.editing === "description" ? (
          item.editInput
        ) : ws.description ? (
          <p className="truncate text-xs text-body/80">{ws.description}</p>
        ) : null}
      </div>

      <p className="hidden md:flex items-center gap-1.5 text-xs text-mute shrink-0">
        <StatusDot status={ws.status} />
        {item.archived && (
          <span className="rounded bg-card-soft px-1 py-0.5 text-[10px] font-semibold text-mute">
            {t("workspaces.filter.archived")}
          </span>
        )}
        {item.formatTime(ws.lastActiveAt, locale)}
      </p>

      <CardMenu
        menuRef={item.menuRef}
        open={item.menuOpen}
        onToggle={() => item.setMenuOpen(!item.menuOpen)}
        onAction={item.handleAction}
        archived={item.archived}
        pinned={ws.isPinned}
        labels={labels}
      />
    </div>
  );
}

