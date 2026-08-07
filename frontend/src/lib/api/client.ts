import { publicEnvironment } from "@/config/env";
import { ApiError, apiErrorFromResponse } from "@/lib/api/api-error";
import type { ApiErrorEnvelope } from "@/lib/api/contracts";

export type ApiRequestOptions<TBody> = Omit<
  RequestInit,
  "body" | "credentials" | "headers"
> & {
  accessToken?: string;
  body?: TBody;
  headers?: HeadersInit;
  idempotencyKey?: string;
};

export async function apiRequest<TResponse, TBody = never>(
  path: `/${string}`,
  options: ApiRequestOptions<TBody> = {},
): Promise<TResponse> {
  const {
    accessToken,
    body,
    headers: customHeaders,
    idempotencyKey,
    ...requestInit
  } = options;
  const headers = new Headers(customHeaders);
  headers.set("Accept", "application/json");

  if (body !== undefined) {
    headers.set("Content-Type", "application/json");
  }
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }
  if (idempotencyKey) {
    headers.set("Idempotency-Key", idempotencyKey);
  }

  let response: Response;
  try {
    const fetchOptions: RequestInit = {
      ...requestInit,
      credentials: "include",
      headers,
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    };
    response = await fetch(
      new URL(path, publicEnvironment.NEXT_PUBLIC_API_URL),
      fetchOptions,
    );
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw error;
    }
    throw new ApiError({
      code: "network_error",
      message: "We could not connect to Monierave.",
      status: 0,
    });
  }

  if (!response.ok) {
    throw apiErrorFromResponse(
      response.status,
      await readJSON<ApiErrorEnvelope>(response),
      response.headers,
    );
  }

  if (response.status === 204) {
    return undefined as TResponse;
  }

  const payload = await readJSON<TResponse>(response);
  if (payload === null) {
    throw new ApiError({
      code: "invalid_response",
      message: "Monierave returned an unexpected response.",
      status: response.status,
    });
  }
  return payload;
}

async function readJSON<T>(response: Response): Promise<T | null> {
  if (!response.headers.get("Content-Type")?.includes("application/json")) {
    return null;
  }
  try {
    return (await response.json()) as T;
  } catch {
    return null;
  }
}
