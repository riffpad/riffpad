"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  ChevronDown,
  FolderOpen,
  Languages,
  Moon,
  Plus,
  Sun,
  Trash2,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface WorkspaceSummary {
  id: string;
  slug: string;
  name: string | null;
  status: string;
  lastActiveAt: string;
  createdAt: string;
}

export default function WorkspaceListPage() {
  const router = useRouter();
  const { t, locale, setLocale } = useI18n();
  const { setTheme, resolvedTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  const [workspaces, setWorkspaces] = useState<WorkspaceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [langOpen, setLangOpen] = useState(false);
  const langRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setMounted(true);
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

  const deleteWorkspace = useCallback(
    async (id: string) => {
      if (!window.confirm(t("workspaces.deleteConfirm"))) return;
      try {
        const res = await fetch(`${API_URL}/api/v1/workspaces/${id}`, {
          method: "DELETE",
        });
        if (res.ok) {
          setWorkspaces((prev) => prev.filter((w) => w.id !== id));
        }
      } catch (err) {
        console.error("Failed to delete workspace:", err);
      }
    },
    [t]
  );

  const toggleTheme = () => {
    setTheme(resolvedTheme === "dark" ? "light" : "dark");
  };

  const isDark = resolvedTheme === "dark";

  const formatTime = (iso: string) => {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "";
    return d.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

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
      <main className="flex-1 w-full max-w-5xl mx-auto px-4 lg:px-6 py-8">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-extrabold text-ink tracking-tight">
            {t("workspaces.title")}
          </h2>
          <Button
            onClick={() => void createWorkspace()}
            disabled={creating}
            size="sm"
            className="bg-primary text-primary-foreground hover:bg-primary-pressed h-8 text-xs font-bold rounded-md gap-1.5"
          >
            <Plus className="h-3.5 w-3.5" />
            {creating ? t("workspaces.creating") : t("workspace.new")}
          </Button>
        </div>

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
        ) : (
          <ul className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {workspaces.map((ws) => (
              <li key={ws.id}>
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => router.push(`/w/${ws.id}`)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") router.push(`/w/${ws.id}`);
                  }}
                  className="group h-full cursor-pointer rounded-xl border border-hairline bg-card p-4 shadow-sm transition hover:shadow-md hover:border-primary/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <FolderOpen className="h-4 w-4 shrink-0 text-mute" />
                        <h3 className="truncate text-sm font-bold text-ink">
                          {ws.name || ws.slug}
                        </h3>
                      </div>
                      <p className="mt-1 flex items-center gap-1.5 text-xs text-mute">
                        <span
                          className={`inline-block h-1.5 w-1.5 rounded-full ${
                            ws.status === "WARM"
                              ? "bg-accent-green"
                              : "bg-mute"
                          }`}
                        />
                        {t("workspaces.lastActive")} · {formatTime(ws.lastActiveAt)}
                      </p>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        void deleteWorkspace(ws.id);
                      }}
                      className="shrink-0 rounded-md p-1.5 text-mute opacity-0 transition group-hover:opacity-100 hover:bg-card-soft hover:text-accent-red focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                      aria-label={t("workspaces.delete")}
                      title={t("workspaces.delete")}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}
