export interface Account {
  id: string;
  email: string;
}

export type AuthResult = { ok: true; account: Account } | { ok: false; status: number; error: string };

const base = import.meta.env.VITE_API_URL ?? "";

async function call(path: string, init: RequestInit): Promise<AuthResult> {
  const response = await fetch(base + path, { ...init, credentials: "include" });
  if (response.status === 204) return { ok: true, account: { id: "", email: "" } };
  const body = (await response.json()) as Account & { error?: string };
  if (response.ok) return { ok: true, account: { id: body.id, email: body.email } };
  return { ok: false, status: response.status, error: body.error ?? "unknown" };
}

const json = { "Content-Type": "application/json" };

export function signUp(email: string, password: string): Promise<AuthResult> {
  return call("/auth/sign-up", { method: "POST", headers: json, body: JSON.stringify({ email, password }) });
}

export function signIn(email: string, password: string): Promise<AuthResult> {
  return call("/auth/sign-in", { method: "POST", headers: json, body: JSON.stringify({ email, password }) });
}

export function signOut(): Promise<AuthResult> {
  return call("/auth/sign-out", { method: "POST" });
}

export function session(): Promise<AuthResult> {
  return call("/auth/session", { method: "GET" });
}
