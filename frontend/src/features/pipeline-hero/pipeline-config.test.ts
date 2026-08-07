import { describe, expect, it } from "vitest";

import {
  pipelineBrand,
  pipelineCamera,
  pipelinePaths,
} from "./pipeline-config";

describe("pipeline hero configuration", () => {
  it("keeps all editable paths rich enough for smooth curves", () => {
    for (const points of Object.values(pipelinePaths)) {
      expect(points.length).toBeGreaterThanOrEqual(5);
    }
  });

  it("uses the canonical static camera and centralized materials", () => {
    expect(pipelineCamera.zoom).toBeGreaterThan(80);
    expect(pipelineCamera.zoom).toBeLessThan(120);
    expect(pipelineBrand.background).toBe("#050505");
    expect(pipelineBrand.silver).not.toBe(pipelineBrand.graphite);
  });
});
