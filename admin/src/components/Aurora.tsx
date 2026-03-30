"use client";

import { useEffect, useRef, type CSSProperties } from "react";
import { Color, Mesh, Program, Renderer, Triangle } from "ogl";
import { cn } from "@/lib/utils";

const DEFAULT_COLOR_STOPS = ["#5227FF", "#7cff67", "#5227FF"];

function toColorStopValues(colorStops: string[]) {
  return colorStops.map((hex) => {
    const color = new Color(hex);
    return [color.r, color.g, color.b];
  });
}

function getColorStopsKey(colorStops: string[]) {
  return colorStops.join("|");
}

const VERT = `#version 300 es
in vec2 position;
void main() {
  gl_Position = vec4(position, 0.0, 1.0);
}
`;

const FRAG = `#version 300 es
precision highp float;

uniform float uTime;
uniform float uAmplitude;
uniform vec3 uColorStops[3];
uniform vec2 uResolution;
uniform float uBlend;
uniform float uDrift;

out vec4 fragColor;

vec3 permute(vec3 x) {
  return mod(((x * 34.0) + 1.0) * x, 289.0);
}

float snoise(vec2 v){
  const vec4 C = vec4(
      0.211324865405187, 0.366025403784439,
      -0.577350269189626, 0.024390243902439
  );
  vec2 i  = floor(v + dot(v, C.yy));
  vec2 x0 = v - i + dot(i, C.xx);
  vec2 i1 = (x0.x > x0.y) ? vec2(1.0, 0.0) : vec2(0.0, 1.0);
  vec4 x12 = x0.xyxy + C.xxzz;
  x12.xy -= i1;
  i = mod(i, 289.0);

  vec3 p = permute(
      permute(i.y + vec3(0.0, i1.y, 1.0))
    + i.x + vec3(0.0, i1.x, 1.0)
  );

  vec3 m = max(
      0.5 - vec3(
          dot(x0, x0),
          dot(x12.xy, x12.xy),
          dot(x12.zw, x12.zw)
      ), 
      0.0
  );
  m = m * m;
  m = m * m;

  vec3 x = 2.0 * fract(p * C.www) - 1.0;
  vec3 h = abs(x) - 0.5;
  vec3 ox = floor(x + 0.5);
  vec3 a0 = x - ox;
  m *= 1.79284291400159 - 0.85373472095314 * (a0*a0 + h*h);

  vec3 g;
  g.x  = a0.x  * x0.x  + h.x  * x0.y;
  g.yz = a0.yz * x12.xz + h.yz * x12.yw;
  return 130.0 * dot(m, g);
}

struct ColorStop {
  vec3 color;
  float position;
};

#define COLOR_RAMP(colors, factor, finalColor) {              \
  int index = 0;                                            \
  for (int i = 0; i < 2; i++) {                               \
     ColorStop currentColor = colors[i];                    \
     bool isInBetween = currentColor.position <= factor;    \
     index = int(mix(float(index), float(i), float(isInBetween))); \
  }                                                         \
  ColorStop currentColor = colors[index];                   \
  ColorStop nextColor = colors[index + 1];                  \
  float range = nextColor.position - currentColor.position; \
  float lerpFactor = (factor - currentColor.position) / range; \
  finalColor = mix(currentColor.color, nextColor.color, lerpFactor); \
}

void main() {
  vec2 uv = gl_FragCoord.xy / uResolution;
  
  ColorStop colors[3];
  colors[0] = ColorStop(uColorStops[0], 0.0);
  colors[1] = ColorStop(uColorStops[1], 0.5);
  colors[2] = ColorStop(uColorStops[2], 1.0);
  
  vec3 rampColor;
  float rampFactor = fract(uv.x + (uTime * uDrift));
  COLOR_RAMP(colors, rampFactor, rampColor);
  
  float height = snoise(vec2(uv.x * 2.0 + uTime * 0.1, uTime * 0.25)) * 0.5 * uAmplitude;
  height = exp(height);
  height = (uv.y * 2.0 - height + 0.2);
  float intensity = 0.6 * height;
  
  float midPoint = 0.20;
  float auroraAlpha = smoothstep(midPoint - uBlend * 0.5, midPoint + uBlend * 0.5, intensity);
  
  vec3 auroraColor = intensity * rampColor;
  
  fragColor = vec4(auroraColor * auroraAlpha, auroraAlpha);
}
`;

