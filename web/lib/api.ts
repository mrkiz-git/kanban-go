export type User = {
  id: string;
  email: string;
  name: string;
  role: "admin" | "user";
};

export type APIErrorBody = {
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
};

export class APIError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "APIError";
    this.status = status;
    this.code = code;
  }
}

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  skipAuthRedirect?: boolean;
};

let onUnauthorized: (() => void) | null = null;

export function setUnauthorizedHandler(handler: (() => void) | null) {
  onUnauthorized = handler;
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, skipAuthRedirect, ...fetchOptions } = options;

  const headers = new Headers(fetchOptions.headers);
  if (body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(`/api${path}`, {
    ...fetchOptions,
    credentials: "include",
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const isJSON = res.headers.get("Content-Type")?.includes("application/json");
  const payload = isJSON ? await res.json() : null;

  if (res.status === 401 && !skipAuthRedirect) {
    onUnauthorized?.();
  }

  if (!res.ok) {
    const errBody = payload as APIErrorBody | null;
    const code = errBody?.error?.code ?? "error";
    const message = errBody?.error?.message ?? "Request failed";
    throw new APIError(res.status, code, message);
  }

  return payload as T;
}

export type AuthResponse = {
  token: string;
  user: User;
};

export function loginRequest(email: string, password: string) {
  return api<AuthResponse>("/auth/login", {
    method: "POST",
    body: { email, password },
    skipAuthRedirect: true,
  });
}

export function registerRequest(email: string, password: string, name: string) {
  return api<AuthResponse>("/auth/register", {
    method: "POST",
    body: { email, password, name },
    skipAuthRedirect: true,
  });
}

export function meRequest() {
  return api<User>("/auth/me", { skipAuthRedirect: true });
}

export function logoutRequest() {
  return api<{ status: string }>("/auth/logout", { method: "POST" });
}
