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
rem       users configure it via `tg serve` or by editing config.json.
rem NOTE: never package erp_data.db either (it holds the customer's table
rem       dictionary / schema / enterprise codes); users run `tt dict db sync`
rem       after configuring an environment.
rem NOTE: skills/ ships as a plain directory (not embedded in the binary) so
rem       users can edit it; `tg install skills` copies it to the target dir.
rem       There are three skills in there, installed together.
setlocal
cd /d "%~dp0"

set STAGE=dist\tg-portable
set GOPROXY=https://goproxy.cn,direct

rem Release version: read from the repo-root VERSION file, the ONE place a version
rem number is written. Injected into the binary via -ldflags (shown by `tg version`).
rem Never put a version literal back in this file: two copies drift, and then
rem `tg version` reports a version no release ever had.
set VERSION=
for /f "usebackq delims=" %%v in ("VERSION") do if not defined VERSION set VERSION=%%v
if not defined VERSION (echo MISSING VERSION file ^(repo root^) & exit /b 1)
rem Shape check via PowerShell, not findstr: `findstr /x` needs CRLF and this file
rem is LF-only, so it would reject a perfectly good version. A malformed version
rem here would quietly become a bogus `tg version` string and a bogus asset name.
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

echo [2/4] Building tg.exe (v%VERSION%) ...
rem Built straight into the stage, never into the repo root: packaging must not
rem leave a tg.exe behind. The exe has to sit next to skills/ (tg install skills
rem reads <exe dir>\skills), and this stage gets skills/ in the next step.
call go build -trimpath -ldflags "-X tg/internal/cli.Version=%VERSION%" -o "%STAGE%\tg.exe" .
if errorlevel 1 (
    echo BUILD FAILED
    exit /b 1
)

echo [3/4] Staging EMPTY config.json / config.example.json / README.md / LICENSE / skills / docs ...
copy /y config.empty.json "%STAGE%\config.json" >nul || (echo COPY empty config FAILED & exit /b 1)
copy /y config.example.json "%STAGE%\" >nul || (echo COPY config.example.json FAILED & exit /b 1)
copy /y README.md "%STAGE%\" >nul || (echo COPY README.md FAILED & exit /b 1)
rem License and third-party notices travel with the package: Apache-2.0 requires
rem shipping a copy of the License with the distributed work.
copy /y LICENSE "%STAGE%\" >nul || (echo COPY LICENSE FAILED & exit /b 1)
copy /y NOTICE "%STAGE%\" >nul || (echo COPY NOTICE FAILED & exit /b 1)
copy /y THIRD-PARTY-NOTICES.md "%STAGE%\" >nul || (echo COPY THIRD-PARTY-NOTICES.md FAILED & exit /b 1)
xcopy /e /i /y /q skills "%STAGE%\skills" >nul || (echo COPY skills FAILED & exit /b 1)
rem Genero BDL docs ship as one zip next to tg.exe, not as loose files: 5470 .md
rem would bloat the package and the installer, and nobody reads them from the
rem program dir anyway. `tg install bdl` unpacks it into the user's data dir
rem (or into config bdldoc.dir). Missing zip is a hard error: the whole point of
rem shipping it is that a fresh install can run `tg install bdl` right away.
if not exist "BDL-Markdown.zip" (echo MISSING BDL-Markdown.zip ^(repo root^) & exit /b 1)
copy /y BDL-Markdown.zip "%STAGE%\" >nul || (echo COPY BDL-Markdown.zip FAILED & exit /b 1)
rem TOP GP product manual, same treatment (`tg install gpdoc`): 237 .md in one zip.
if not exist "TOP-GP-Markdown.zip" (echo MISSING TOP-GP-Markdown.zip ^(repo root^) & exit /b 1)
copy /y TOP-GP-Markdown.zip "%STAGE%\" >nul || (echo COPY TOP-GP-Markdown.zip FAILED & exit /b 1)
if not exist "%STAGE%\tg.exe" (echo MISSING staged tg.exe & exit /b 1)
rem Portable marker: the CLI keeps its config inside the package instead of
rem writing to the user directory (see internal/config/paths.go, IsPortable).
type nul > "%STAGE%\.portable"

echo [4/4] Packing zip ...
rem Recursive zip (includes the skills/ subtree); keeps the tg-portable/ top-level
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

echo Done: dist\tg-portable.zip
dir /b dist
endlocal
