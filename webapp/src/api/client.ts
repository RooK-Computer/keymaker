import createClient from 'openapi-fetch';
import type { paths } from './schema';

export const apiClient = createClient<paths>({
  baseUrl: '/api/v1'
});

export class APIError extends Error {
  public readonly status: number;
  public readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.code = code;
  }
}

function normalizeErrorMessage(error: unknown): { message: string; code?: string } {
  if (!error || typeof error !== 'object') {
    return { message: 'request failed' };
  }

  const maybeError = error as { message?: unknown; error?: unknown };
  const message = typeof maybeError.message === 'string' ? maybeError.message : 'request failed';
  const code = typeof maybeError.error === 'string' ? maybeError.error : undefined;
  return { message, code };
}

async function unwrapJson<T>(
  response: Promise<{ data?: T; error?: unknown; response: Response }>
): Promise<T> {
  const { data, error, response: rawResponse } = await response;
  if (error) {
    const details = normalizeErrorMessage(error);
    throw new APIError(details.message, rawResponse.status, details.code);
  }
  if (data === undefined) {
    throw new APIError('empty response', rawResponse.status);
  }
  return data;
}

export function getCartridgeInfo(signal?: AbortSignal) {
  return unwrapJson(apiClient.GET('/cartridgeinfo', { signal }));
}

export function ejectCartridge(signal?: AbortSignal) {
  return unwrapJson(apiClient.POST('/eject', { signal }));
}

export function listRetroPieSystems(signal?: AbortSignal) {
  return unwrapJson(apiClient.GET('/retropie', { signal }));
}

export function listRetroPieGames(system: string, signal?: AbortSignal) {
  return unwrapJson(apiClient.GET('/retropie/{system}', { params: { path: { system } }, signal }));
}

export async function deleteRetroPieGame(system: string, game: string, signal?: AbortSignal) {
  return unwrapJson(
    apiClient.DELETE('/retropie/{system}/{game}', {
      params: { path: { system, game } },
      signal
    })
  );
}

async function fetchOrThrow(request: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const response = await fetch(request, init);
  if (response.ok) {
    return response;
  }

  // Best-effort parse of JSON error body.
  try {
    const errorBody = (await response.json()) as unknown;
    const details = normalizeErrorMessage(errorBody);
    throw new APIError(details.message, response.status, details.code);
  } catch {
    throw new APIError(response.statusText || 'request failed', response.status);
  }
}

// Raw-byte endpoints (download/upload/flash) are handled via plain fetch.
// openapi-typescript models `format: binary` as `string`, which is not ergonomic
// for browser streaming types (Blob/ArrayBuffer).

export function downloadRetroPieGame(system: string, game: string, signal?: AbortSignal) {
  const url = `/api/v1/retropie/${encodeURIComponent(system)}/${encodeURIComponent(game)}`;
  return fetchOrThrow(url, { method: 'GET', signal });
}

export async function uploadRetroPieGame(system: string, game: string, body: BodyInit, signal?: AbortSignal) {
  const url = `/api/v1/retropie/${encodeURIComponent(system)}/${encodeURIComponent(game)}`;
  const response = await fetchOrThrow(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/octet-stream'
    },
    body,
    signal
  });
  return (await response.json()) as { ok: boolean };
}

export async function flashCartridge(body: BodyInit, contentType: 'application/octet-stream' | 'application/gzip', signal?: AbortSignal) {
  const response = await fetchOrThrow('/api/v1/flash', {
    method: 'POST',
    headers: {
      'Content-Type': contentType
    },
    body,
    signal
  });
  return (await response.json()) as { ok: boolean };
}
