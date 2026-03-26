// History-based router for plain Svelte SPA.
// Reads window.location.pathname, strips basePath if present,
// matches against known route patterns, and exposes reactive state.

import { basePath } from './base';

interface Route {
  name: string;
  path: string;
  params: Record<string, string>;
}

const patterns: { name: string; pattern: RegExp; paramNames: string[] }[] = [
  { name: 'dashboard', pattern: /^\/$/, paramNames: [] },
  { name: 'settings', pattern: /^\/settings$/, paramNames: [] },
  { name: 'newGameserver', pattern: /^\/gameservers\/new$/, paramNames: [] },
  { name: 'gameserverConsole', pattern: /^\/gameservers\/([^/]+)\/console$/, paramNames: ['id'] },
  { name: 'gameserverFiles', pattern: /^\/gameservers\/([^/]+)\/files$/, paramNames: ['id'] },
  { name: 'gameserverBackups', pattern: /^\/gameservers\/([^/]+)\/backups$/, paramNames: ['id'] },
  { name: 'gameserverSchedules', pattern: /^\/gameservers\/([^/]+)\/schedules$/, paramNames: ['id'] },
  { name: 'gameserverSettings', pattern: /^\/gameservers\/([^/]+)\/settings$/, paramNames: ['id'] },
  { name: 'gameserverMods', pattern: /^\/gameservers\/([^/]+)\/mods$/, paramNames: ['id'] },
  { name: 'gameserverOverview', pattern: /^\/gameservers\/([^/]+)$/, paramNames: ['id'] },
];

function parsePath(): Route {
  // Strip basePath prefix (e.g., /panel/abc123) to get the app-relative path
  let path = window.location.pathname;
  if (basePath && path.startsWith(basePath)) {
    path = path.slice(basePath.length) || '/';
  }

  for (const { name, pattern, paramNames } of patterns) {
    const match = path.match(pattern);
    if (match) {
      const params: Record<string, string> = {};
      for (let i = 0; i < paramNames.length; i++) {
        params[paramNames[i]] = match[i + 1];
      }
      return { name, path, params };
    }
  }
  return { name: 'dashboard', path: '/', params: {} };
}

let route = $state<Route>(parsePath());

function handlePopState() {
  route = parsePath();
}

if (typeof window !== 'undefined') {
  window.addEventListener('popstate', handlePopState);
}

export function navigate(path: string) {
  window.history.pushState(null, '', basePath + path);
  route = parsePath();
}

export function getRoute(): Route {
  return route;
}

export function isActive(path: string): boolean {
  const current = route.path;
  if (path === current) return true;
  if (path !== '/' && current.startsWith(path + '/')) return true;
  return false;
}
