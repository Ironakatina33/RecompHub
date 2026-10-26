@echo off
chcp 65001 >nul
title Diagnostic RecompHub v3
set "DATA=%LOCALAPPDATA%\RecompHub"
echo RecompHub v3 - diagnostic
where msedge.exe
if exist "%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe" echo Edge x86 PRESENT
if exist "%ProgramFiles%\Microsoft\Edge\Application\msedge.exe" echo Edge x64 PRESENT
if exist "%DATA%\v3.log" (echo ===== LOG ===== & type "%DATA%\v3.log") else (echo Aucun log v3 trouve.)
pause
