@echo off
setlocal

set "arch=%PROCESSOR_ARCHITEW6432%"
if "%arch%"=="" set "arch=%PROCESSOR_ARCHITECTURE%"

if /I "%arch%"=="AMD64" (
    "%~dp0..\libexec\x\windows-amd64\x.exe" %*
    exit /b %ERRORLEVEL%
)

echo x: unsupported platform: Windows %arch% 1>&2
exit /b 126
