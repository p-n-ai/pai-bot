const ELEVATED_ROLES = new Set(["teacher", "admin", "platform_admin"]);

export function isPublicEntryRoute(pathname) {
  return pathname === "/" || pathname === "/login";
}

export function isElevatedRole(role) {
  return ELEVATED_ROLES.has(role);
}

export function hasAdminUIAccess(user) {
  return Boolean(user?.user_id && (user?.role === "parent" || isElevatedRole(user?.role)));
}

export function canAccessPath(user, pathname) {
  if (!pathname) return false;

  if (isPublicEntryRoute(pathname)) {
    return true;
  }

  if (!hasAdminUIAccess(user)) {
    return false;
  }

  if (user.role === "parent") {
    return pathname === `/parents/${user.user_id}` || pathname.startsWith(`/parents/${user.user_id}/`);
  }

  return (
    pathname === "/dashboard" ||
    pathname.startsWith("/dashboard/") ||
    pathname === "/students" ||
    pathname.startsWith("/students/") ||
    pathname === "/parents" ||
    pathname.startsWith("/parents/")
  );
}

export function getDefaultRouteForUser(user) {
  if (!hasAdminUIAccess(user)) {
    return "/login";
  }

  if (user.role === "parent") {
    return `/parents/${user.user_id}`;
  }

  return "/dashboard";
}

export function getSafeNextPath(user, pathname) {
  if (pathname && canAccessPath(user, pathname)) {
    return pathname;
  }

  return getDefaultRouteForUser(user);
}
