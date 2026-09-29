@echo off
setlocal
cd /d "%~dp0"
title chaOS - source launcher

where node >nul 2>nul
if errorlevel 1 (
  echo [chaOS] Node.js 22 or newer is required.
  echo [chaOS] Install Node.js 22 LTS, then double-click this file again.
  pause
  exit /b 1
)

node -e "const m=Number(process.versions.node.split('.')[0]); if(m<22){console.error('[chaOS] Node.js 22 or newer is required. Found '+process.versions.node); process.exit(1)}"
if errorlevel 1 (
  pause
  exit /b 1
)

where npm >nul 2>nul
if errorlevel 1 (
  echo [chaOS] npm was not found. Reinstall Node.js with npm included.
  pause
  exit /b 1
)

echo [chaOS] Installing the exact dependencies from package-lock.json...
call npm ci
if errorlevel 1 goto :failed

echo.
echo [chaOS] Starting the desktop app in development mode...
echo [chaOS] Keep this window open while chaOS is running.
call npm run dev
if errorlevel 1 goto :failed
exit /b 0

:failed
echo.
echo [chaOS] Startup failed. The error is shown above.
pause
exit /b 1
