@echo off
setlocal EnableDelayedExpansion

:: ─── Yêu cầu quyền Admin nếu chưa có ─────────────────────────
net session >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Yeu cau quyen Admin de cai dat...
    powershell -Command "Start-Process -FilePath '%~f0' -Verb RunAs"
    exit /b
)

echo.
echo  ========================================
echo   Job Aggregator - Setup
echo  ========================================
echo.

:: ─── Cài Go nếu chưa có ───────────────────────────────────────
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [...] Go chua duoc cai. Dang tai va cai tu dong...
    powershell -NoProfile -Command ^
        "$url = (Invoke-WebRequest 'https://go.dev/dl/?mode=json' | ConvertFrom-Json)[0].files | Where-Object { $_.os -eq 'windows' -and $_.arch -eq 'amd64' -and $_.kind -eq 'installer' } | Select-Object -ExpandProperty filename;" ^
        "$dlUrl = 'https://go.dev/dl/' + $url;" ^
        "$out = \"$env:TEMP\go_installer.msi\";" ^
        "Write-Host \"  Dang tai: $dlUrl\";" ^
        "Invoke-WebRequest -Uri $dlUrl -OutFile $out -UseBasicParsing;" ^
        "Write-Host '  Dang cai Go...';" ^
        "Start-Process msiexec.exe -ArgumentList \"/i `\"$out`\" /quiet /norestart\" -Wait;" ^
        "Write-Host '[OK] Go da duoc cai xong';"
    :: Refresh PATH trong session hiện tại
    for /f "skip=2 tokens=3*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH') do set "SYSPATH=%%A %%B"
    for /f "skip=2 tokens=3*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USERPATH=%%A %%B"
    set "PATH=!SYSPATH!;!USERPATH!"
    where go >nul 2>&1
    if %ERRORLEVEL% neq 0 (
        echo [LOI] Cai Go that bai. Vui long cai thu cong tai https://go.dev/dl/ roi chay lai.
        pause & exit /b 1
    )
) else (
    echo [OK] Go da co san
)

:: ─── Cài Node.js nếu chưa có ──────────────────────────────────
where node >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [...] Node.js chua duoc cai. Dang tai va cai tu dong...
    powershell -NoProfile -Command ^
        "$ver = ((Invoke-WebRequest 'https://nodejs.org/dist/index.json' -UseBasicParsing | ConvertFrom-Json) | Where-Object { $_.lts -ne $false })[0].version;" ^
        "$dlUrl = \"https://nodejs.org/dist/$ver/node-$ver-x64.msi\";" ^
        "$out = \"$env:TEMP\node_installer.msi\";" ^
        "Write-Host \"  Dang tai: $dlUrl\";" ^
        "Invoke-WebRequest -Uri $dlUrl -OutFile $out -UseBasicParsing;" ^
        "Write-Host '  Dang cai Node.js...';" ^
        "Start-Process msiexec.exe -ArgumentList \"/i `\"$out`\" /quiet /norestart\" -Wait;" ^
        "Write-Host '[OK] Node.js da duoc cai xong';"
    :: Refresh PATH trong session hiện tại
    for /f "skip=2 tokens=3*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH') do set "SYSPATH=%%A %%B"
    for /f "skip=2 tokens=3*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USERPATH=%%A %%B"
    set "PATH=!SYSPATH!;!USERPATH!"
    where node >nul 2>&1
    if %ERRORLEVEL% neq 0 (
        echo [LOI] Cai Node.js that bai. Vui long cai thu cong tai https://nodejs.org/ roi chay lai.
        pause & exit /b 1
    )
) else (
    echo [OK] Node.js da co san
)

::Cài npm dependencies nếu chưa có
if not exist "%~dp0frontend\node_modules" (
    echo.
    echo [...] Dang cai frontend dependencies...
    cd /d "%~dp0frontend"
    call npm install
    if %ERRORLEVEL% neq 0 (
        echo [LOI] npm install that bai!
        pause & exit /b 1
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
