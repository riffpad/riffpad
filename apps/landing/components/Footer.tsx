const footerLinks = [
  {
    title: "产品",
    links: [
      { label: "功能", href: "#features" },
      { label: "价格", href: "#pricing" },
      { label: "更新日志", href: "/changelog" },
    ],
  },
  {
    title: "资源",
    links: [
      { label: "文档", href: "/docs" },
      { label: "GitHub", href: "https://github.com/riffpad/riffpad" },
      { label: "状态页", href: "https://status.riffpad.ai" },
    ],
  },
  {
    title: "公司",
    links: [
      { label: "关于我们", href: "/about" },
      { label: "联系我们", href: "mailto:hello@riffpad.ai" },
      { label: "隐私政策", href: "/privacy" },
    ],
  },
];

export function Footer() {
  return (
    <footer className="border-t border-gray-200 bg-white px-4 py-12 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl">
        <div className="grid gap-10 md:grid-cols-2 lg:grid-cols-5">
          <div className="lg:col-span-2">
            <a href="/" className="flex items-center gap-2 text-lg font-bold text-gray-950">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-950 text-white">
                R
              </span>
              Riffpad
            </a>
            <p className="mt-4 max-w-xs text-sm text-gray-600">
              AI 时代的代码灵感草稿本。随时随地捕捉、运行、交付你的下一个想法。
            </p>
          </div>

          {footerLinks.map((group) => (
            <div key={group.title}>
              <h4 className="text-sm font-semibold text-gray-950">{group.title}</h4>
              <ul className="mt-4 space-y-2">
                {group.links.map((link) => (
                  <li key={link.href}>
                    <a
                      href={link.href}
                      className="text-sm text-gray-600 transition hover:text-gray-950"
                    >
                      {link.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-12 flex flex-col items-center justify-between gap-4 border-t border-gray-100 pt-8 text-sm text-gray-500 sm:flex-row">
          <p>© {new Date().getFullYear()} Riffpad. All rights reserved.</p>
          <div className="flex gap-6">
            <a href="https://github.com/riffpad/riffpad" className="transition hover:text-gray-950">
              GitHub
            </a>
            <a href="https://twitter.com/riffpad" className="transition hover:text-gray-950">
              X / Twitter
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
