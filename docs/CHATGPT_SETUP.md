# ChatGPT Developer Mode Setup

Verified on 2026-05-14 against OpenAI's current ChatGPT Developer Mode and MCP documentation.

## Prerequisites

- Go installed
- `cloudflared` installed locally
- A ChatGPT plan/workspace that can create custom MCP apps in Developer Mode
- A configured Project Brain root in `configs/project-brain.yml`

## 1. Build

```powershell
go build -o server.exe ./cmd/server
```

## 2. Configure

Copy the example config before editing secrets:

```powershell
Copy-Item configs/project-brain.example.yml configs/project-brain.yml
```

Set:

- `roots[].path` to the project parent directories ChatGPT may inspect.
- `auth.type` to `oauth` for any tunnel-exposed server.
- `auth.owner_secret` and `auth.jwt_secret` to long random values.

Development-only `noauth_dev` is acceptable only when the endpoint is not publicly reachable.

## 3. Start With a Quick Tunnel

```powershell
.\launch.bat
```

The launcher starts `cloudflared`, detects the temporary `trycloudflare.com` URL, and starts Project Brain with that URL as the OAuth issuer.

Use the printed MCP endpoint:

```text
https://<temporary-name>.trycloudflare.com/mcp/
```

Quick Tunnels are for development and testing. The URL can change between launches and Cloudflare does not provide uptime guarantees for TryCloudflare. If you need stable day-to-day usage, create a named Cloudflare Tunnel with your own hostname and use that stable HTTPS URL as `server.public_base_url`, `auth.issuer_url`, and the ChatGPT MCP endpoint base.

## 4. Add the App in ChatGPT

In ChatGPT web:

1. Open Settings.
2. Enable Developer Mode from Apps advanced settings if available for your account/workspace.
3. Create a custom app from the MCP server URL.
4. Use the printed `/mcp/` HTTPS endpoint.
5. Select OAuth unless you are doing a private local-only test.
6. Open Advanced OAuth settings.
7. Keep the registration method as user-defined OAuth client.
8. Set OAuth client ID to `project-brain-client`.
9. Leave OAuth client secret empty and token endpoint auth method as `none`.
10. Save, authorize, and approve access on the Project Brain approval page.

OpenAI's UI wording changes over time. The important invariant is that ChatGPT web needs a remote HTTPS MCP endpoint; it cannot connect directly to a local-only MCP server.

Project Brain does not currently implement Dynamic Client Registration (DCR) or Client Identifier Metadata Document (CIMD), so ChatGPT cannot auto-register an OAuth client. Use the static client ID configured in `configs/project-brain.yml`.

When the tunnel URL or `auth.jwt_secret` changes, reconnect the Project Brain app in ChatGPT. Existing access tokens will fail because the issuer URL or signing key no longer matches.

## 5. Test in ChatGPT

Before testing from ChatGPT, run doctor against the same public URL:

```powershell
.\server.exe doctor --config configs\project-brain.yml --public-url https://<temporary-name>.trycloudflare.com
```

If doctor reports public endpoint failure, fix the tunnel or local server first. A visible tool catalog does not prove that later tool calls can still reach the upstream service.

Use prompts like:

```text
Use Project Brain. List my project roots.
```

```text
Use Project Brain. Inspect project "personal-projects:my-app", then create a quick phased implementation plan for this scoped bug fix with create_quick_plan.
```

```text
Use Project Brain. Start a full planning workflow for this product idea. Produce only the current phase and wait for my approval before continuing.
```

For this local setup, Project Brain is intentionally limited to `~/Projects`. Use the `personal-projects` root and the project folder name under `C:\Users\muham\Projects`:

```text
Use Project Brain. Inspect project "personal-projects:<project-folder>".
```

Avoid adding extra roots unless you explicitly want ChatGPT to see more of the machine.

## 6. Run an Implementation Agent From the Generated Prompt

From the target project root:

```powershell
<agent-command> < ".chatgpt/implementation-prompts/<prompt-file>.md"
```

## Troubleshooting

### Connection refused

- Ensure `server.exe` is running.
- Ensure `cloudflared` is running.
- Check the printed tunnel URL and `/healthz`.
- Run `.\server.exe doctor --config configs\project-brain.yml --public-url https://<your-public-host>`.

### 502 / 1033 from Cloudflare

- 502 usually means Cloudflare reached the tunnel edge, but the local service behind it did not return a valid response.
- 1033 usually means the public hostname has no active tunnel connection.
- Restart `cloudflared`, confirm it forwards to `http://127.0.0.1:3939`, then reconnect or refresh the ChatGPT app if the URL changed.

### Tools not discovered

- Confirm the URL ends with `/mcp/`.
- Refresh the app/tools in ChatGPT settings after server tool changes.
- Recreate the draft app if your ChatGPT workspace freezes tool snapshots.

### Unauthorized or 401

- Confirm `--issuer-url` matches the tunnel URL.
- Confirm Advanced OAuth settings use OAuth client ID `project-brain-client`.
- Regenerate and save `owner_secret` and `jwt_secret`.
- Restart the server after changing OAuth config.
- Reconnect the app from ChatGPT settings after changing tunnel URL or JWT secret.

### Quick Tunnel URL not detected

- Check `%TEMP%\cloudflared.log`.
- Ensure `cloudflared.exe` is either in PATH or available at `%USERPROFILE%\bin\cloudflared.exe`.
- If `~/.cloudflared/config.yaml` exists, temporarily move it or use a named tunnel instead.

## Important Boundaries

- Project Brain reads files and writes markdown planning artifacts only.
- Project Brain may also write a project-root `AGENTS.md` bootstrap file.
- Project Brain may append timestamped planning-assistant messages to project-root `fromgpt.md`.
- Downstream implementation agents may write project-root `togpt.md`; Project Brain can read it as the agent response channel.
- Project Brain does not execute shell commands and does not run implementation agents.
- Implementation happens in the downstream agent's own runtime and approval model.
