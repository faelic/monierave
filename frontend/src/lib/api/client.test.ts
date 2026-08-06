import { HttpResponse, http } from "msw";
import { describe, expect, it } from "vitest";

import { apiRequest } from "@/lib/api/client";
import { mockServer } from "@/test/mocks/server";

describe("apiRequest", () => {
  it("uses credentialed requests and sends an access token", async () => {
    mockServer.use(
      http.get("http://localhost:8080/users/me", ({ request }) => {
        expect(request.credentials).toBe("include");
        expect(request.headers.get("Authorization")).toBe(
          "Bearer access-token",
        );
        return HttpResponse.json({ username: "favour" });
      }),
    );

    const response = await apiRequest<{ username: string }>("/users/me", {
      accessToken: "access-token",
    });

    expect(response.username).toBe("favour");
  });

  it("normalizes stable backend errors", async () => {
    mockServer.use(
      http.post("http://localhost:8080/accounts/resolve", () =>
        HttpResponse.json(
          {
            code: "recipient_not_found",
            message: "recipient not found",
            request_id: "request-123",
          },
          { status: 404 },
        ),
      ),
    );

    const request = apiRequest("/accounts/resolve", {
      method: "POST",
      body: { account_number: "4839201756" },
    });

    await expect(request).rejects.toMatchObject({
      code: "recipient_not_found",
      requestID: "request-123",
      status: 404,
    });
  });

  it("forwards idempotency keys without persisting them", async () => {
    mockServer.use(
      http.post("http://localhost:8080/transfers", ({ request }) => {
        expect(request.headers.get("Idempotency-Key")).toBe("transfer-key");
        return HttpResponse.json({ fee: 0 }, { status: 201 });
      }),
    );

    const response = await apiRequest<{ fee: number }, { amount: number }>(
      "/transfers",
      {
        body: { amount: 5000 },
        idempotencyKey: "transfer-key",
        method: "POST",
      },
    );

    expect(response.fee).toBe(0);
  });
});
