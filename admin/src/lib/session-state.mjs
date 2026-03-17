export function hasClientSession({ accessToken, user }) {
  return Boolean(accessToken && user?.user_id && user?.email);
}
