import { render } from "@testing-library/react";
import { vi } from "vitest";
import Aurora from "@/components/Aurora";

const oglState = vi.hoisted(() => ({
  rendererCreations: 0,
  loseContext: vi.fn(),
  setSize: vi.fn(),
}));

const resizeObserverState = vi.hoisted(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  callback: null as ResizeObserverCallback | null,
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

    setSize = oglState.setSize;
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
    oglState.setSize.mockReset();
    resizeObserverState.observe.mockReset();
    resizeObserverState.disconnect.mockReset();
    resizeObserverState.callback = null;
    vi.stubGlobal("requestAnimationFrame", vi.fn(() => 1));
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
    vi.stubGlobal(
      "ResizeObserver",
      class ResizeObserver {
        constructor(callback: ResizeObserverCallback) {
          resizeObserverState.callback = callback;
        }

        observe = resizeObserverState.observe;
        disconnect = resizeObserverState.disconnect;
      },
    );
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

  it("resizes when the container becomes visible after mount", () => {
    let width = 0;
    let height = 0;
    const widthSpy = vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockImplementation(() => width);
    const heightSpy = vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(() => height);

    try {
      const { unmount } = render(
        <Aurora colorStops={["#43aedf", "#057aff", "#ffffff"]} amplitude={0.96} blend={0.39} speed={0.97} />,
      );

      expect(resizeObserverState.observe).toHaveBeenCalled();

      width = 560;
      height = 420;
      resizeObserverState.callback?.([], {} as ResizeObserver);

      expect(oglState.setSize).toHaveBeenLastCalledWith(560, 420);

      unmount();

      expect(resizeObserverState.disconnect).toHaveBeenCalled();
    } finally {
      widthSpy.mockRestore();
      heightSpy.mockRestore();
    }
  });
});
