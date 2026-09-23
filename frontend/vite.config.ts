/// <reference types="vitest/config" />
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'
import { resolve as nodeResolve } from 'node:path'

export default defineConfig(({ mode }) => {
  // Load .env.* into process.env so the proxy config below can read
  // VITE_API_PROXY_TARGET. loadEnv returns the merged env for the
  // given mode (defaults to development for `vite dev`).
  const env = loadEnv(mode, process.cwd(), 'VITE_');

  // API_BASE_URL is what the SPA actually requests. In dev we want
  // it to be SAME-ORIGIN so CSP `connect-src 'self'` works and
  // cookies don't need CORS at all. The Vite proxy below forwards
  // those requests to the real backend.
  //
  // In production (or whenever VITE_API_PROXY_TARGET is unset), the
  // SPA talks to the absolute URL — typically the public API host.
  const apiProxyTarget = env.VITE_API_PROXY_TARGET;
  const apiBaseUrl = apiProxyTarget
    ? '/api/v1'
    : (env.VITE_API_BASE_URL ?? '/api/v1');

  return {
    define: {
      // Expose the resolved base URL to the SPA so http-client.ts
      // can use it without re-deriving.
      __VITE_API_BASE_URL__: JSON.stringify(apiBaseUrl),
    },
    plugins: [
      TanStackRouterVite({
        routesDirectory: './src/routes',
        generatedRouteTree: './src/route-tree.gen.ts',
      }),
      react(),
      tailwindcss(),
      VitePWA({ registerType: 'autoUpdate' }),
    ],
    resolve: {
      alias: { '@': nodeResolve(import.meta.dirname, 'src') },
    },
    server: {
      // Proxy /api/* to the real backend during dev. This makes the
      // SPA's fetch calls same-origin from the browser's point of
      // view, which sidesteps:
      //   - CSP `connect-src 'self'` (no cross-origin allowed)
      //   - CORS preflight (not needed for same-origin)
      //   - Cookie Secure attribute (cookies set on http://localhost
      //     are accepted by the browser without Secure)
      //
      // The browser sees requests to http://localhost:5173/api/v1/*
      // and cookies belong to localhost:5173. Vite forwards them
      // server-side to VITE_API_PROXY_TARGET and re-emits the
      // response headers. The backend's Set-Cookie arrives on
      // localhost:5173 — the browser stores it.
      proxy: apiProxyTarget
        ? {
            '/api': {
              target: apiProxyTarget,
              changeOrigin: true,
              secure: false,
            },
          }
        : undefined,
      // Mirror the production CSP in the dev server so the browser
      // enforces it during local development too. Catches issues
      // before they hit production.
      headers: {
        'Content-Security-Policy':
          "default-src 'self'; img-src 'self' data: https:; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; base-uri 'self'; form-action 'self'",
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'DENY',
        'Referrer-Policy': 'strict-origin-when-cross-origin',
        'Permissions-Policy': 'camera=(), microphone=(), geolocation=()',
      },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: './src/test-setup.ts',
    },
  };
});
