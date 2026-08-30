:; exec "$CLAUDE_PLUGIN_ROOT/bin/x" --hook
@echo off
call "%CLAUDE_PLUGIN_ROOT%\bin\x.cmd" --hook
exit /b %ERRORLEVEL%
