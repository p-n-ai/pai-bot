const ACCESS_TOKEN_COOKIE = "pai_admin_access";
const USER_COOKIE = "pai_admin_user";

export function hasClientSession({ accessToken, user }) {
  return Boolean(accessToken && user?.user_id && user?.email);
}

export function hasSessionCookies(cookieString) {
  if (!cookieString) {
    return false;
  }

  const cookies = new Set(
    cookieString
      .split(";")
      .map((entry) => entry.trim().split("=", 1)[0])
      .filter(Boolean),
  );

  return cookies.has(ACCESS_TOKEN_COOKIE) && cookies.has(USER_COOKIE);
}

export function syncSessionCookies({ accessToken, user, cookieString, writeCookie }) {
  if (!hasClientSession({ accessToken, user }) || hasSessionCookies(cookieString)) {
    return false;
  }

  writeCookie(
    `${ACCESS_TOKEN_COOKIE}=${encodeURIComponent(accessToken)}; Path=/; Max-Age=${60 * 60 * 24 * 7}; SameSite=Lax`,
  );
  writeCookie(
    `${USER_COOKIE}=${encodeURIComponent(JSON.stringify(user))}; Path=/; Max-Age=${60 * 60 * 24 * 7}; SameSite=Lax`,
  );

  return true;
}
