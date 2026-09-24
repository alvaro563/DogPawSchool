import env from '@/config/env';

export interface ApiError {
  status: number;
  body: unknown;
  /**
   * Seconds to wait before retrying, parsed from the RFC 6585
   * `Retry-After` response header. Only populated for 429 and 503
   * responses; otherwise null. The LoginForm / RegisterForm
   * components use this to show a live countdown and disable the
   * submit button for the duration.
   */
  retryAfterSeconds: number | null;
}

/**
 * refreshInFlight is the single in-flight refresh Promise shared
 * across all concurrent requests. When a 401 hits the SPA, multiple
 * in-flight fetches may notice the access token expired at the
 * same moment. Without coordination, each would fire its own
 * POST /auth/refresh — wasting 3 round-trips, racing the cookie
 * write, and (worst case) emitting 3 different access tokens.
 *
 * The pattern: the FIRST request to see the 401 kicks off the
 * refresh; every concurrent request awaits the same Promise. After
 * it resolves, they retry once with the new cookie. Subsequent 401s
 * after the refresh resolved with a fresh token mean the session is
 * truly dead — we redirect to /auth/login.
 */
let refreshInFlight: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    try {
      const r = await fetch(`${env.API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      });
      return r.ok;
    } catch {
      return false;
    } finally {
      // Allow the next 401 to start a new refresh.
      setTimeout(() => {
        refreshInFlight = null;
      }, 0);
    }
  })();
  return refreshInFlight;
}

class HttpClient {
  private baseURL: string;

  constructor(baseURL: string) {
    this.baseURL = baseURL;
  }

  public getBaseURL(): string {
    return this.baseURL;
  }

  private async request<T>(
    method: string,
    path: string,
    options: {
      body?: unknown;
      params?: Record<string, string>;
      skipAuthRedirect?: boolean;
      _retried?: boolean;
    } = {},
  ): Promise<T> {
    const url = this.buildURL(path, options.params);
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    let response: Response;
    try {
      response = await fetch(url, {
        method,
        headers,
        credentials: 'include',
        body: options.body ? JSON.stringify(options.body) : undefined,
      });
    } catch (err) {
      // fetch() threw a TypeError. This happens when the request
      // never reached the server: network failure, CORS rejection,
      // or CSP block. The browser refuses to surface the response
      // to JS in those cases — we never see a Response object.
      //
      // Surface a sentinel ApiError with status=undefined so the
      // caller can distinguish "the server said no" from "we never
      // got an answer". The login form uses this to show a
      // network-shaped message instead of the generic "credenciales
      // incorrectas" path.
      throw {
        status: undefined,
        body: { error: 'network_error', details: err instanceof Error ? err.message : String(err) },
        retryAfterSeconds: null,
      } as unknown as ApiError;
    }

    if (response.status === 401 && !options._retried) {
      // Attempt a single refresh + retry on ANY 401, regardless of
      // skipAuthRedirect. The two concerns are decoupled on purpose:
      //   - refresh attempt: desired even for auth endpoints (e.g.
      //     me() with an expired access cookie but valid refresh
      //     cookie must renew silently);
      //   - hard redirect: gated by skipAuthRedirect below, because
      //     a 401 from /auth/login or /auth/password is a domain
      //     rejection (wrong credentials), not a dead session.
      const refreshed = await tryRefresh();
      if (refreshed) {
        return this.request<T>(method, path, { ...options, _retried: true });
      }
      // Fall through to the 401 handling below.
    }

    if (response.status === 401) {
      // Hard redirect: the access cookie may still be set in the
      // browser but the server has rejected it (token_version
      // bumped or cookie expired beyond refresh). Send the user to
      // the login page.
      //
      // Two guards prevent a reload loop:
      //   1. skipAuthRedirect: callers that render the error
      //      inline (login form, bootstrap me(), logout) opt out.
      //   2. pathname check: never assign href to the page we are
      //      already on — that would re-navigate the document on
      //      every 401, and /auth/login itself 401s on bootstrap.
      if (
        !options.skipAuthRedirect &&
        window.location.pathname !== '/auth/login'
      ) {
        window.location.href = '/auth/login';
      }
      const errorBody = await response.json().catch(() => null);
      throw {
        status: 401,
        body: errorBody ?? { error: 'unauthorized' },
        retryAfterSeconds: parseRetryAfter(response.headers.get('Retry-After')),
      } as ApiError;
    }

    if (!response.ok) {
      const errorBody = await response.json().catch(() => null);
      throw {
        status: response.status,
        body: errorBody,
        retryAfterSeconds: parseRetryAfter(response.headers.get('Retry-After')),
      } as ApiError;
    }

    return response.json() as Promise<T>;
  }

  get<T>(
    path: string,
    params?: Record<string, string>,
    options?: { skipAuthRedirect?: boolean },
  ): Promise<T> {
    return this.request<T>('GET', path, {
      params,
      skipAuthRedirect: options?.skipAuthRedirect,
    });
  }

  post<T>(
    path: string,
    body?: unknown,
    options?: { skipAuthRedirect?: boolean },
  ): Promise<T> {
    return this.request<T>('POST', path, {
      body,
      skipAuthRedirect: options?.skipAuthRedirect,
    });
  }

  patch<T>(path: string, body?: unknown, options?: { skipAuthRedirect?: boolean }): Promise<T> {
    return this.request<T>('PATCH', path, { body, skipAuthRedirect: options?.skipAuthRedirect });
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>('DELETE', path);
  }

  /**
   * GET that returns the body as a Blob. Used for binary downloads
   * (e.g. CSV). The cookie travels via credentials: 'include'; the
   * SPA never reads the token directly.
   */
  public async getBlob(path: string, params?: Record<string, string>): Promise<Blob> {
    const url = this.buildURL(path, params);
    const response = await fetch(url, {
      method: 'GET',
      credentials: 'include',
    });

    if (response.status === 401) {
      // No skipAuthRedirect path here because there's no
      // caller-driven login form to render — a binary download
      // that needs auth always redirects on 401.
      const refreshed = await tryRefresh();
      if (refreshed) {
        return this.getBlob(path, params);
      }
      window.location.href = '/auth/login';
      throw {
        status: 401,
        body: { error: 'unauthorized' },
        retryAfterSeconds: parseRetryAfter(response.headers.get('Retry-After')),
      } as ApiError;
    }

    if (!response.ok) {
      let errorBody: unknown = null;
      try {
        errorBody = await response.json();
      } catch {
        errorBody = null;
      }
      throw {
        status: response.status,
        body: errorBody,
        retryAfterSeconds: parseRetryAfter(response.headers.get('Retry-After')),
      } as ApiError;
    }

    return response.blob();
  }

  private buildURL(path: string, params?: Record<string, string>): string {
    // baseURL may be relative in dev ("/api/v1" via the Vite proxy)
    // or absolute in production ("https://api.example.com/api/v1").
    // new URL() throws TypeError on a relative URL with no base, so
    // always pass window.location.origin — it is ignored when the
    // target is already absolute.
    const url = new URL(`${this.baseURL}${path}`, window.location.origin);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
      });
    }
    return url.toString();
  }
}

function parseRetryAfter(header: string | null): number | null {
  if (!header) return null;
  const seconds = Number.parseInt(header, 10);
  if (Number.isFinite(seconds) && seconds >= 0) {
    return seconds;
  }
  return null;
}

const apiClient = new HttpClient(env.API_BASE_URL);
export default apiClient;
