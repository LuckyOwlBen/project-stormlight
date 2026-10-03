# project-stormlight
I just learned how much better Go would be for this kind of thing so I figured I'd try it out.

## Development Instructions

Here are some basic commands to help you operate, build, and test the Go application:

- **Run the application:** `go run cmd/stormlight/main.go`
- **Generate UI templates:** `templ generate`
- **Build Tailwind CSS:** `npx @tailwindcss/cli -i assets/css/input.css -o assets/css/output.css` (add `--watch` for auto-rebuild during dev)
- **Run all tests:** `go test ./...`
- **Build the executable:** `go build -o stormlight.exe ./cmd/stormlight`
- **Format your Go code:** `go fmt ./...`
- **Update dependencies:** `go mod tidy`

---

## Configuration & Deployment

The app needs a PostgreSQL database and reads its settings from environment variables (or a `.env` file). Copy [.env.example](./.env.example) to `.env` and fill it in.

| Variable | Purpose |
| --- | --- |
| `DATABASE_URL` | Full Postgres connection string. If set, the `POSTGRES_*` values are ignored. |
| `POSTGRES_HOST` / `PORT` / `USER` / `PASSWORD` / `DB` / `SCHEMA` | Connection parts used when `DATABASE_URL` is unset (credentials are URL-escaped for you). |
| `POSTGRES_SSLMODE` | Postgres `sslmode`, default `disable`. Use `require` for hosted databases. |
| `SESSION_SECRET` | Signs session cookies. Set a long random value in production; if unset, a random key is used and sessions reset on every restart. |
| `COOKIE_SECURE` | Set to `true` when serving over HTTPS. |
| `GM_SECRET` | Secret needed to register a GM account at `/register/gm`. |
| `PORT` | HTTP port, default `3000`. |
| `ALLOWED_ORIGINS` | Extra hosts allowed to open WebSockets (the app's own host is always allowed). |

Tables are created automatically on startup. `GET /healthz` returns `ok` when the database is reachable, for load balancer or host health checks. The server shuts down gracefully on `SIGINT`/`SIGTERM`.

Build the CSS and templates before building a release:

```
templ generate
npx @tailwindcss/cli -i assets/css/input.css -o assets/css/output.css --minify
go build -o stormlight ./cmd/stormlight
```

Talent, path, and spren data is validated at startup; the server refuses to start if a talent references something that doesn't exist.

---

## Tech Stack & UI

This project uses a modern Go-based hypermedia stack for the frontend:
- **[templ](https://templ.guide/):** HTML templating language for Go.
- **[HTMX](https://htmx.org/):** High-power tools for HTML (access AJAX, CSS Transitions, WebSockets, and Server Sent Events directly in HTML).
- **[Tailwind CSS (v4)](https://tailwindcss.com/):** Utility-first CSS framework.
- **[daisyUI](https://daisyui.com/):** Tailwind CSS component library.

### Editor Setup (VS Code)

To get the best developer experience, please install the following VS Code extensions:
1. **`templ`** (`a-h.templ`): Provides syntax highlighting, formatting, and auto-complete for `.templ` files.
2. **Tailwind CSS IntelliSense** (`bradlc.vscode-tailwindcss`): Enables autocomplete for Tailwind and daisyUI classes directly inside templates. Note: The `.vscode/settings.json` is already configured to make Tailwind autocomplete work with `templ` files via `"tailwindCSS.includeLanguages": { "templ": "html" }`.
