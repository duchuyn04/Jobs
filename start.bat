@echo off
echo  Starting JobAggregator...
echo.

echo [1/2] Starting Go backend (port 8080)...
start "JobAggregator - Backend" cmd /k "cd /d "%~dp0backend-go" && go run ."

timeout /t 2 /nobreak >nul

echo [2/2] Starting Next.js frontend (port 3000)...
start "JobAggregator - Frontend" cmd /k "cd /d "%~dp0frontend" && npm run dev"

echo.
echo  Both servers are starting:
echo    Backend:  http://localhost:8080
echo    Frontend: http://localhost:3000
echo.
echo  Opening browser in 5 seconds...
timeout /t 5 /nobreak >nul
start http://localhost:3000
