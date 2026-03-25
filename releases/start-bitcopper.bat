@echo off
title BITCOPPER DAEMON - Proof of Heat
color 0A
echo.
echo  ██████╗ ██╗████████╗ ██████╗ ██████╗ ██████╗ ██████╗ ███████╗██████╗ 
echo  ██╔══██╗██║╚══██╔══╝██╔════╝██╔═══██╗██╔══██╗██╔══██╗██╔════╝██╔══██╗
echo  ██████╔╝██║   ██║   ██║     ██║   ██║██████╔╝██████╔╝█████╗  ██████╔╝
echo  ██╔══██╗██║   ██║   ██║     ██║   ██║██╔═══╝ ██╔═══╝ ██╔══╝  ██╔══██╗
echo  ██████╔╝██║   ██║   ╚██████╗╚██████╔╝██║     ██║     ███████╗██║  ██║
echo  ╚═════╝ ╚═╝   ╚═╝    ╚═════╝ ╚═════╝ ╚═╝     ╚═╝     ╚══════╝╚═╝  ╚═╝
echo.
echo  Proof of Heat Protocol v2.0.0 — In cuprum veritas
echo  Bitcopper Tech SpA — Calama, Chile
echo.

:: Verificar si OpenHardwareMonitor está corriendo (mejor temperatura)
tasklist /FI "IMAGENAME eq OpenHardwareMonitor.exe" 2>NUL | find /I /N "OpenHardwareMonitor.exe">NUL
if "%ERRORLEVEL%"=="0" (
    echo  [OK] OpenHardwareMonitor detectado - temperatura real disponible
) else (
    echo  [INFO] OpenHardwareMonitor no detectado - usando estimacion de temperatura
    echo  [INFO] Para temperatura real: https://openhardwaremonitor.org
)

echo.
echo  Iniciando daemon...
echo  Presiona Ctrl+C para detener
echo.

bitcopper-daemon-windows.exe

pause
