@echo off
setlocal enabledelayedexpansion

set "CLOUDFLARED=cloudflared.exe"
where cloudflared.exe > nul 2>&1
if errorlevel 1 (
    if exist "%USERPROFILE%\bin\cloudflared.exe" (
        set "CLOUDFLARED=%USERPROFILE%\bin\cloudflared.exe"
    )
)

if "!CLOUDFLARED!"=="cloudflared.exe" (
    where cloudflared.exe > nul 2>&1
    if errorlevel 1 (
        echo cloudflared.exe not found in PATH or %USERPROFILE%\bin.
        echo Install cloudflared or set it in PATH before running this launcher.
        exit /b 1
    )
)

if not exist "server.exe" (
    echo server.exe not found. Build it first:
    echo go build -o server.exe ./cmd/server
    exit /b 1
)

echo Starting Cloudflare Quick Tunnel...
echo Using: !CLOUDFLARED!
start /B "" "!CLOUDFLARED!" tunnel --url http://127.0.0.1:3939 > "%TEMP%\cloudflared.log" 2>&1

echo Waiting for tunnel URL...
set "TUNNEL_URL="
for /l %%i in (1,1,30) do (
    for /f "tokens=*" %%a in ('findstr /R "https://.*\.trycloudflare\.com" "%TEMP%\cloudflared.log" 2^>nul') do (
        if "!TUNNEL_URL!"=="" (
            for /f "tokens=2 delims=|" %%b in ("%%a") do (
                set "raw=%%b"
                set "TUNNEL_URL=!raw: =!"
            )
        )
    )
    if not "!TUNNEL_URL!"=="" goto tunnel_found
    timeout /t 1 /nobreak > nul
)

:tunnel_found
if "!TUNNEL_URL!"=="" (
    echo Could not detect tunnel URL. Check %TEMP%\cloudflared.log manually.
    type "%TEMP%\cloudflared.log"
    exit /b 1
)

echo.
echo ========================================
echo  Tunnel URL: !TUNNEL_URL!
echo  MCP Endpoint: !TUNNEL_URL!/mcp/
echo ========================================
echo.
echo Paste this URL into ChatGPT connector:
echo !TUNNEL_URL!/mcp/
echo.

if exist "configs\project-brain.yml" (
    set "CONFIG=configs\project-brain.yml"
) else (
    set "CONFIG=configs\project-brain.example.yml"
)
server.exe --config !CONFIG! --issuer-url !TUNNEL_URL!
