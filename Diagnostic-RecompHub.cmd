@echo off
setlocal
chcp 65001 >nul
echo RecompHub v4 - diagnostic
if not exist "%~dp0RecompHub.exe" (
 echo RecompHub.exe introuvable dans le dossier.
 pause
 exit /b 1
)
start "" "%~dp0RecompHub.exe"
timeout /t 4 /nobreak >nul
if exist "%LOCALAPPDATA%\RecompHub\v4.log" (
 start "" notepad "%LOCALAPPDATA%\RecompHub\v4.log"
) else (
 echo Aucun journal trouve dans %%LOCALAPPDATA%%\RecompHub.
 pause
)
