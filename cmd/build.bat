@echo on
REM OpenTether Build Script for Windows
REM Builds both frontend and backend, embeds frontend into binary
REM Usage: build.bat [options]

set "RED="
set "GREEN="
set "YELLOW="
set "NC="

echo ========================================
echo   OpenTether Build Script (Windows)
echo ========================================
echo.

:: Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."
cd /d "%PROJECT_ROOT%"

:: Default values
set "BUILD_MODE=full"
set "SKIP_FRONTEND="
set "SKIP_BACKEND="
set "OUTPUT_NAME=opentether"

:: Parse arguments
:parse_args
if "%~1"=="" goto args_done
if /i "%~1"=="--skip-frontend" set "SKIP_FRONTEND=1" & shift & goto parse_args
if /i "%~1"=="--skip-backend" set "SKIP_BACKEND=1" & shift & goto parse_args
if /i "%~1"=="--output" set "OUTPUT_NAME=%~2" & shift & shift & goto parse_args
if /i "%~1"=="--dev" set "BUILD_MODE=dev" & shift & goto parse_args
if /i "%~1"=="-h" goto show_help
if /i "%~1"=="--help" goto show_help
echo Unknown option: %~1
exit /b 1

:show_help
echo Usage: %~nx0 [options]
echo.
echo Options:
echo   --skip-frontend    Skip frontend build
echo   --skip-backend     Skip backend build
echo   --output NAME      Output binary name ^(default: wisehoof^)
echo   --dev              Development mode ^(keep local files^)
echo   -h, --help         Show this help message
exit /b 0

:args_done

echo.
echo Project root: %PROJECT_ROOT%
echo Build mode: %BUILD_MODE%
echo.

:: Step 1: Build Frontend
if not defined SKIP_FRONTEND (
    echo [1/2] Building frontend...

    if not exist "admin-ui" (
        echo Warning: admin-ui directory not found, skipping frontend build
    ) else (
        :: Check if this is a Node.js project (has package.json)
        if exist "admin-ui\package.json" (
            cd admin-ui

            :: Check if node_modules exists
            if not exist "node_modules" (
                echo Installing Node.js dependencies...
                call npm install
            )

            :: Build the frontend
            echo Running npm run build...
            call npm run build

            cd /d "%PROJECT_ROOT%"

            if exist "admin-ui\build" (
                echo Frontend built successfully!
            ) else (
                echo Error: Frontend build failed - build directory not found
                exit /b 1
            )
        ) else (
            echo Note: admin-ui appears to be pre-built (no package.json found)
            if exist "admin-ui\build" (
                echo Using existing build at admin-ui\build
            ) else (
                echo Warning: admin-ui\build not found
            )
        )
    )
) else (
    echo [1/2] Skipping frontend build
)

:: Step 2: Build Backend
if not defined SKIP_BACKEND (
    echo [2/2] Building backend...

    :: Create output directory
    if not exist "output" mkdir output

    if "%BUILD_MODE%"=="dev" (
        :: Development build - use main.go directly
        echo Development build ^(using main.go^)...
        go build -o "output\%OUTPUT_NAME%" .
    ) else (
        :: Production build - use cmd/main.go with embedded frontend
        echo Production build ^(embedding frontend^)...

        :: Verify frontend exists
        if not exist "admin-ui\build" (
            echo Error: admin-ui\build is not found
            echo Please run with frontend build first
            exit /b 1
        )

        :: Check if build directory is empty
        dir /b "admin-ui\build" 2>nul | findstr "^" >nul
        if errorlevel 1 (
            echo Error: admin-ui\build is empty
            exit /b 1
        )

        :: Build with embedded frontend
        go build -o "output\%OUTPUT_NAME%" .\cmd
    )

    if exist "output\%OUTPUT_NAME%" (
        echo Backend built successfully!
    ) else (
        echo Error: Backend build failed
        exit /b 1
    )
) else (
    echo [2/2] Skipping backend build
)

echo.
echo ========================================
echo   Build Complete!
echo ========================================
echo.
echo Output: %PROJECT_ROOT%\output\%OUTPUT_NAME%
echo.

:: Show file info
if exist "output\%OUTPUT_NAME%" (
    echo Binary info:
    dir /b "output\%OUTPUT_NAME%"
)

pause
