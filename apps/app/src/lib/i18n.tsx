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
    "chat.placeholder": "描述你的想法，让 Agent 开始工作…",
    "chat.empty": "还没有消息，发送一个提示开始。",
    "chat.agent": "Agent",
    "chat.user": "You",
    "theme.light": "浅色",
    "theme.dark": "深色",
    "theme.system": "跟随系统",
    "lang.zh": "中文",
    "lang.en": "English",
  },
  en: {
    "app.name": "Riffpad",
    "app.tagline": "AI-native sketchbook for code. Describe an idea, make it run.",
    "workspace.new": "New workspace",
    "workspace.connected": "connected",
    "workspace.disconnected": "disconnected",
    "workspace.empty": "No workspace",
    "workspace.hint": "No setup needed—just describe your idea to start.",
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
    "chat.placeholder": "Describe your idea and let the agent work...",
    "chat.empty": "No messages yet. Send a prompt to start.",
    "chat.agent": "Agent",
    "chat.user": "You",
    "theme.light": "Light",
    "theme.dark": "Dark",
    "theme.system": "System",
    "lang.zh": "中文",
    "lang.en": "English",
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
