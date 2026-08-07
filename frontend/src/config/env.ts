import { z } from "zod";

const publicEnvironmentSchema = z.object({
  NEXT_PUBLIC_API_URL: z.url(),
});

export const publicEnvironment = publicEnvironmentSchema.parse({
  NEXT_PUBLIC_API_URL:
    process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080",
});
