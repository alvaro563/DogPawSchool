// __VITE_API_BASE_URL__ is injected at build time by vite.config.ts
// (see `define: { __VITE_API_BASE_URL__: ... }`). In dev the value
// is "/api/v1" (same-origin via the Vite proxy). In production it
// is the absolute URL of the public API.
//
// We declare it as a global so TypeScript stops complaining. The
// actual injection happens at vite build time; the fallback string
// is what TS sees during plain `tsc --noEmit` against this file.
declare const __VITE_API_BASE_URL__: string | undefined;

const env = {
  API_BASE_URL: (typeof __VITE_API_BASE_URL__ !== 'undefined'
    ? __VITE_API_BASE_URL__
    : '/api/v1') as string,
} as const;

export default env;
