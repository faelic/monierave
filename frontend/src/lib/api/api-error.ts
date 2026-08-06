import type { ApiErrorEnvelope } from "@/lib/api/contracts";

export class ApiError extends Error {
  readonly code: string;
  readonly requestID?: string;
  readonly retryAfterSeconds?: number;
  readonly status: number;

  constructor({
    code,
    message,
    requestID,
    retryAfterSeconds,
    status,
  }: {
    code: string;
    message: string;
    requestID?: string;
    retryAfterSeconds?: number;
    status: number;
  }) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    if (requestID !== undefined) {
      this.requestID = requestID;
    }
    if (retryAfterSeconds !== undefined) {
      this.retryAfterSeconds = retryAfterSeconds;
    }
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export function apiErrorFromResponse(
  status: number,
  body: ApiErrorEnvelope | null,
  headers: Headers,
) {
  const retryAfter = Number(headers.get("Retry-After"));
  const requestID =
    body?.request_id ?? headers.get("X-Request-ID") ?? undefined;

  return new ApiError({
    code: body?.code ?? "request_failed",
    message: body?.message ?? "The request could not be completed.",
    status,
    ...(requestID ? { requestID } : {}),
    ...(Number.isFinite(retryAfter) && retryAfter > 0
      ? { retryAfterSeconds: retryAfter }
      : {}),
  });
}
