export async function authenticatedFetch(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  const resp = await fetch(url, {
    ...options,
    credentials: "include",
  });

  if (resp.status === 401) {
    redirectToLogin();
    throw new Error("Unauthorized");
  }

  return resp;
}

export async function checkAuth(): Promise<boolean> {
  try {
    const resp = await fetch("/api/me", { credentials: "include" });
    return resp.ok;
  } catch {
    return false;
  }
}

export function redirectToLogin() {
  const authURL =
    ((window as unknown as Record<string, unknown>).__AUTH_URL__ as string) ||
    "/login";
  const redirectParam = encodeURIComponent(window.location.href);
  window.location.href = `${authURL}?redirect=${redirectParam}`;
}

export async function logout() {
  await fetch("/api/logout", {
    method: "POST",
    credentials: "include",
  });
  redirectToLogin();
}
