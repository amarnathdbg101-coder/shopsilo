@echo off
title ShopMe Go Backend Server
cd /d "%~dp0"
cls
echo ===================================================
echo       ShopMe / Shopsilo Go Backend API Server
echo ===================================================
echo.
echo Building latest backend binary...
go build -o bin\api.exe .\cmd\api
if %errorlevel% neq 0 (
    echo [ERROR] Backend compilation failed! Please fix errors above.
    pause
    exit /b %errorlevel%
)

echo [OK] Build successful!
echo.
echo ===================================================
echo Server starting on:
echo   - Localhost : http://localhost:8080
echo   - Network IP: http://192.168.1.16:8080
echo   - Health API: http://localhost:8080/health
echo ===================================================
echo Press Ctrl+C anytime to stop the server.
echo.

bin\api.exe
pause
