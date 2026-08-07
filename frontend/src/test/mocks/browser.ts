import { setupWorker } from "msw/browser";

import { handlers } from "@/test/mocks/handlers";

export const mockWorker = setupWorker(...handlers);
