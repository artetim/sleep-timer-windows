@echo off
set /p minutes=Enter minutes until sleep: 
set /a seconds=%minutes% * 60
echo Sleeping in %minutes% minutes...
timeout /t %seconds% /nobreak
rundll32.exe powrprof.dll,SetSuspendState Sleep