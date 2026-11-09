@echo off
chcp 65001 >nul
rem Build the TT Windows portable package: frontend + exe + empty config + example
rem + README + skills, zipped.
rem
rem NOTE: this file must stay ASCII-only. cmd.exe parses .bat bytes in the OEM
rem       codepage, so UTF-8 Chinese in a .bat breaks the parser (even after
rem       chcp 65001) -- lines get split mid-character and cmd tries to run the
rem       fragments. Chinese text belongs in README.md / docs/, not here.
rem       The merged project's three source scripts carried the same warning.
rem
rem NOTE: never package this machine's config.json (it holds real SSH/DB
rem       credentials). The portable package ships config.empty.json instead;
rem       users configure it via `tt serve` or by editing config.json.
rem NOTE: never package erp_data.db either (it holds the customer's table
rem       dictionary / schema / enterprise codes); users run `tt dict db sync`
rem       after configuring an environment.
rem NOTE: skills/ ships as a plain directory (not embedded in the binary) so
rem       users can edit it; `tt install skills` copies it to the target dir.
rem       After the merge there are six skills in there, installed together.
setlocal
cd /d "%~dp0"

set STAGE=dist\tt-portable
set GOPROXY=https://goproxy.cn,direct

rem Release version: read from the repo-root VERSION file, the ONE place a version
rem number is written. Injected into the binary via -ldflags (shown by `tt version`).
rem Never put a version literal back in this file: two copies drift, and then
rem `tt version` reports a version no release ever had.
set VERSION=
for /f "usebackq delims=" %%v in ("VERSION") do if not defined VERSION set VERSION=%%v
if not defined VERSION (echo MISSING VERSION file ^(repo root^) & exit /b 1)
rem Shape check via PowerShell, not findstr: `findstr /x` needs CRLF and this file
rem is LF-only, so it would reject a perfectly good version. A malformed version
rem here would quietly become a bogus `tt version` string and a bogus asset name.
powershell -NoProfile -Command "exit ([int]((Get-Content VERSION -Raw).Trim() -notmatch '^\d+\.\d+\.\d+$'))" || (echo BAD VERSION in VERSION file: %VERSION% -- expected MAJOR.MINOR.PATCH & exit /b 1)

rem Only the portable stage is ours to rebuild. dist\ also holds other products
rem (build_msi.bat keeps its staging and the
rem MSI there), so never wipe dist\ as a whole.
if exist "%STAGE%" rmdir /s /q "%STAGE%"
mkdir "%STAGE%"

echo [1/5] Building frontend (web/dist) ...
pushd web
if not exist node_modules (
    call npm install
    if errorlevel 1 (
        echo NPM INSTALL FAILED
        popd
        exit /b 1
    )
)
call npm run build
if errorlevel 1 (
    echo FRONTEND BUILD FAILED
    popd
    exit /b 1
)
popd
rem main.go embeds web/dist, so the frontend must exist before go build.
if not exist "web\dist\index.html" (echo MISSING web\dist\index.html & exit /b 1)

echo [2/5] Building tt.exe (v%VERSION%) ...
rem Built straight into the stage, never into the repo root: packaging must not
rem leave a tt.exe behind. The exe has to sit next to skills/ (tt install skills
rem reads <exe dir>\skills), and this stage gets skills/ in the next step.
call go build -trimpath -ldflags "-X tt/internal/cli.Version=%VERSION%" -o "%STAGE%\tt.exe" .
if errorlevel 1 (
    echo BUILD FAILED
    exit /b 1
)

echo [3/5] Staging EMPTY config.json / config.example.json / README.md / skills ...
copy /y config.empty.json "%STAGE%\config.json" >nul || (echo COPY empty config FAILED & exit /b 1)
copy /y config.example.json "%STAGE%\" >nul || (echo COPY config.example.json FAILED & exit /b 1)
copy /y README.md "%STAGE%\" >nul || (echo COPY README.md FAILED & exit /b 1)
xcopy /e /i /y /q skills "%STAGE%\skills" >nul || (echo COPY skills FAILED & exit /b 1)
if not exist "%STAGE%\tt.exe" (echo MISSING staged tt.exe & exit /b 1)
rem Portable marker: the CLI keeps its config inside the package instead of
rem writing to the user directory (see internal/config/paths.go, IsPortable).
type nul > "%STAGE%\.portable"

