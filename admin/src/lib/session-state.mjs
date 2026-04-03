const SESSION_COOKIE = "pai_session";

export function hasClientSession({ sessionToken, user }) {
  return Boolean(sessionToken && user?.user_id && user?.email);
}

export function getClientSessionSnapshot({ sessionToken, user }) {
  const isLoggedIn = hasClientSession({ sessionToken, user });

  return {
    isLoggedIn,
    currentUser: isLoggedIn ? user : null,
  };
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

  return cookies.has(SESSION_COOKIE);
}

export function syncSessionCookies({ sessionToken, user, cookieString, writeCookie }) {
  if (!hasClientSession({ sessionToken, user }) || hasSessionCookies(cookieString)) {
    return false;
  }

  writeCookie(
    `${SESSION_COOKIE}=${encodeURIComponent(sessionToken)}; Path=/; Max-Age=${60 * 60 * 24 * 7}; SameSite=Lax`,
  );

  return true;
}
