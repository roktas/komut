:; KOMUT_HOOK_HOST=claude exec "$CLAUDE_PLUGIN_ROOT/bin/x" --hook
@echo off
set "KOMUT_HOOK_HOST=claude"
call "%CLAUDE_PLUGIN_ROOT%\bin\x.cmd" --hook
exit /b %ERRORLEVEL%
