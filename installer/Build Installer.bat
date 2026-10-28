@echo off
setlocal EnableExtensions
title Checkers - Build Installer
cd /d "%~dp0"

echo.
echo   Builds Checkers-Setup-v^<version^>.exe into the Output folder.
echo.

call :findiscc
if defined ISCC goto :build
echo   Inno Setup 6 was not found on this PC. It is free from https://jrsoftware.org/isdl.php
where winget >nul 2>nul
if errorlevel 1 goto :noinno
choice /C YN /M "  Install Inno Setup now with winget"
if errorlevel 2 goto :noinno
winget install --id JRSoftware.InnoSetup -e --silent --accept-package-agreements --accept-source-agreements
call :findiscc
if not defined ISCC goto :noinno

:build
echo   Using: %ISCC%
if not exist "Output" mkdir "Output"
"%ISCC%" /Qp "installer\CheckersSetup.iss"
if errorlevel 1 (
  echo   The installer failed to build - see the messages above.
  pause
  exit /b 1
)
echo   Done:
dir /b "Output\*.exe"
start "" explorer "%CD%\Output"
pause
exit /b 0

:findiscc
set "ISCC="
for %%D in ("%ProgramFiles(x86)%\Inno Setup 6" "%ProgramFiles%\Inno Setup 6" "%LOCALAPPDATA%\Programs\Inno Setup 6" "%ProgramFiles(x86)%\Inno Setup 7" "%ProgramFiles%\Inno Setup 7" "%LOCALAPPDATA%\Programs\Inno Setup 7") do (
  if exist "%%~D\ISCC.exe" set "ISCC=%%~D\ISCC.exe"
)
if defined ISCC exit /b 0
for /f "delims=" %%I in ('where ISCC.exe 2^>nul') do if not defined ISCC set "ISCC=%%I"
exit /b 0

:noinno
echo   Please install Inno Setup 6.3 or newer (free), then run this file again.
start "" "https://jrsoftware.org/isdl.php"
pause
exit /b 1
