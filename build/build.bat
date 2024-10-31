@echo off
set SCRIPTS_PATH=%~dp0
set EBUS_PATH=%SCRIPTS_PATH%..
set INPUT_ARGS=%1
cd /d %SCRIPTS_PATH%
if "%INPUT_ARGS%" == "" GOTO INPUT
GOTO NEXT

:INPUT
echo Please input build platform is [lin(u)x] or [(w)in]:
set /p INPUT_ARGS=
if "%INPUT_ARGS%" == "" GOTO END

for %%i in (a b c d e f g h i j k l m n o p q r s t u v w x y z) do call set INPUT_ARGS=%%INPUT_ARGS:%%i=%%i%%
if "%INPUT_ARGS%" == "u" (
    set OS_TYPE=linux
    GOTO NEXT
)
if "%INPUT_ARGS%" == "w" (
    set OS_TYPE=win
    GOTO NEXT
)
if "%INPUT_ARGS%" == "linux" (
    set OS_TYPE=linux
    GOTO NEXT
)
if "%INPUT_ARGS%" == "win" (
    set OS_TYPE=win
    GOTO NEXT
)
GOTO END

:NEXT
echo build %OS_TYPE% platform version ...
set GO111MODULE=on
set CURR_PATH=%SCRIPTS_PATH%
set CURR_PATH=%CURR_PATH:\go\=*%
for /f "delims=* " %%i in ("%CURR_PATH%") do ( set GOPATH=%%i\go)
set GOMODCACHE=%GOPATH%\pkg\mod
set GOARCH=amd64
set GOOS=linux
set EXEFILE_EXTNAME=
if "%OS_TYPE%" == "win" (
  set GOOS=windows
  set EXEFILE_EXTNAME=.exe
)

cd /d %EBUS_PATH%
set BUILD_TIME=%date% %time%
for /F "tokens=* delims= " %%i in ('git rev-parse --short HEAD') do ( set COMMIT_HASH=%%i)
for /F "tokens=3,4 delims= " %%i in ('go version') do ( set GO_VERSION=%%i %%j )
set VPREFIX=main
set LDFLAGS=-ldflags "-s -w -X '%VPREFIX%.GoVersion=%GO_VERSION%' -X '%VPREFIX%.BuildTime=%BUILD_TIME%' -X '%VPREFIX%.CommitHash=%COMMIT_HASH%'"

cd /d %SCRIPTS_PATH%
if not exist "..\dist" (
    mkdir ..\dist
)
set OUT_PATH=%SCRIPTS_PATH%..\dist\%OS_TYPE%
set OUT_BIN_PATH=%OUT_PATH%\bin
set OUT_CONF_PATH=%OUT_PATH%\conf
if not exist "%OUT_PATH%" mkdir %OUT_PATH%
if not exist "%OUT_BIN_PATH%" mkdir %OUT_BIN_PATH%
if not exist "%OUT_CONF_PATH%" mkdir %OUT_CONF_PATH%

echo build ebus service...
cd .. && go build %LDFLAGS% -o %OUT_BIN_PATH%\ebus%EXEFILE_EXTNAME% && cd /d %SCRIPTS_PATH%
copy ..\conf\ebus.toml %OUT_CONF_PATH%\ /Y

:END
goto :eof
