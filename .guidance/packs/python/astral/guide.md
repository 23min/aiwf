# When working in Python

**Version & house style:** read `pyproject.toml` (`requires-python`) for the version and use
version-appropriate idioms; the repo's own conventions win.

## Toolchain — the Astral stack
- **`uv`** for everything (deps, venvs, running tools) — never pip/venv directly.
- **`ruff`** for both format and lint (`ruff format` + `ruff check`); enable `pyupgrade` and
  `flake8-bugbear` rules. Not black/flake8.
- **`ty`** (Astral) or **`mypy --strict`** for type-checking. Type hints everywhere.

## Testing
- **`pytest`**; `tmp_path` for filesystem work, fixtures in `conftest.py`.
- **Hypothesis** for property tests; **golden files** for extraction/regression.

## Idioms (added best practice — prune any you don't want)
- `pathlib.Path` over `os.path`; atomic file writes via temp + `os.replace`.
- **Dataclasses or Pydantic** for records crossing a boundary — not `dict[str, Any]`.
- Context managers (`with`) for every resource. Never a mutable default argument.
- `if __name__ == "__main__":` guard; prefer comprehensions/generators where they read
  clearly.
- Prefer small, composable CLIs (callable as tools) over one monolithic script; run tooling
  inside the devcontainer.
