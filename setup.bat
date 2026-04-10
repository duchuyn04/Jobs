@echo off
setlocal EnableDelayedExpansion
echo.
echo  ========================================
echo   Job Aggregator - Setup
echo  ========================================
echo.

::Kiểm tra Go
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [...] Go chua duoc cai. Dang cai tu dong...
    winget install -e --id GoLang.Go --silent --accept-source-agreements --accept-package-agreements
    if %ERRORLEVEL% neq 0 (
        echo [LOI] Khong the cai Go tu dong.
        echo   Vui long cai thu cong tai: https://go.dev/dl/
        pause
        exit /b 1
    )
    :: Refresh PATH trong session hiện tại
    for /f "tokens=2*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH 2^>nul') do set "SYSPATH=%%B"
    for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USERPATH=%%B"
    set "PATH=!SYSPATH!;!USERPATH!"
    echo [OK] Go da duoc cai dat
) else (
    echo [OK] Go da co san
)

::Kiểm tra Node.js
where node >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [...] Node.js chua duoc cai. Dang cai tu dong...
    winget install -e --id OpenJS.NodeJS.LTS --silent --accept-source-agreements --accept-package-agreements
    if %ERRORLEVEL% neq 0 (
        echo [LOI] Khong the cai Node.js tu dong.
        echo   Vui long cai thu cong tai: https://nodejs.org/
        pause
        exit /b 1
    )
    :: Refresh PATH trong session hiện tại
    for /f "tokens=2*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH 2^>nul') do set "SYSPATH=%%B"
    for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USERPATH=%%B"
    set "PATH=!SYSPATH!;!USERPATH!"
    echo [OK] Node.js da duoc cai dat
) else (
    echo [OK] Node.js da co san
)

::Cài npm dependencies nếu chưa có
if not exist "%~dp0frontend\node_modules" (
    echo.
    echo [...] Dang cai frontend dependencies (lan dau chay)...
    cd /d "%~dp0frontend"
    call npm install
    if %ERRORLEVEL% neq 0 (
        echo [LOI] npm install that bai!
        pause
        exit /b 1
    )
    echo [OK] Frontend dependencies da cai xong
) else (
    echo [OK] Frontend dependencies da co san
)

::Tạo Desktop shortcut trỏ về start.bat trong project
echo.
echo [...] Dang tao shortcut tren Desktop...
powershell -NoProfile -Command ^
  "$shell = New-Object -ComObject WScript.Shell;" ^
  "$sc = $shell.CreateShortcut([Environment]::GetFolderPath('Desktop') + '\Job Aggregator.lnk');" ^
  "$sc.TargetPath = '%~dp0start.bat';" ^
  "$sc.WorkingDirectory = '%~dp0';" ^
  "$sc.Description = 'Khoi dong Job Aggregator';" ^
  "$sc.Save();"

echo [OK] Shortcut 'Job Aggregator' da duoc tao tren Desktop
echo.
echo  ========================================
echo   Setup hoan tat!
echo   Su dung shortcut 'Job Aggregator'
echo   tren Desktop de khoi dong app.
echo  ========================================
echo.
pause
