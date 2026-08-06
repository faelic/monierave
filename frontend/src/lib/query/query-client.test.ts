import { ApiError } from "@/lib/api/api-error";
import { createQueryClient } from "@/lib/query/query-client";
import { describe, expect, it } from "vitest";

describe("createQueryClient", () => {
  it("does not retry client or authorization errors", () => {
    const client = createQueryClient();
    const retry = client.getDefaultOptions().queries?.retry;

    expect(typeof retry).toBe("function");
    expect(
      (retry as (count: number, error: Error) => boolean)(
        0,
        new ApiError({
          code: "recipient_not_found",
          message: "recipient not found",
          status: 404,
        }),
      ),
    ).toBe(false);
  });

  it("never retries mutations by default", () => {
    const client = createQueryClient();

    expect(client.getDefaultOptions().mutations?.retry).toBe(false);
  });
});
