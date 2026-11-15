@echo off
setlocal
cd /d "%~dp0"
set "GOOS=windows"
set "GOARCH=amd64"
set "CGO_ENABLED=0"

echo Running timer logic tests...
go test ./... || exit /b 1

echo Building the Windows GUI...
go build -trimpath -ldflags="-s -w -H=windowsgui" -o SleepTimer_temp.exe . || exit /b 1

echo Embedding app icon...
python embed_icon.py SleepTimer_temp.exe SleepTimer.ico SleepTimer_v1.0.1.exe || exit /b 1

del SleepTimer_temp.exe
echo Built SleepTimer_v1.0.1.exe
