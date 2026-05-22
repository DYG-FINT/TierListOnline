@echo off
setlocal

set BUILD_DIR=.\build
set APP_NAME=TLO

for /f "usebackq delims=" %%v in ("version.txt") do set VERSION=%%v
set LDFLAGS=-s -w -X main.Version=%VERSION%

if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
mkdir "%BUILD_DIR%"

echo ==> Building version %VERSION% for Linux (amd64)...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o "%BUILD_DIR%\%APP_NAME%-%VERSION%_linux_amd64" .

echo ==> Building version %VERSION% for macOS (amd64)...
set CGO_ENABLED=0
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o "%BUILD_DIR%\%APP_NAME%-%VERSION%_darwin_amd64" .

echo ==> Building version %VERSION% for Windows (amd64)...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o "%BUILD_DIR%\%APP_NAME%-%VERSION%_windows_amd64.exe" .

echo.
echo Done. Binaries in %BUILD_DIR%:
dir "%BUILD_DIR%"
endlocal
