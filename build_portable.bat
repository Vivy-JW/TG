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
rem       There are three skills in there, installed together.
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

echo [1/4] Building frontend (web/dist) ...
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

echo [2/4] Building tt.exe (v%VERSION%) ...
rem Built straight into the stage, never into the repo root: packaging must not
rem leave a tt.exe behind. The exe has to sit next to skills/ (tt install skills
rem reads <exe dir>\skills), and this stage gets skills/ in the next step.
call go build -trimpath -ldflags "-X tg/internal/cli.Version=%VERSION%" -o "%STAGE%\tt.exe" .
if errorlevel 1 (
    echo BUILD FAILED
    exit /b 1
)

echo [3/4] Staging EMPTY config.json / config.example.json / README.md / skills ...
copy /y config.empty.json "%STAGE%\config.json" >nul || (echo COPY empty config FAILED & exit /b 1)
copy /y config.example.json "%STAGE%\" >nul || (echo COPY config.example.json FAILED & exit /b 1)
copy /y README.md "%STAGE%\" >nul || (echo COPY README.md FAILED & exit /b 1)
xcopy /e /i /y /q skills "%STAGE%\skills" >nul || (echo COPY skills FAILED & exit /b 1)
if not exist "%STAGE%\tt.exe" (echo MISSING staged tt.exe & exit /b 1)
rem Portable marker: the CLI keeps its config inside the package instead of
rem writing to the user directory (see internal/config/paths.go, IsPortable).
type nul > "%STAGE%\.portable"

echo [4/4] Packing zip ...
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

echo Done: dist\tt-portable.zip
dir /b dist
endlocal
