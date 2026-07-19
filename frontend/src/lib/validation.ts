// Mirrors the server-side constraints in API_ENDPOINTS.md's error table so
// the form can fail fast client-side, but the server response is always the
// final word (these are duplicated, not a substitute).

export function validateUsername(username: string): string | null {
  if (username.length < 3 || username.length > 32) {
    return "Username must be 3-32 characters";
  }
  return null;
}

export function validatePassword(password: string): string | null {
  if (password.length < 8 || password.length > 72) {
    return "Password must be 8-72 characters";
  }
  return null;
}
