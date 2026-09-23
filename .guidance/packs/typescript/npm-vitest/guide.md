# When working in TypeScript

**Version & house style:** read `tsconfig.json` (target, strictness) and `package.json`
(`engines`, `type`) for the project's settings; the repo's own conventions win.

## Toolchain
- **npm** (not pnpm/yarn/bun). **Vitest** for unit tests; **Playwright** for E2E / visual
  regression. ESLint + Prettier where configured.

## Type discipline (added best practice — prune any you don't want)
- `strict: true` plus `noUncheckedIndexedAccess`. **`unknown` over `any`**; narrow before use.
- Shapes crossing a boundary are named types validated on read (e.g. **zod**) — **no blind
  `as Foo` casts on external data**.
- **Discriminated unions** for variants; `readonly` / `as const` for immutable data;
  `satisfies` to check a literal against a type without widening it.
- ⚖️ Union literal types over `enum`. ⚖️ Named exports over default exports.

## Structure
- Keep parsers and core logic as **pure functions**. Keep a platform-neutral core free of
  runtime-specific imports (`node:`, `cloudflare:`); push I/O to the edges.
