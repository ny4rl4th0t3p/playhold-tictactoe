# Two images, from the folder with go.mod: the game server (--target server), and the page on Caddy, which also ends
# TLS in front of the server (--target page). compose.prod.yaml builds both. DIR is where the game is inside the build's
# folder: "." in a project of its own, where go.mod is next to this file.
ARG DIR=.

# The page, built with the address of the game's server, which it can't learn at run time: it's only files.
FROM node:24-alpine AS page-build
ARG DIR
ARG GAME_ADDRESS
# The TypeScript SDK installs from its GitHub repository, which npm fetches with git.
RUN apk add --no-cache git
WORKDIR /src
COPY ${DIR}/web/package.json ${DIR}/web/package-lock.json web/
RUN npm --prefix web ci
COPY ${DIR}/web web
RUN test -n "${GAME_ADDRESS}" && VITE_SERVER_URL="wss://${GAME_ADDRESS}/ws" npm --prefix web run build

FROM golang:1.27.4-alpine AS server-build
ARG DIR
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
# At the same path as in the build's folder, since Go names a package by where it is under go.mod.
COPY ${DIR} ${DIR}
ENV CGO_ENABLED=0
RUN go build -trimpath -o /out/server ./${DIR}/server

# The game server: one binary, nothing else, as a user with no rights.
FROM scratch AS server
COPY --from=server-build /out/server /server
USER 65534
EXPOSE 8080
ENTRYPOINT ["/server"]

# The page, and the proxy in front of the server, in one Caddy image.
FROM caddy:2.10-alpine AS page
ARG DIR
COPY ${DIR}/Caddyfile /etc/caddy/Caddyfile
COPY --from=page-build /src/web/dist /srv/page