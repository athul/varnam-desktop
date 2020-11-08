CheckNetIsolation.exe LoopbackExempt -a -n="Microsoft.Win32WebViewHost_cw5n1h2txyewy"

SET VARNAM_SYMBOLS_DIR=%APPDATA%\varnam\vst
SET VARNAM_SUGGESTIONS_DIR=%APPDATA%\varnam\suggestions

IF NOT EXIST "%VARNAM_SYMBOLS_DIR%" (
  mkdir "%VARNAM_SYMBOLS_DIR%"
)

IF NOT EXIST "%VARNAM_SUGGESTIONS_DIR%" (
  mkdir "%VARNAM_SUGGESTIONS_DIR%"
)
