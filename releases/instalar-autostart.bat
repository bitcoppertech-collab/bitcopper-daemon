@echo off
echo Instalando BITCOPPER DAEMON en inicio de Windows...
set STARTUP=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup
copy bitcopper-daemon-windows.exe "%STARTUP%\bitcopper-daemon-windows.exe"
copy start-bitcopper.bat "%STARTUP%\start-bitcopper.bat"
echo.
echo [OK] Daemon instalado. Se ejecutara automaticamente al iniciar Windows.
echo [OK] Para desinstalar, elimina los archivos de:
echo      %STARTUP%
pause
