import eslint from "@eslint/js";
import vitestPlugin from "@vitest/eslint-plugin";
import prettierConfig from "eslint-config-prettier/flat";
import importPlugin from "eslint-plugin-import";
import jsxA11yPlugin from "eslint-plugin-jsx-a11y";
import promisePlugin from "eslint-plugin-promise";
import reactPlugin from "eslint-plugin-react";
import reactHooksPlugin from "eslint-plugin-react-hooks";
import reactRefreshPlugin from "eslint-plugin-react-refresh";
import { defineConfig, globalIgnores } from "eslint/config";
import globals from "globals";
import tseslint from "typescript-eslint";

export default defineConfig([
  globalIgnores([
    "**/.cache/**",
    "**/.env.*",
    "**/.env",
    "**/.react-router/**",
    "**/.vite/**",
    "**/*.config.{js,mjs,cjs,ts}",
    "**/build/**",
    "**/coverage/**",
    "**/dist/**",
    "**/node_modules/**",
    "**/public/**",
  ]),
  eslint.configs.recommended,
  promisePlugin.configs["flat/recommended"],
  reactPlugin.configs.flat.recommended,
  reactPlugin.configs.flat["jsx-runtime"],
  reactHooksPlugin.configs.flat.recommended,
  jsxA11yPlugin.flatConfigs.recommended,
  reactRefreshPlugin.configs.recommended,
  reactRefreshPlugin.configs.vite,
  tseslint.configs.recommendedTypeChecked,
  vitestPlugin.configs.recommended,
  {
    settings: {
      react: {
        version: "detect",
      },
    },
    languageOptions: {
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
        // @ts-ignore
        tsconfigRootDir: import.meta.dirname,
        projectService: {
          allowDefaultProject: ["*.js", "*.mjs", "*.cjs"],
          noWarnOnMultipleProjects: true,
        },
      },
    },
  },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [importPlugin.flatConfigs.recommended, importPlugin.flatConfigs.typescript],
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
          project: "{apps,packages}/*/tsconfig.json",
        },
      },
    },
    rules: {
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          args: "all",
          argsIgnorePattern: "^_",
          caughtErrors: "all",
          caughtErrorsIgnorePattern: "^_",
          destructuredArrayIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          ignoreRestSiblings: true,
        },
      ],
      "no-empty-pattern": "warn",
      "jsx-a11y/no-autofocus": "off",
      "react-refresh/only-export-components": [
        "warn",
        { allowExportNames: ["meta", "links", "headers", "loader", "action", "clientLoader", "clientAction"] },
      ],
    },
  },
  {
    files: ["**/*.{js,mjs,cjs,jsx}"],
    extends: [tseslint.configs.disableTypeChecked],
  },
  {
    files: ["**/*.cjs"],
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
    rules: {
      "@typescript-eslint/no-require-imports": "off",
    },
  },
  prettierConfig,
]);
