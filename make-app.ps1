# Build VLessBar for macOS 10.13 (High Sierra, Intel) from Windows.
# Produces: dist\VLessBar.app and dist\VLessBar-macos10.13.zip
#
# Requires: Go 1.20 (golang.org/dl/go1.20), and resources\xray (built with Go 1.20).
# Usage:   powershell -ExecutionPolicy Bypass -File .\make-app.ps1

$ErrorActionPreference = 'Stop'
$Root   = $PSScriptRoot
$Go     = 'C:\Users\stepa\go\bin\go1.20.exe'
$Engine = Join-Path $Root 'engine'
$XrayBin= Join-Path $Root 'resources\xray'
$Dist   = Join-Path $Root 'dist'
$App    = Join-Path $Dist 'VLessBar.app'
$Contents = Join-Path $App 'Contents'
$MacOS  = Join-Path $Contents 'MacOS'
$Res    = Join-Path $Contents 'Resources'

if (-not (Test-Path $Go))      { throw "Go 1.20 not found at $Go" }
if (-not (Test-Path $XrayBin)) { throw "xray binary not found at $XrayBin (build it with Go 1.20 first)" }

# 1. clean
if (Test-Path $Dist) { Remove-Item -Recurse -Force $Dist }
New-Item -ItemType Directory -Force -Path $MacOS, $Res | Out-Null

# 2. build engine for darwin/amd64 with Go 1.20
$env:GOOS='darwin'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
Write-Host "building engine..."
Push-Location $Engine
try {
	& $Go build -trimpath -ldflags "-s -w" -o (Join-Path $MacOS 'VLessBar') .
	if ($LASTEXITCODE -ne 0) { throw "engine build failed" }
} finally {
	Pop-Location
}

# 3. assemble bundle
Copy-Item (Join-Path $Root 'app\Info.plist') (Join-Path $Contents 'Info.plist') -Force
Copy-Item (Join-Path $Root 'app\ui.applescript') (Join-Path $Res 'ui.applescript') -Force
Copy-Item $XrayBin (Join-Path $Res 'xray') -Force

# 4. zip with preserved Unix permissions (exec bit set on unpack -> no chmod for user)
$zip = Join-Path $Dist 'VLessBar-macos10.13.zip'
$py  = 'C:\Users\stepa\AppData\Local\Programs\Python\Python312\python.exe'
if (-not (Test-Path $py)) { $py = 'python' }
& $py (Join-Path $Root 'make_zip.py') $App $zip
if ($LASTEXITCODE -ne 0) { throw "zip creation failed" }

# 5. install notes
$notes = @"
VLessBar - macOS 10.13 (High Sierra, Intel)

УСТАНОВКА (без Терминала):
1. Распакуйте VLessBar-macos10.13.zip (двойной клик) - появится VLessBar.app.
   Права на запуск уже выставлены внутри архива, chmod не нужен.
2. Перетащите VLessBar.app в папку «Программы», ЛИБО просто запустите его из
   Загрузок - приложение само предложит установиться (в «Программы», а если нет
   прав администратора - в личную папку ~/Программы, без запроса пароля).
3. Первый запуск после скачивания из интернета:
   - Если macOS пишет «неизвестный разработчик»: правый клик по VLessBar.app ->
     «Открыть» -> «Открыть» (один раз). Это обычное поведение для приложений
     без платной подписи Apple.
   - Если приложение запущено из Загрузок и предлагает установку, нажмите
     «Установить» - оно скопируется в «Программы» и перезапустится оттуда.
   (Если файл пришёл по AirDrop/с флешки, карантина нет - просто двойной клик.)

ИСПОЛЬЗОВАНИЕ:
- «Указать ссылку подписки» -> вставьте https-ссылку подписки.
  HWID генерируется автоматически и отправляется как x-hwid (совместимо с Happ).
- «Сменить сервер» -> выберите сервер.
- «Подключить» / «Отключить».

ПРИМЕЧАНИЯ:
- Прокси: HTTP 127.0.0.1:10809, SOCKS 127.0.0.1:10808 (через networksetup).
  Если не сработало, разово в Терминале:
     sudo networksetup -setwebproxy "Wi-Fi" 127.0.0.1 10809
     sudo networksetup -setsecurewebproxy "Wi-Fi" 127.0.0.1 10809
     sudo networksetup -setsocksfirewallproxy "Wi-Fi" 127.0.0.1 10808
- Состояние: ~/.vlessbar/  (state.json, config.json, xray.log)
"@
Set-Content -Path (Join-Path $Dist 'README-install.txt') -Value $notes -Encoding UTF8


Write-Host "done:"
Write-Host "  $App"
Write-Host "  $zip"
