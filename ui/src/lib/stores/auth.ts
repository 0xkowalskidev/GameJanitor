import { writable, derived } from 'svelte/store';

export const token = writable<string | null>(null);
export const permissions = writable<string[]>([]);
export const gameserverIds = writable<string[]>([]); // empty = all access

export const isAdmin = derived(permissions, ($perms) =>
  $perms.length > 0 && $perms.includes('settings.edit')
);

export const isAuthenticated = derived(token, ($token) => $token !== null);

export function hasPermission(perm: string): boolean {
  let perms: string[] = [];
  permissions.subscribe(p => perms = p)();
  if (perms.length === 0) return true; // no auth = full access
  return perms.includes(perm);
}

export function hasGameserverAccess(gsId: string): boolean {
  let ids: string[] = [];
  gameserverIds.subscribe(i => ids = i)();
  if (ids.length === 0) return true; // empty = all access
  return ids.includes(gsId);
}

// Load token from cookie on init
export function initAuth() {
  const match = document.cookie.match(/(?:^|; )_token=([^;]*)/);
  if (match) {
    token.set(decodeURIComponent(match[1]));
  }
}

// Store token in cookie
export function setToken(rawToken: string) {
  document.cookie = `_token=${encodeURIComponent(rawToken)}; path=/; max-age=${60 * 60 * 24 * 365}; SameSite=Lax`;
  token.set(rawToken);
}

// Clear token
export function clearToken() {
  document.cookie = '_token=; path=/; max-age=0';
  token.set(null);
  permissions.set([]);
  gameserverIds.set([]);
}
