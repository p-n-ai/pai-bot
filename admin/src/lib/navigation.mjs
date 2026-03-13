export const primaryNavigation = [
  {
    title: "Overview",
    href: "/",
    description: "Admin home and rollout summary",
  },
  {
    title: "Teacher Dashboard",
    href: "/dashboard",
    description: "Class mastery heatmap and nudges",
  },
];

export function isRouteActive(pathname, href) {
  if (!pathname || !href) return false;
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
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
