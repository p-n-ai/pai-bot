import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const bannedUseEffectRules = {
  "no-restricted-imports": [
    "error",
    {
      paths: [
        {
          name: "react",
          importNames: ["useEffect"],
          message: "useEffect is banned here. Prefer composition, event handlers, useSyncExternalStore, or an explicit hook abstraction.",
        },
      ],
    },
  ],
  "no-restricted-properties": [
    "error",
    {
      object: "React",
      property: "useEffect",
      message: "React.useEffect is banned here. Prefer composition, event handlers, useSyncExternalStore, or an explicit hook abstraction.",
    },
  ],
};

const legacyUseEffectFiles = [
  "src/components/admin-shell.tsx",
  "src/components/Aurora.tsx",
  "src/components/PixelBlast.jsx",
  "src/components/ShapeGrid.tsx",
  "src/components/theme-provider.tsx",
  "src/components/ui/calendar.tsx",
  "src/components/ui/carousel.tsx",
  "src/components/ui/sidebar.tsx",
  "src/hooks/use-async-resource.ts",
  "src/hooks/use-auth-redirect-notice.ts",
  "src/hooks/use-admin-session-bootstrap.ts",
  "src/hooks/use-mobile.ts",
  "src/hooks/use-session-redirect.ts",
];

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    files: [
      "src/app/**/*.{js,jsx,ts,tsx}",
      "src/components/*.{js,jsx,ts,tsx}",
      "src/components/login-gate/**/*.{js,jsx,ts,tsx}",
      "src/hooks/*.{js,jsx,ts,tsx}",
    ],
    rules: bannedUseEffectRules,
  },
  {
    files: legacyUseEffectFiles,
    rules: {
      "no-restricted-imports": "off",
      "no-restricted-properties": "off",
      "react-hooks/exhaustive-deps": "off",
      "react-hooks/set-state-in-effect": "off",
    },
  },
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
]);

export default eslintConfig;
