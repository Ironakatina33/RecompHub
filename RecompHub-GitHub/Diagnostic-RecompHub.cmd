@echo off
setlocal
set "EXE=%~dp0RecompHub.exe"
set "APPDATA_DIR=%LOCALAPPDATA%\RecompHub"
set "REPORT=%TEMP%\RecompHub-Diagnostic.txt"
(
  echo RECOMPHUB - DIAGNOSTIC LOCAL
  echo Date: %date% %time%
  ver
  echo.
  echo Executable: %EXE%
  if exist "%EXE%" (echo Fichier RecompHub.exe present) else (echo ERREUR: RecompHub.exe absent)
  echo.
) > "%REPORT%"
if exist "%EXE%" start "" "%EXE%"
timeout /t 4 /nobreak >nul
(
  echo === Processus RecompHub ===
  tasklist /FI "IMAGENAME eq RecompHub.exe"
  echo.
  echo === Journal Windows natif ===
  if exist "%APPDATA_DIR%\startup.log" (type "%APPDATA_DIR%\startup.log") else (echo Aucun journal de demarrage natif)
  echo.
  echo === Erreur Windows native ===
  if exist "%APPDATA_DIR%\startup-error.log" (type "%APPDATA_DIR%\startup-error.log") else (echo Aucun journal d'erreur natif)
  echo.
  echo === Ancien journal RecompHub v1.2 ===
  if exist "%APPDATA_DIR%\error.log" (type "%APPDATA_DIR%\error.log") else (echo Pas d'ancien error.log)
  if exist "%APPDATA_DIR%\launcher.log" (type "%APPDATA_DIR%\launcher.log") else (echo Pas d'ancien launcher.log)
) >> "%REPORT%" 2>&1
start "" notepad.exe "%REPORT%"
echo Rapport cree: "%REPORT%"
echo Ce diagnostic ne transmet rien sur Internet.
pause
