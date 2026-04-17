export const primaryNavigation = [
  {
    title: "Dashboard",
    href: "/dashboard",
    description: "Mastery heatmap, nudges, and learner drill-down.",
    group: "Teaching",
    roles: ["teacher", "admin", "platform_admin"],
  },
  {
    title: "Classes",
    href: "/dashboard/classes",
    description: "Mock class setup, join codes, roster, and topic assignment layout.",
    group: "Teaching",
    roles: ["teacher", "admin", "platform_admin"],
  },
  {
    title: "Onboarding",
    href: "/setup/onboard",
    description: "Set curriculum, first class, and a minimal bot preset for a school workspace.",
    group: "Administration",
    roles: ["admin", "platform_admin"],
  },
  {
    title: "Users",
    href: "/settings/users",
    description: "Manage teacher, parent, and admin access plus invite status.",
    group: "Administration",
    roles: ["admin", "platform_admin"],
  },
  {
    title: "Export",
    href: "/export",
    description: "Download student, conversation, and progress exports.",
    group: "Administration",
    roles: ["admin", "platform_admin"],
  },
  {
    title: "WhatsApp",
    href: "/settings/whatsapp",
    description: "Link your WhatsApp account via QR code.",
    group: "Integration",
    roles: ["admin", "platform_admin"],
  },
];

export function getNavigationForUser(user) {
  if (user?.role === "parent" && user?.user_id) {
    return [
      {
        title: "Child Summary",
        href: `/parents/${user.user_id}`,
        description: "Weekly momentum, mastery, and encouragement for home support.",
        group: "Workspace",
      },
    ];
  }

  if (!user?.role) {
    return primaryNavigation;
  }

  return primaryNavigation.filter((item) => item.roles?.includes(user.role));
}

export function isRouteActive(pathname, href) {
  if (!pathname || !href) return false;
  if (href === "/") return pathname === "/";

  const matches = pathname === href || pathname.startsWith(`${href}/`);
  if (!matches) return false;

  const moreSpecificMatch = primaryNavigation.some((item) => {
    if (item.href === href || item.href === "/") {
      return false;
    }

    return item.href.startsWith(`${href}/`) && (pathname === item.href || pathname.startsWith(`${item.href}/`));
  });

  return !moreSpecificMatch;
}

export function getCurrentSection(pathname) {
  if (!pathname) {
    return {
      eyebrow: "Admin panel",
      title: "Dashboard",
      description: "Review mastery, nudges, and learner drill-down.",
    };
  }

  if (pathname.startsWith("/students/")) {
    return {
      eyebrow: "Learner detail",
      title: "Learner profile",
      description: "Review mastery, activity, and conversation history before the next intervention.",
    };
  }

  if (pathname.startsWith("/parents/")) {
    return {
      eyebrow: "Parent support",
      title: "Child summary",
      description: "Review weekly momentum, topic mastery, and a suggested encouragement for home support.",
    };
  }

  if (pathname.startsWith("/settings/whatsapp")) {
    return {
      eyebrow: "Integration",
      title: "WhatsApp setup",
      description: "Link your WhatsApp account via QR code to enable the bot on WhatsApp.",
    };
  }
  if (pathname.startsWith("/settings/users")) {
    return {
      eyebrow: "Administration",
      title: "User management",
      description: "Review active users, invite status, and access operations for the current workspace.",
    };
  }

  if (pathname.startsWith("/setup/onboard")) {
    return {
      eyebrow: "Administration",
      title: "School onboarding",
      description: "Set curriculum, the first class, and a minimal bot preset before sharing the join link.",
    };
  }

  if (pathname.startsWith("/export")) {
    return {
      eyebrow: "Administration",
      title: "Data export",
      description: "Download student, conversation, and progress exports for audit and analysis workflows.",
    };
  }

  if (pathname.startsWith("/dashboard/retrieval-lab") || pathname.startsWith("/dashboard/retreival-lab")) {
    return {
      eyebrow: "Retrieval",
      title: "BM25 query lab",
      description: "Try search queries, filters, and repeat runs against the shared retrieval service.",
    };
  }

  if (pathname.startsWith("/dashboard/classes")) {
    return {
      eyebrow: "Teaching operations",
      title: "Class management",
      description: "Review the planned class setup, join code, roster, and topic assignment layout.",
    };
  }

  const match = primaryNavigation.find((item) => isRouteActive(pathname, item.href));
  if (match) {
    return {
      eyebrow: "Admin panel",
      title: match.title,
      description: match.description,
    };
  }

  return {
    eyebrow: "Admin panel",
    title: "Workspace",
    description: "Monitor teachers, students, and class momentum.",
  };
}

export function getBreadcrumbs(pathname, user) {
  if (!pathname) {
    return [{ label: "Dashboard", href: "/dashboard" }];
  }

  if (pathname.startsWith("/students/")) {
    return [
      { label: "Dashboard", href: "/dashboard" },
      { label: "Learner profile", href: pathname },
    ];
  }

  if (pathname.startsWith("/parents/")) {
    const parentHref = user?.role === "parent" && user?.user_id ? `/parents/${user.user_id}` : pathname;
    return [{ label: "Child summary", href: parentHref }];
  }

  if (pathname.startsWith("/dashboard/classes")) {
    return [
      { label: "Dashboard", href: "/dashboard" },
      { label: "Classes", href: "/dashboard/classes" },
    ];
  }

  if (pathname.startsWith("/settings/whatsapp")) {
    return [
      { label: "Dashboard", href: "/dashboard" },
      { label: "WhatsApp", href: "/settings/whatsapp" },
    ];
  }

  if (pathname.startsWith("/settings/users")) {
    return [
      { label: "Dashboard", href: "/dashboard" },
      { label: "Users", href: "/settings/users" },
    ];
  }

  if (pathname.startsWith("/setup/onboard")) {
    return [
      { label: "Dashboard", href: "/dashboard" },
      { label: "Onboarding", href: "/setup/onboard" },
    ];
  }

  if (pathname.startsWith("/export")) {
    return [
      { label: "Dashboard", href: "/dashboard" },
      { label: "Export", href: "/export" },
    ];
  }

  const activeItem = primaryNavigation.find((item) => isRouteActive(pathname, item.href));
  if (activeItem) {
    return activeItem.href === "/dashboard"
      ? [{ label: activeItem.title, href: activeItem.href }]
      : [
          { label: "Dashboard", href: "/dashboard" },
          { label: activeItem.title, href: activeItem.href },
        ];
  }

  return [{ label: "Dashboard", href: "/dashboard" }];
}
