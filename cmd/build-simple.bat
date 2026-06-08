@echo on
REM Simple OpenTether Build Script for Windows
REM This builds the cmd version with filesystem-based admin UI

cd /d %~dp0..
echo Building OpenTether from project root: %CD%

REM Create output directory
if not exist "output" mkdir output

REM Build the cmd version
echo Building cmd/opentether.exe...
go build -o output/opentether.exe ./cmd

if exist "output/opentether.exe" (
    echo Build successful!
    echo Output: output/opentether.exe
    dir output/opentether.exe
) else (
    echo Build failed!
    pause
    exit /b 1
)

pause
