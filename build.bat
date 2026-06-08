@echo off
REM OpenTether Build Script for Windows
REM Usage: build.bat [platform]
REM   platform: linux, windows, darwin, all (default: all)
REM   Example: build.bat all

setlocal enabledelayedexpansion

REM Configuration
set OUTPUT_DIR=output
set BINARY_NAME=opentether
set STATIC_SRC=admin-ui\build
set VERSION=1.0.0
set BUILD_TIME=%date:~0,4%-%date:~5,2%-%date:~8,2%_%time:~0,2%:%time:~3,2%:%time:~6,2%
set BUILD_TIME=%BUILD_TIME: =0%
set LDFLAGS=-ldflags "-X main.version=%VERSION% -X main.buildTime=%BUILD_TIME%"

set INFO=[INFO]
set ERROR=[ERROR]

echo.
echo %INFO% OpenTether Build Script v%VERSION%
echo %INFO% =================================
echo.

REM ========================================
REM Step 1: Verify admin-ui exists
REM ========================================
echo %INFO% Step 1: Checking admin-ui/build...

if not exist "%STATIC_SRC%" (
    echo %ERROR% %STATIC_SRC% not found! Cannot build without admin UI.
    exit /b 1
)
echo %INFO%   admin-ui/build found
echo.

REM ========================================
REM Step 2: Get build target
REM ========================================
set TARGET=%~1
if "%TARGET%"=="" set TARGET=all

REM ========================================
REM Step 3: Build
REM ========================================
call :build_%TARGET% 2>nul
if errorlevel 1 (
    echo %ERROR% Unknown target: %TARGET%
    echo %ERROR% Valid targets: all, linux, windows, darwin, current, help
    exit /b 1
)
goto :done

:build_all
    echo %INFO% Step 2: Building all platforms...
    call :build_one windows amd64 .exe
    call :build_one linux amd64
    call :build_one darwin amd64
    call :copy_static
    echo.
    echo %INFO% All builds completed!
    exit /b 0

:build_windows
    echo %INFO% Step 2: Building Windows amd64...
    call :build_one windows amd64 .exe
    call :copy_static windows
    exit /b 0

:build_linux
    echo %INFO% Step 2: Building Linux amd64...
    call :build_one linux amd64
    call :copy_static linux
    exit /b 0

:build_darwin
    echo %INFO% Step 2: Building Darwin (macOS) amd64...
    call :build_one darwin amd64
    call :copy_static darwin
    exit /b 0

:build_current
    echo %INFO% Step 2: Building for current platform...
    for /f "delims=" %%i in ('go env GOOS') do set T_GOOS=%%i
    for /f "delims=" %%i in ('go env GOARCH') do set T_GOARCH=%%i
    call :build_one !T_GOOS! !T_GOARCH! .exe 2>nul
    call :copy_static
    exit /b 0

:build_help
    goto :show_help

REM ========================================
REM Internal: Build single platform
REM ========================================
:build_one
    set TARGET_PLATFORM=%1
    set TARGET_ARCH=%2
    set TARGET_EXT=%3

    if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

    set OUT_FILE=%OUTPUT_DIR%\%BINARY_NAME%-%TARGET_PLATFORM%-%TARGET_ARCH%%TARGET_EXT%
    echo %INFO%   Building: %OUT_FILE%

    set GOOS=%TARGET_PLATFORM%
    set GOARCH=%TARGET_ARCH%
    set CGO_ENABLED=0
    go build %LDFLAGS% -o "%OUT_FILE%" .
    if errorlevel 1 (
        echo %ERROR% Build failed for %TARGET_PLATFORM%/%TARGET_ARCH%
        exit /b 1
    )
    echo %INFO%     Done: %OUT_FILE%
    exit /b 0

REM ========================================
REM Step 4: Copy static files
REM ========================================
:copy_static
    echo.
    echo %INFO% Step 3: Copying admin-ui/build to output directory...

    set STATIC_DEST=%OUTPUT_DIR%\admin-ui\build

    REM Remove old static files
    if exist "%STATIC_DEST%" rd /s /q "%STATIC_DEST%"

    REM Copy admin-ui/build recursively
    xcopy "%STATIC_SRC%" "%STATIC_DEST%" /E /I /Q /Y
    if errorlevel 1 (
        echo %ERROR% Failed to copy static files!
        exit /b 1
    )
    echo %INFO%   Static files copied to %STATIC_DEST%
    exit /b 0

REM ========================================
REM Done - Show output
REM ========================================
:done
echo.
echo %INFO% Build outputs:
dir /b "%OUTPUT_DIR%\%BINARY_NAME%*" 2>nul
echo.
echo %INFO% Directory structure:
echo %INFO%   %OUTPUT_DIR%\
echo %INFO%     +-- %BINARY_NAME%-windows-amd64.exe
echo %INFO%     +-- %BINARY_NAME%-linux-amd64
echo %INFO%     +-- %BINARY_NAME%-darwin-amd64
echo %INFO%     +-- admin-ui\build\     ^(static web files^)
echo.
echo %INFO% Run: cd %OUTPUT_DIR% ^&^& %BINARY_NAME%-windows-amd64.exe
exit /b 0

:show_help
echo.
echo OpenTether Build Script
echo =======================
echo.
echo Usage: build.bat [target]
echo.
echo Targets:
echo   all       - Build for all platforms + copy static files (default)
echo   linux     - Build for Linux amd64 + copy static files
echo   windows   - Build for Windows amd64 + copy static files
echo   darwin    - Build for macOS amd64 + copy static files
echo   current   - Build for current platform + copy static files
echo   help      - Show this help
echo.
echo Examples:
echo   build.bat              - Build all platforms
echo   build.bat windows      - Build Windows only
echo.
echo Note: Static files (admin-ui/build) are automatically copied to
echo       the output directory after build. Run the executable from
echo       the output directory to serve the admin UI.
exit /b 0
