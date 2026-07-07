import js from '@eslint/js';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import prettierConfig from 'eslint-config-prettier';

export default [
  { ignores: ['dist', 'node_modules'] },
  js.configs.recommended,
  {
    files: ['**/*.{js,jsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: { ...globals.browser, ...globals.node },
      parserOptions: {
        ecmaFeatures: { jsx: true },
      },
    },
    settings: {
      react: { version: 'detect' },
    },
    plugins: {
      react,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...react.configs.recommended.rules,
      ...react.configs['jsx-runtime'].rules,
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
      // Project does not use the `prop-types` package anywhere (not part of
      // .claude/docs/standard-libraries.md) — runtime prop validation is not
      // part of this project's conventions, so this rule is pure noise here.
      'react/prop-types': 'off',
    },
  },
  {
    // Test files commonly import `describe` from vitest purely to group tests
    // in some files and not others; unused-var linting on test-only imports
    // is not meaningful here and must not force edits to test files (tests
    // are the RED/GREEN contract and must not be touched to satisfy lint).
    files: ['**/*.test.{js,jsx}'],
    rules: {
      'no-unused-vars': 'off',
    },
  },
  prettierConfig,
];
