# syntax=docker/dockerfile:1

# Build the Tailwind/daisyUI stylesheet
FROM docker.io/library/node:22-alpine AS css
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY assets/css/input.css assets/css/input.css
COPY internal internal
RUN npx @tailwindcss/cli -i assets/css/input.css -o assets/css/output.css --minify

# Build the Go binary (templ *_templ.go files are committed)
FROM docker.io/library/golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY data data
COPY internal internal
COPY pkg pkg
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/stormlight ./cmd/stormlight

FROM docker.io/library/alpine:3.22
RUN adduser -D -u 10001 stormlight
WORKDIR /app
COPY --from=build /out/stormlight /app/stormlight
# The server serves static files from ./assets at runtime
COPY --from=css /src/assets/css/output.css /app/assets/css/output.css
USER stormlight
EXPOSE 3000
ENTRYPOINT ["/app/stormlight"]