interface AuroraProps {
  className?: string;
  colorStops?: string[];
  amplitude?: number;
  blend?: number;
  drift?: number;
  time?: number;
  speed?: number;
  style?: CSSProperties;
}

export default function Aurora(props: AuroraProps) {
  const propsRef = useRef<AuroraProps>(props);
  const ctnDom = useRef<HTMLDivElement>(null);

  useEffect(() => {
    propsRef.current = props;
  }, [props]);

  useEffect(() => {
    const ctn = ctnDom.current;
    if (!ctn) return;
    const initialProps = propsRef.current;
    const initialColorStops = initialProps.colorStops ?? DEFAULT_COLOR_STOPS;
    const initialAmplitude = initialProps.amplitude ?? 1.0;
    const initialBlend = initialProps.blend ?? 0.5;
    const initialDrift = initialProps.drift ?? 0.0;

    const renderer = new Renderer({
      alpha: true,
      premultipliedAlpha: true,
      antialias: true,
    });
    const gl = renderer.gl;
    gl.clearColor(0, 0, 0, 0);
    gl.enable(gl.BLEND);
    gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);
    gl.canvas.style.backgroundColor = "transparent";

    function resize() {
      if (!ctn) return;
      const width = ctn.offsetWidth;
      const height = ctn.offsetHeight;
      renderer.setSize(width, height);
      program.uniforms.uResolution.value = [width, height];
    }
    window.addEventListener("resize", resize);

    const geometry = new Triangle(gl);
    if (geometry.attributes.uv) {
      delete geometry.attributes.uv;
    }

    const program = new Program(gl, {
      vertex: VERT,
      fragment: FRAG,
      uniforms: {
        uTime: { value: 0 },
        uAmplitude: { value: initialAmplitude },
        uColorStops: { value: toColorStopValues(initialColorStops) },
        uResolution: { value: [ctn.offsetWidth, ctn.offsetHeight] },
        uBlend: { value: initialBlend },
        uDrift: { value: initialDrift },
      },
    });

    const mesh = new Mesh(gl, { geometry, program });
    ctn.appendChild(gl.canvas);

    let animateId = 0;
    let colorStopsKey = getColorStopsKey(initialColorStops);
    const update = (t: number) => {
      animateId = requestAnimationFrame(update);
      const nextProps = propsRef.current;
      const nextTime = nextProps.time ?? t * 0.01;
      const nextSpeed = nextProps.speed ?? 1.0;
      const nextAmplitude = nextProps.amplitude ?? 1.0;
      const nextBlend = nextProps.blend ?? 0.5;
      const nextDrift = nextProps.drift ?? 0.0;
      const nextColorStops = nextProps.colorStops ?? DEFAULT_COLOR_STOPS;
      const nextColorStopsKey = getColorStopsKey(nextColorStops);

      program.uniforms.uTime.value = nextTime * nextSpeed * 0.1;
      program.uniforms.uAmplitude.value = nextAmplitude;
      program.uniforms.uBlend.value = nextBlend;
      program.uniforms.uDrift.value = nextDrift;

      if (nextColorStopsKey !== colorStopsKey) {
        program.uniforms.uColorStops.value = toColorStopValues(nextColorStops);
        colorStopsKey = nextColorStopsKey;
      }

      renderer.render({ scene: mesh });
    };
    animateId = requestAnimationFrame(update);

    resize();

    return () => {
      cancelAnimationFrame(animateId);
      window.removeEventListener("resize", resize);
      if (ctn && gl.canvas.parentNode === ctn) {
        ctn.removeChild(gl.canvas);
      }
      gl.getExtension("WEBGL_lose_context")?.loseContext();
    };
  }, []);

  return <div ref={ctnDom} className={cn("size-full", props.className)} style={props.style} />;
}
