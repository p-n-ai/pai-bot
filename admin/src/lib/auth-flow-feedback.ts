export function getGoogleAuthErrorMessage(code: string | null): string {
  switch (code) {
    case "tenant_required":
      return "Multiple school accounts match this Google email. Sign in with email once, then link Google from inside the admin UI.";
    case "link_required":
      return "We found no Google-linked admin account yet. Sign in with email once, then link Google from inside the admin UI.";
    case "already_linked":
      return "That Google account is already linked to a different admin account.";
    case "flow_invalid":
      return "Your Google sign-in session expired. Start again.";
    case "domain_not_allowed":
      return "That Google account is outside the allowed workspace domain for this admin.";
    case "provider_unavailable":
      return "Google sign-in is not configured in this environment yet.";
    case "google_auth_failed":
      return "Google sign-in failed. Please try again.";
    default:
      return "";
  }
}
