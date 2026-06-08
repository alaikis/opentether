@echo off
REM ============================================
REM  OpenTether Build Script
REM  Step 1: Build Frontend (npm run build)
REM  Step 2: Restore Setup Page
REM  Step 3: Build Go Binary (embed frontend)
REM  Usage: build.bat [all|windows|linux|darwin|dev]
REM ============================================

setlocal enabledelayedexpansion
set OUTPUT_DIR=output
set BIN=opentether
set VER=1.0.0

echo.
echo ==========================================
echo  OpenTether Build v%VER%
echo ==========================================
echo.

REM ============================================
REM Step 1: Build Frontend
REM ============================================
echo [1/4] Building frontend...

if exist "admin-ui\package.json" (
    cd admin-ui
    if not exist "node_modules" ( call npm install --silent )
    call npm run build
    if errorlevel 1 ( echo [ERROR] Frontend build failed & cd .. & exit /b 1 )
    cd ..
    echo   Frontend built: admin-ui/build/
) else (
    echo   [SKIP] admin-ui/package.json not found
)

if not exist "admin-ui\build\index.html" (
    echo [ERROR] admin-ui/build/index.html not found
    exit /b 1
)

REM ============================================
REM Step 2: Restore Setup Page
REM ============================================
echo [2/4] Restoring setup page...
if exist "admin-ui\static\setup\index.html" (
    mkdir "admin-ui\build\setup" 2>nul
    copy /y "admin-ui\static\setup\index.html" "admin-ui\build\setup\index.html" >nul
    echo   Setup page restored
) else (
    echo   [WARN] Setup page not found, using existing
)

REM ============================================
REM Step 3: Get Target
REM ============================================
set TARGET=%~1
if "%TARGET%"=="" set TARGET=all

echo [3/4] Building Go binary...

if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

if "%TARGET%"=="all" goto :all
if "%TARGET%"=="windows" goto :win
if "%TARGET%"=="linux" goto :linux
if "%TARGET%"=="darwin" goto :darwin
if "%TARGET%"=="dev" goto :dev
echo [ERROR] Unknown target: %TARGET%
echo Valid: all, windows, linux, darwin, dev
exit /b 1

:all
    call :build windows amd64 .exe
    call :build linux amd64
    call :build darwin amd64
    goto :done

:win
    call :build windows amd64 .exe
    goto :done

:linux
    call :build linux amd64
    goto :done

:darwin
    call :build darwin amd64
    goto :done

:dev
    echo   Building dev mode (filesystem)...
    go build -ldflags "-X main.version=%VER%" -o "%OUTPUT_DIR%\%BIN%-dev.exe" .\cmd
    echo [4/4] Dev build done
    echo   Run: %OUTPUT_DIR%\%BIN%-dev.exe
    exit /b 0

REM ============================================
REM Build function
REM ============================================
:build
    set OS=%1
    set ARCH=%2
    set EXT=%3
    set OUT=%OUTPUT_DIR%\%BIN%-%OS%-%ARCH%%EXT%
    echo   Building: %OUT%
    set GOOS=%OS%
    set GOARCH=%ARCH%
    set CGO_ENABLED=0
    go build -ldflags "-X main.version=%VER%" -o "%OUT%" .
    if errorlevel 1 ( echo [ERROR] Build failed & exit /b 1 )
    exit /b 0

:done
echo [4/4] Build complete
echo.
echo ==========================================
echo  Output
echo ==========================================
dir /b "%OUTPUT_DIR%\%BIN%*" 2>nul
echo.
echo  cd %OUTPUT_DIR% ^&^& %BIN%-windows-amd64.exe
