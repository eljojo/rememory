# Hosting ReMemory

There are two ways to host ReMemory for your friends: **static pages** (simplest) and a **self-hosted server** (full-featured).

## Static pages

The lightest option. Generate a folder with `recover.html` and `MANIFEST.age`, then upload it anywhere that serves files — GitHub Pages, Netlify, an S3 bucket, any web server.

```bash
rememory seal --pages
# or after sealing:
rememory bundle --pages
```

This creates `output/pages/` in your project. The `recover.html` page fetches `MANIFEST.age` from the same directory automatically. Friends visit the URL, add their shares, and recover. No server-side code runs — it's just static files.

Works well when:
- You want to give friends a URL instead of (or alongside) a ZIP file
- You don't need the ability to create bundles from the browser
- You don't want to run a server

Limitations:
- Friends still need their shares (from their bundles or README.txt files)
- No admin interface — you manage files directly
- No bundle creation in the browser (use the CLI or [maker.html](https://eljojo.github.io/rememory/maker.html))

## Self-hosted server

Run ReMemory as a web app on your own server — create bundles, store encrypted archives, and recover, all from a browser.

### Docker

A pre-built image is published to GitHub Container Registry on every release.

```bash
docker run -d \
  --name rememory \
  -p 127.0.0.1:8080:8080 \
  -v rememory-data:/data \
  ghcr.io/eljojo/rememory:latest
```

The examples publish the port on `127.0.0.1`, so only this machine can reach it; to share it, put the [reverse proxy](#reverse-proxy) below in front.

Because the container listens on every interface, the server chooses its admin password at first start and prints it once. Find it in the logs:

```bash
docker logs rememory
```

To choose the password yourself, set `REMEMORY_ADMIN_PASSWORD` (it replaces a stored one, so changing it later works too). Without Docker, `rememory serve` listens on `127.0.0.1` and the first page asks you to choose the password.

To pin a specific version:

```bash
docker run -d \
  --name rememory \
  -p 127.0.0.1:8080:8080 \
  -v rememory-data:/data \
  ghcr.io/eljojo/rememory:v0.0.16
```

**Docker Compose:**

```yaml
services:
  rememory:
    image: ghcr.io/eljojo/rememory:latest
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - rememory-data:/data
    restart: unless-stopped
    # environment:
    #   REMEMORY_MAX_MANIFEST_SIZE: 200MB
    #   REMEMORY_ADMIN_PASSWORD: choose-a-long-password

volumes:
  rememory-data:
```

The container is a single static binary with no dependencies. Data lives in `/data` — mount a volume there to persist across restarts.

### Without Docker

If you have the CLI installed:

```bash
rememory serve
```

### Options

| Flag | Env var | Default | Description |
|------|--------|---------|-------------|
| `--port, -p` | `REMEMORY_PORT` | `8080` | Port to listen on |
| `--host` | `REMEMORY_HOST` | `127.0.0.1` | Host to bind to |
| `--data, -d` | `REMEMORY_DATA` | `./rememory-data` | Data directory for bundles and config |
| `--max-manifest-size` | `REMEMORY_MAX_MANIFEST_SIZE` | `50MB` | Maximum MANIFEST.age size (e.g. `50MB`, `1GB`) |
| `--open-uploads` | `REMEMORY_OPEN_UPLOADS` | `false` | Let anyone who can reach the server create bundles, without the admin password |
| | `REMEMORY_ADMIN_PASSWORD` | | Admin password. If unset, a server listening beyond `127.0.0.1` generates one and prints it once |

Flags take precedence over environment variables.

## Deployment

### Reverse proxy

Put the server behind a reverse proxy with TLS. Set `REMEMORY_ADMIN_PASSWORD` before you expose it: the setup page refuses requests that come through a proxy, so no visitor can choose the password. A proxy that adds no forwarding headers (`X-Forwarded-For`, `X-Real-IP`, `Forwarded`) can't be told apart, so don't skip this step.

**Caddy:**
```
rememory.example.com {
    reverse_proxy localhost:8080
}
```

**nginx:**
```nginx
server {
    listen 443 ssl;
    server_name rememory.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        client_max_body_size 100M;
    }
}
```

### Authentication

The admin password protects creating and deleting bundles. Your browser asks for it when you open the maker; any user name works. Recovery pages stay open, so friends never need it. To let anyone who can reach the server create bundles, start it with `--open-uploads`.

For access control beyond that, use an auth proxy:

- [Authelia](https://www.authelia.com/)
- [Cloudflare Access](https://www.cloudflare.com/products/zero-trust/access/)
- [Pocket ID](https://github.com/pocket-id/pocket-id)
- [OAuth2 Proxy](https://oauth2-proxy.github.io/oauth2-proxy/)

## Security

- The server stores only encrypted archives (MANIFEST.age). Without enough shares, the archive is useless.
- Shares are never sent to the server. They stay in each friend's bundle.
- The admin password uses age's scrypt-based passphrase encryption. Choose a strong one.
- Put the server behind HTTPS and authentication appropriate for your threat model.

Friends still get self-contained offline bundles. The server is a convenience — if it goes away, they can recover without it.

## Data directory

The data directory contains:

```
rememory-data/
  admin.age               # Admin password (age-encrypted)
  bundles/
    <uuid>/
      meta.json           # Non-secret metadata
      MANIFEST.age        # Encrypted archive
```

Back up this directory to preserve your encrypted archives. The admin.age file can be recreated by setting a new password (you'd lose the ability to delete existing bundles with the old password, but the bundles themselves are unaffected).
