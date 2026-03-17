export function getDefaultRouteForUser(user) {
  if (!user) {
    return "/dashboard";
  }

  if (user.role === "parent") {
    return `/parents/${user.user_id}`;
  }

  return "/dashboard";
}
