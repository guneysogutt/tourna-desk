@echo off
setlocal

echo ========================================================
echo Building Tourna-Desk Frontend (React + Vite + Tailwind)
echo ========================================================
cd web
call npm install
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%
call npx vite build
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%
cd ..

if not exist "cmd\server\dist" mkdir "cmd\server\dist"
xcopy /E /I /Y "web\dist\*" "cmd\server\dist\"

echo ========================================================
echo Building Standalone Single-Binary Executable (CGO=0)
echo ========================================================
set CGO_ENABLED=0
go build -ldflags="-s -w" -o tourna-desk.exe ./cmd/server
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%

echo ========================================================
echo Build Complete: tourna-desk.exe
echo ========================================================
endlocal
