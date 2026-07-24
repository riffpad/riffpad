"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  ReactNode,
} from "react";

export type Locale = "zh" | "en";

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string) => string;
}

const I18nContext = createContext<I18nContextValue | undefined>(undefined);

const dictionaries: Record<Locale, Record<string, string>> = {
  zh: {
    "app.name": "Riffpad",
    "app.tagline": "AI 原生代码灵感草稿本。描述一个想法，让它跑起来。",
    "workspace.new": "新工作区",
    "workspace.connected": "已连接",
    "workspace.disconnected": "已断开",
    "workspace.empty": "暂无工作区",
    "workspace.hint": "不需要配置，描述你的想法即可开始。",
    "workspace.notFound": "工作区不存在或已被删除",
    "workspace.loading": "正在加载工作区…",
    "workspaces.title": "工作区",
    "workspaces.back": "工作区列表",
    "workspaces.creating": "创建中…",
    "workspaces.lastActive": "最近活跃",
    "workspaces.delete": "删除工作区",
    "workspaces.deleteConfirm": "确定删除这个工作区吗？聊天记录和文件将一并删除，且不可恢复。",
    "workspaces.searchPlaceholder": "搜索名称、代号或备注…",
    "workspaces.sortLabel": "排序方式",
    "workspaces.sort.lastActive": "最近活跃",
    "workspaces.sort.created": "创建时间",
    "workspaces.sort.name": "名称",
    "workspaces.filterLabel": "状态筛选",
    "workspaces.filter.all": "全部",
    "workspaces.filter.warm": "活跃",
    "workspaces.filter.archived": "已归档",
    "workspaces.viewLabel": "视图切换",
    "workspaces.view.grid": "网格视图",
    "workspaces.view.list": "列表视图",
    "workspaces.selected": "已选 {n} 项",
    "workspaces.cancel": "取消",
    "workspaces.deleteSelected": "删除所选 ({n})",
    "workspaces.deleteBatchConfirm": "确定删除选中的 {n} 个工作区吗？聊天记录和文件将一并删除，且不可恢复。",
    "workspaces.selectAll": "全选",
    "workspaces.selectOne": "选择工作区",
    "workspaces.actions": "工作区操作",
    "workspaces.rename": "重命名",
    "workspaces.editNote": "编辑备注",
    "workspaces.pin": "置顶",
    "workspaces.unpin": "取消置顶",
    "workspaces.copyLink": "复制链接",
    "workspaces.linkCopied": "链接已复制",
    "workspaces.archive": "归档",
    "workspaces.unarchive": "恢复",
    "workspaces.noMatch": "没有匹配的工作区",
    "workspaces.loadingMore": "加载更多…",
    "workspaces.emptyTitle": "准备好把你的想法变成代码了吗？",
    "workspaces.emptySubtitle": "只需用自然语言描述，Riffpad 将为你搭建完整的代码环境。",
    "files.title": "文件",
    "files.empty": "还没有文件",
    "files.close": "关闭文件列表",
    "preview.title": "预览",
    "preview.empty": "暂无实时预览",
    "preview.hint": "Agent 可以在工作区启动服务器后，这里会显示运行结果。",
    "code.title": "代码",
    "code.empty": "从左侧文件树选择一个文件查看代码",
    "prompt.send": "发送",
    "chat.title": "聊天",
    "chat.placeholder": "描述你的想法",
    "chat.empty": "还没有消息，发送一个提示开始。",
    "chat.agent": "Agent",
    "chat.user": "You",
    "chat.copy": "复制",
    "chat.regenerate": "重新生成",
    "chat.more": "更多",
    "chat.stop": "停止生成",
    "theme.light": "浅色",
    "theme.dark": "深色",
    "theme.system": "跟随系统",
    "lang.zh": "中文",
    "lang.en": "English",
    "common.cancel": "取消",
    "common.delete": "删除",
  },
  en: {
    "app.name": "Riffpad",
    "app.tagline": "AI-native sketchbook for code. Describe an idea, make it run.",
    "workspace.new": "New workspace",
    "workspace.connected": "connected",
    "workspace.disconnected": "disconnected",
    "workspace.empty": "No workspace",
    "workspace.hint": "No setup needed—just describe your idea to start.",
    "workspace.notFound": "Workspace not found or deleted",
    "workspace.loading": "Loading workspace…",
    "workspaces.title": "Workspaces",
    "workspaces.back": "All workspaces",
    "workspaces.creating": "Creating…",
    "workspaces.lastActive": "Last active",
    "workspaces.delete": "Delete workspace",
    "workspaces.deleteConfirm": "Delete this workspace? Chat history and files will be permanently removed.",
    "workspaces.searchPlaceholder": "Search by name, slug, or note…",
    "workspaces.sortLabel": "Sort by",
    "workspaces.sort.lastActive": "Last active",
    "workspaces.sort.created": "Created",
    "workspaces.sort.name": "Name",
    "workspaces.filterLabel": "Status filter",
    "workspaces.filter.all": "All",
    "workspaces.filter.warm": "Active",
    "workspaces.filter.archived": "Archived",
    "workspaces.viewLabel": "View mode",
    "workspaces.view.grid": "Grid view",
    "workspaces.view.list": "List view",
    "workspaces.selected": "{n} selected",
    "workspaces.cancel": "Cancel",
    "workspaces.deleteSelected": "Delete selected ({n})",
    "workspaces.deleteBatchConfirm": "Delete the {n} selected workspaces? Chat history and files will be permanently removed.",
    "workspaces.selectAll": "Select all",
    "workspaces.selectOne": "Select workspace",
    "workspaces.actions": "Workspace actions",
    "workspaces.rename": "Rename",
    "workspaces.editNote": "Edit note",
    "workspaces.pin": "Pin",
    "workspaces.unpin": "Unpin",
    "workspaces.copyLink": "Copy link",
    "workspaces.linkCopied": "Link copied",
    "workspaces.archive": "Archive",
    "workspaces.unarchive": "Unarchive",
    "workspaces.noMatch": "No workspaces match",
    "workspaces.loadingMore": "Loading more…",
    "workspaces.emptyTitle": "Ready to turn your idea into code?",
    "workspaces.emptySubtitle": "Describe it in plain language — Riffpad builds the full coding environment for you.",
    "files.title": "Files",
    "files.empty": "No files yet",
    "files.close": "Close file list",
    "preview.title": "Preview",
    "preview.empty": "No live preview yet",
    "preview.hint": "The agent can start a server in your workspace; the preview will appear here.",
    "code.title": "Code",
    "code.empty": "Select a file from the sidebar to view its code.",
    "prompt.send": "Send",
    "chat.title": "Chat",
    "chat.placeholder": "Describe your idea",
    "chat.empty": "No messages yet. Send a prompt to start.",
    "chat.agent": "Agent",
    "chat.user": "You",
    "chat.copy": "Copy",
    "chat.regenerate": "Regenerate",
    "chat.more": "More",
    "chat.stop": "Stop generating",
    "theme.light": "Light",
    "theme.dark": "Dark",
    "theme.system": "System",
    "lang.zh": "中文",
    "lang.en": "English",
    "common.cancel": "Cancel",
    "common.delete": "Delete",
  },
};

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("zh");

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
  }, []);

  useEffect(() => {
    if (typeof document !== "undefined") {
      document.documentElement.lang = locale === "zh" ? "zh-CN" : "en";
    }
  }, [locale]);

  const t = useCallback(
    (key: string) => {
      return dictionaries[locale][key] ?? key;
    },
    [locale]
  );

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return ctx;
}
