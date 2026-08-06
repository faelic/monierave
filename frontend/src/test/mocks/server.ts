import { setupServer } from "msw/node";

import { handlers } from "@/test/mocks/handlers";

export const mockServer = setupServer(...handlers);