rem [3b/5] Staging the .tzs engine (C#). It is built SEPARATELY -- see engine\BUILD.md.
rem   Exactly three files, listed by name on purpose: engine\out\ also holds a dozen probe
rem   programs (Probe / Edit / AddField / RoundTrip / Test* / E2E) that must NOT ship, so an
rem   xcopy of the whole directory would put them all in the portable package.
rem   tzs-cli.exe (the engine's own C# client) is DELIBERATELY NOT shipped: tt has its own Go
rem   client (internal/dev/tzs) which is the only supported entry point. Shipping a second
rem   client would mean two command surfaces and two exit-code tables (tzs-cli is 0/1/2/3/4,
rem   tt is 0/1/2/4/5) for the same engine. It still builds in the repo
rem   (engine\test\tzs-cli.cs) because the engine's own acceptance harness drives it.
rem   Not built here: the engine needs csc + a bash script, and rebuilding it changes its MVID,
rem   which orphans every daemon started from the previous build (engine\BUILD.md explains).
set TZSENGINE=engine\out
for %%f in (tzs-server.exe TzsCli.dll TzsCli.Designer.dll) do (
    if not exist "%TZSENGINE%\%%f" (
        echo MISSING %TZSENGINE%\%%f -- build the engine first, see engine\BUILD.md
        exit /b 1
    )
)
if not exist "%STAGE%\tzs" mkdir "%STAGE%\tzs"
for %%f in (tzs-server.exe TzsCli.dll TzsCli.Designer.dll) do (
    copy /y "%TZSENGINE%\%%f" "%STAGE%\tzs\" >nul || (echo COPY engine %%f FAILED & exit /b 1)
)

rem [3c/5] Staging the designer payload the engine boots from.
rem   The engine loads the designer's assemblies from <its own directory>\designer (see
rem   engine\src\Designer\Bootstrap.cs), so the package carries them and a run depends on THIS
rem   package's designer version instead of on whatever the machine happens to have installed.
rem
rem   Only *.dll is copied -- and the repo's copy at engine\designer\ already IS exactly that set,
rem   with the exes left out: T100Designer.exe (the GUI, not referenced by the engine's assembly
rem   closure), AutoUpdater.exe + AppLimit's Sparkle updater (that one phones a vendor update
rem   channel -- we do not ship it), AutoUpdater.exe.config and a shell shortcut.
rem
rem   The default source is that committed copy, so packaging a release needs nothing configured
rem   and the version shipped is the version in the tree. TZSDESIGNER overrides it, for bundling
rem   a different designer version than the one committed.
set TZSDESIGNER_DEFAULT=engine\designer
if "%TZSDESIGNER%"=="" set TZSDESIGNER=%TZSDESIGNER_DEFAULT%
if not exist "%TZSDESIGNER%\SpecDesignerCommon.dll" (
    echo NOT A DESIGNER DIRECTORY: %TZSDESIGNER%
    echo   expected SpecDesignerCommon.dll in it. See engine\BUILD.md.
    exit /b 1
)
if not exist "%STAGE%\tzs\designer" mkdir "%STAGE%\tzs\designer"
xcopy /y /q "%TZSDESIGNER%\*.dll" "%STAGE%\tzs\designer\" >nul || (echo COPY designer FAILED & exit /b 1)
echo     designer staged from %TZSDESIGNER%

echo [4/5] Packing zip ...
rem Recursive zip (includes the skills/ subtree); keeps the tt-portable/ top-level
rem directory inside the archive. Go tool (tools/zip) replaced the old python
rem script (zip.py, removed 2026-10). NO fallback by design: a failed pack must
rem stop the build, not degrade silently. The old PowerShell fallback
rem (Compress-Archive) wrote backslash entry names, which breaks cross-platform
rem unzip tools -- that was a quality downgrade, not a backup.
go run ./tools/zip >nul
if errorlevel 1 (
    echo ZIP FAILED
    exit /b 1
)

echo [5/5] Done: dist\tt-portable.zip
dir /b dist
endlocal
