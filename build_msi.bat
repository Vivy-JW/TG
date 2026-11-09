@echo off
rem Console codepage for the Go tools' UTF-8 output (same as build_portable.bat).
rem The file itself must stay ASCII-only -- see the note below.
chcp 65001 >nul
rem Build the **user-level** MSI installer for TT.
rem
rem NOTE: this file must stay ASCII-only. cmd.exe parses .bat bytes in the OEM
rem       codepage, so UTF-8 Chinese in a .bat breaks the parser (even after
rem       chcp 65001) -- lines get split mid-character and cmd tries to run the
rem       fragments. Chinese text belongs in README.md / docs/, not here.
rem
rem What it does: package the same payload as the portable zip (tt.exe + skills/
rem + README + config.example.json) into an MSI that installs per-user, without
rem admin rights. See installer/tt.wxs for why per-user.
rem
rem Difference from the portable zip: the MSI deliberately does NOT ship
rem .portable -- an installed TT must keep its config in
rem %APPDATA%\T100\tt\config.json (user-writable), not next to the exe (which
rem lives under %LOCALAPPDATA%\Programs and is not where user data belongs).
rem
rem Requires the WiX v3 toolset (candle.exe + light.exe + heat.exe). Point
rem WIX_BIN at it, or drop the binaries in the default location below.
rem   default: D:\tt-build-tools\wix3
setlocal
cd /d "%~dp0"

set STAGE=dist\tt-msi
set OBJ=dist\msi-obj

rem Release version: read from the repo-root VERSION file, the ONE place a version
rem number is written (build_portable.bat reads the same file and validates its
rem shape; `make version-check` pins it against web/package.json).
set VERSION=
for /f "usebackq delims=" %%v in ("VERSION") do if not defined VERSION set VERSION=%%v
if not defined VERSION (echo MISSING VERSION file ^(repo root^) & exit /b 1)

if "%WIX_BIN%"=="" set WIX_BIN=D:\tt-build-tools\wix3
if not exist "%WIX_BIN%\candle.exe" (
    echo WIX NOT FOUND: %WIX_BIN%\candle.exe
    echo Set WIX_BIN to the WiX v3 binaries directory ^(candle.exe / light.exe / heat.exe^).
    exit /b 1
)
if not exist "installer\icon.ico" (echo MISSING installer\icon.ico & exit /b 1)

echo [1/4] Building the payload ^(same as the portable package^) ...
rem Call by absolute path: with NoDefaultCurrentDirectoryInExePath set, a bare
rem file name is not resolved from the current directory.
call "%~dp0build_portable.bat"
if errorlevel 1 (echo PORTABLE BUILD FAILED & exit /b 1)

echo [2/4] Staging without .portable / config.json ...
if exist "%STAGE%" rmdir /s /q "%STAGE%"
mkdir "%STAGE%" || exit /b 1
xcopy /e /i /y /q dist\tt-portable "%STAGE%" >nul || (echo STAGE COPY FAILED & exit /b 1)
rem .portable: the installed copy must NOT be portable -- its config belongs in
rem   %APPDATA%\T100\tt\config.json (user-writable), not next to the exe.
rem config.json: ship no config at all. The installer would otherwise drop an
rem   empty one in the install dir, which "tt" can pick up as a legacy fallback
rem   location -- a confusing second config that looks like the real one.
if exist "%STAGE%\.portable" del /q "%STAGE%\.portable"
if exist "%STAGE%\config.json" del /q "%STAGE%\config.json"
if not exist "%STAGE%\tt.exe" (echo MISSING staged tt.exe & exit /b 1)
if not exist "%STAGE%\config.example.json" (echo MISSING staged config.example.json & exit /b 1)

echo [3/4] Harvesting files into a component group ...
if exist "%OBJ%" rmdir /s /q "%OBJ%"
mkdir "%OBJ%" || exit /b 1
"%WIX_BIN%\heat.exe" dir "%STAGE%" -cg StagedFiles -gg -g1 -sfrag -srd -sreg ^
    -dr INSTALLFOLDER -var var.SourceDir -out "dist\msi-files.wxs"
if errorlevel 1 (echo HEAT FAILED & exit /b 1)
rem heat does not emit the RemoveFolder entries ICE64 requires for per-user
rem installs (see tools\wixremovefolders); patch them in before compiling.
go run ./tools/wixremovefolders dist\msi-files.wxs
if errorlevel 1 (echo REMOVEFOLDER PATCH FAILED & exit /b 1)

echo [4/4] Compiling and linking TT-%VERSION%-x64.msi ...
rem One candle call per source, each with an explicit .wixobj path: passing a
rem directory to -out needs a trailing backslash, and cmd hands "%OBJ%\" to the
rem program as a literal quote (the backslash escapes it), which merges the rest
rem of the command line into the path. Explicit file names sidestep that.
"%WIX_BIN%\candle.exe" -nologo -arch x64 -dVersion=%VERSION% ^
    -dSourceDir="%STAGE%" -dIconPath="installer\icon.ico" ^
    -out "%OBJ%\tt.wixobj" installer\tt.wxs
if errorlevel 1 (echo CANDLE FAILED & exit /b 1)
"%WIX_BIN%\candle.exe" -nologo -arch x64 -dVersion=%VERSION% ^
    -dSourceDir="%STAGE%" -dIconPath="installer\icon.ico" ^
    -out "%OBJ%\msi-files.wixobj" dist\msi-files.wxs
if errorlevel 1 (echo CANDLE FAILED & exit /b 1)
"%WIX_BIN%\light.exe" -nologo -out "dist\TT-%VERSION%-x64.msi" ^
    "%OBJ%\tt.wixobj" "%OBJ%\msi-files.wixobj"
if errorlevel 1 (echo LIGHT FAILED & exit /b 1)

echo [done] dist\TT-%VERSION%-x64.msi
dir /b dist
endlocal
