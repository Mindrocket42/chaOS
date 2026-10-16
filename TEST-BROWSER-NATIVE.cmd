@echo off
setlocal
cd /d "%~dp0"
echo.
echo chaOS browser-native test bootstrap
echo ==================================
echo This does NOT start Electron, MCP, or an OpenAI tunnel.
echo.
where go >nul 2>nul
if errorlevel 1 (
  echo ERROR: Go is not installed or not on PATH.
  echo chaOS native-host currently requires Go 1.23 or newer to build from source.
  echo Install Go, reopen this terminal, then run this file again.
  exit /b 2
)
go version
echo.
echo [1/4] Testing standalone native host...
pushd "%~dp0native-host"
go test ./...
if errorlevel 1 goto :buildfail
echo.
echo [2/4] Building chaos-host.exe...
go build -trimpath -ldflags="-s -w" -o "%~dp0chaos-host.exe" .
if errorlevel 1 goto :buildfail
popd
echo.
echo [3/4] Registering Chrome Native Messaging host for this Windows user...
"%~dp0chaos-host.exe" --register
if errorlevel 1 goto :fail
echo.
echo [4/4] Showing closed-by-default host configuration...
"%~dp0chaos-host.exe" --status
if errorlevel 1 goto :fail
echo.
echo BUILD AND REGISTRATION PASSED.
echo.
echo NEXT - manual Chrome step:
echo   1. Open chrome://extensions
echo   2. Turn on Developer mode
echo   3. Click Load unpacked
echo   4. Select: %~dp0extension-chaos
echo   5. Open chatgpt.com, click the chaOS Local Bridge extension, and test connection.
echo.
echo Expected initial state: native host connected; approved roots NONE; capabilities OFF.
echo No npm command or Electron process is required for this browser-native test.
exit /b 0

:buildfail
popd
:fail
echo.
echo FAILED. The line above is the relevant failure; do not start the Electron app as a workaround.
exit /b 1
