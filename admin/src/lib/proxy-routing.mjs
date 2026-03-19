import { canAccessPath, getDefaultRouteForUser, hasAdminUIAccess } from "./rbac.mjs";

const protectedPrefixes = ["/dashboard", "/students", "/parents"];

export function isProtectedPath(pathname) {
  return protectedPrefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));
}

export function getProxyRedirect(pathname, hasSession, user) {
  if (isProtectedPath(pathname) && !hasSession) {
    return {
      pathname: "/login",
      addNext: true,
    };
  }

  if (isProtectedPath(pathname) && (!hasAdminUIAccess(user) || !canAccessPath(user, pathname))) {
    return {
      pathname: hasAdminUIAccess(user) ? getDefaultRouteForUser(user) : "/login",
      addNext: !hasAdminUIAccess(user),
    };
  }

  if (pathname === "/login" && hasSession && hasAdminUIAccess(user)) {
    return {
      pathname: getDefaultRouteForUser(user),
      addNext: false,
    };
  }

  return null;
}
