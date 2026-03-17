export const primaryNavigation = [
  {
    title: "Overview",
    href: "/",
    description: "Admin home and rollout summary",
    roles: ["teacher", "admin", "platform_admin"],
  },
  {
    title: "Teacher Dashboard",
    href: "/dashboard",
    description: "Class mastery heatmap and nudges",
    roles: ["teacher", "admin", "platform_admin"],
  },
  {
    title: "AI Usage",
    href: "/dashboard/ai-usage",
    description: "Review token volume by provider and model across the teacher workspace.",
    roles: ["teacher", "admin", "platform_admin"],
  },
];

export function getNavigationForUser(user) {
  if (user?.role === "parent" && user?.user_id) {
    return [
      {
        title: "Child Summary",
        href: `/parents/${user.user_id}`,
        description: "Weekly momentum, mastery, and encouragement for home support.",
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
      title: "Overview",
      description: "Track rollout progress and open the teacher workspace.",
    };
  }

  if (pathname.startsWith("/students/")) {
    return {
      eyebrow: "Student detail",
      title: "Learner profile",
      description: "Review mastery, streaks, and recent tutoring conversations.",
    };
  }

  if (pathname.startsWith("/parents/")) {
    return {
      eyebrow: "Parent view",
      title: "Child summary",
      description: "Review weekly momentum, topic mastery, and a suggested encouragement for home support.",
    };
  }

  if (pathname.startsWith("/dashboard/ai-usage")) {
    return {
      eyebrow: "Admin panel",
      title: "AI Usage",
      description: "Review token volume by provider and model across the teacher workspace.",
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
