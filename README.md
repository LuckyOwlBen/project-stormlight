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
