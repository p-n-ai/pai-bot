import { render } from "@testing-library/react";
import { vi } from "vitest";
import Aurora from "@/components/Aurora";

const oglState = vi.hoisted(() => ({
  rendererCreations: 0,
  loseContext: vi.fn(),
}));

vi.mock("ogl", () => {
  class Renderer {
    gl = {
      BLEND: 1,
      ONE: 1,
      ONE_MINUS_SRC_ALPHA: 1,
      canvas: document.createElement("canvas"),
      clearColor: vi.fn(),
      enable: vi.fn(),
      blendFunc: vi.fn(),
      getExtension: vi.fn(() => ({ loseContext: oglState.loseContext })),
    };

    constructor() {
      oglState.rendererCreations += 1;
    }

    setSize = vi.fn();
    render = vi.fn();
  }

  class Triangle {
    attributes = { uv: true };
  }

  class Program {
    uniforms: Record<string, { value: unknown }>;

    constructor(_gl: unknown, options: { uniforms: Record<string, { value: unknown }> }) {
      this.uniforms = options.uniforms;
    }
  }

  class Mesh {}

  class Color {
    r = 1;
    g = 1;
    b = 1;
  }

  return {
    Color,
    Mesh,
    Program,
    Renderer,
    Triangle,
  };
});

describe("Aurora", () => {
  beforeEach(() => {
    oglState.rendererCreations = 0;
    oglState.loseContext.mockReset();
    vi.stubGlobal("requestAnimationFrame", vi.fn(() => 1));
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not recreate the renderer when rerendered with equal color stops", () => {
    const { rerender } = render(
      <Aurora colorStops={["#0ea5e9", "#2563eb", "#06101d"]} amplitude={1.15} blend={0.42} speed={0.9} />,
    );

    rerender(<Aurora colorStops={["#0ea5e9", "#2563eb", "#06101d"]} amplitude={1.15} blend={0.42} speed={0.9} />);

    expect(oglState.rendererCreations).toBe(1);
    expect(oglState.loseContext).not.toHaveBeenCalled();
  });
});
