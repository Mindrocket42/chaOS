@echo off
setlocal
cd /d "%~dp0"
echo.
echo chaOS second-refresh browser-native verification
echo ================================================
echo ChatGPT decides. chaOS executes. Receipts prove what happened.
echo This does NOT start Electron, MCP, or an OpenAI tunnel.
echo.
where go >nul 2>nul
if errorlevel 1 (
  echo ERROR: Go is not installed or not on PATH.
  echo chaOS native-host requires Go 1.23 or newer to build from source.
  exit /b 2
)
go version
echo.
echo [1/6] Checking extension JavaScript when Node is available...
where node >nul 2>nul
if errorlevel 1 (
  echo WARNING: Node is not on PATH, so extension JavaScript syntax checks are skipped.
  echo The browser runtime does not require npm or Electron. Node 22+ is recommended for full source verification.
) else (
  node --version
  node --check "%~dp0extension-chaos\background.js"
  if errorlevel 1 goto :fail
  node --check "%~dp0extension-chaos\content.js"
  if errorlevel 1 goto :fail
  node --check "%~dp0extension-chaos\popup.js"
  if errorlevel 1 goto :fail
)
echo.
echo [2/6] Testing standalone native host...
pushd "%~dp0native-host"
go test ./...
if errorlevel 1 goto :buildfail
echo.
echo [3/6] Vetting standalone native host...
go vet ./...
if errorlevel 1 goto :buildfail
echo.
echo [4/6] Building chaos-host.exe...
go build -trimpath -ldflags="-s -w" -o "%~dp0chaos-host.exe" .
if errorlevel 1 goto :buildfail
popd
echo.
echo [5/6] Registering Chrome Native Messaging host for this Windows user...
"%~dp0chaos-host.exe" --register
if errorlevel 1 goto :fail
echo.
echo [6/6] Showing closed-by-default host configuration...
"%~dp0chaos-host.exe" --status
if errorlevel 1 goto :fail
echo.
echo SOURCE BUILD AND REGISTRATION PASSED.
echo.
echo NEXT - live Chrome evidence:
echo   1. Open chrome://extensions
echo   2. Turn on Developer mode
echo   3. Click Load unpacked
echo   4. Select: %~dp0extension-chaos
echo   5. Open ChatGPT in Chrome and open chaOS Local Bridge
echo   6. Confirm health returns capability_fingerprint, host_instance and receipt_schema
echo   7. Approve a disposable test root and enable only the capabilities needed
echo   8. Verify fs.write returns receipt evidence with the resulting SHA-256
echo   9. Verify proc.run returns a process_token, reload ChatGPT, then recover that process using the token
echo.
echo Expected fresh state: native host connected; approved roots NONE; capabilities OFF.
echo process_id alone is not authority across browser documents; preserve process_token for reload recovery.
echo No npm install or Electron process is required for this browser-native path.
exit /b 0

:buildfail
popd
:fail
echo.
echo FAILED. The line above is the relevant failure; do not start the inherited Electron app as a workaround.
exit /b 1
