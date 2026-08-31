@echo off
setlocal

set "arch=%PROCESSOR_ARCHITEW6432%"
if "%arch%"=="" set "arch=%PROCESSOR_ARCHITECTURE%"

if /I not "%arch%"=="AMD64" goto unsupported

"%~dp0..\libexec\x\windows-amd64\x.exe" %*
exit /b %ERRORLEVEL%

:unsupported
echo x: unsupported platform: Windows %arch% 1>&2
exit /b 126
