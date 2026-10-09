//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// A small native Win32 GUI built with the stdlib only (no cgo, no third-party
// packages), so a single VLessBar.exe runs on older Windows (7/8/8.1/10/11).

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")
	k32    = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassExW = user32.NewProc("RegisterClassExW")
	pCreateWindowExW  = user32.NewProc("CreateWindowExW")
	pDefWindowProcW   = user32.NewProc("DefWindowProcW")
	pGetMessageW      = user32.NewProc("GetMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pDispatchMessageW = user32.NewProc("DispatchMessageW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pSendMessageW     = user32.NewProc("SendMessageW")
	pPostMessageW     = user32.NewProc("PostMessageW")
	pShowWindow       = user32.NewProc("ShowWindow")
	pUpdateWindow     = user32.NewProc("UpdateWindow")
	pDestroyWindow    = user32.NewProc("DestroyWindow")
	pSetWindowTextW   = user32.NewProc("SetWindowTextW")
	pGetWindowTextW   = user32.NewProc("GetWindowTextW")
	pEnableWindow     = user32.NewProc("EnableWindow")
	pSetFocus         = user32.NewProc("SetFocus")
	pGetModuleHandleW = k32.NewProc("GetModuleHandleW")
	pGetStockObject   = gdi32.NewProc("GetStockObject")
)

const (
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsBorder           = 0x00800000
	wsVScroll          = 0x00200000
	wsTabStop          = 0x00010000
	wsOverlappedWindow = 0x00CF0000
	wsPopupWindow      = 0x80000000
	wsCaption          = 0x00C00000
	esMultiline        = 0x0004
	esAutoVScroll      = 0x0040
	esAutoHScroll      = 0x0080
	esReadOnly         = 0x0800
	lbsNotify          = 0x0001
	lbsHasStrings      = 0x0040
	csHRedraw          = 0x0002
	csVRedraw          = 0x0001
	swShow             = 5
	cwUseDefault       = 0x80000000

	wmCreate  = 0x0001
	wmDestroy = 0x0002
	wmClose   = 0x0010
	wmCommand = 0x0111
	wmSetFont = 0x0030

	wmAppUpdate = 0x8001 // custom: settings window refresh after async work

	bsAutoCheckbox = 0x00000003
	bmGetCheck     = 0x00F0
	bmSetCheck     = 0x00F1
	bstChecked     = 1

	lbAddString    = 0x0180
	lbGetCurSel    = 0x0188
	lbResetContent = 0x0184

	lbnDblClk      = 2
	defaultGuiFont = 17
	colorWindowBg  = 6 // COLOR_WINDOW + 1
)

const (
	idStatus = 100
	idList   = 101
	idEdit   = 102
	idHint   = 103

	idUpdate = 200
	idAddSub = 201
	idHwid   = 202
	idStats  = 203
	idQuit   = 204
	idUp     = 205
	idDown   = 206

	idSettings = 207
	idConn     = 208
	idPing     = 209
	idToggle   = 210

	idInputEdit   = 300
	idInputOK     = 301
	idInputCancel = 302

	idSetVer     = 400
	idSetCore    = 401
	idSetLan     = 402
	idSetLanInfo = 403
	idSetCheck   = 404
	idSetUpdate  = 405
	idSetStatus  = 406
	idSetClose   = 407
	idSetConn    = 408
	idSetConnInfo = 409
)

var (
	hwndMain  uintptr
	hInstance uintptr
	hFont     uintptr

	hStatus uintptr
	hList   uintptr
	hEdit   uintptr

	hwndInput  uintptr
	hInputEdit uintptr

	hBtnToggle uintptr

	hwndSettings              uintptr
	hSetLan                   uintptr
	hSetLanInfo, hSetStatus, hSetConnInfo uintptr
	hSetUpdateBtn             uintptr
	updStatus, updURL, updVer string
)

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type msgW struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func sendW(h uintptr, m uint32, w, l uintptr) uintptr {
	r, _, _ := pSendMessageW.Call(h, uintptr(m), w, l)
	return r
}

func setText(h uintptr, s string) {
	p := u16(s)
	pSetWindowTextW.Call(h, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

func createCtl(parent uintptr, cls, text string, style uintptr, x, y, cx, cy int32, id uintptr) uintptr {
	pc := u16(cls)
	pt := u16(text)
	h, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(pc)),
		uintptr(unsafe.Pointer(pt)),
		style,
		uintptr(x), uintptr(y), uintptr(cx), uintptr(cy),
		parent, id, hInstance, 0,
	)
	runtime.KeepAlive(pc)
	runtime.KeepAlive(pt)
	if hFont != 0 {
		sendW(h, wmSetFont, hFont, 1)
	}
	return h
}

func registerClass(name string, proc uintptr) {
	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		style:         csHRedraw | csVRedraw,
		lpfnWndProc:   proc,
		hInstance:     hInstance,
		hbrBackground: colorWindowBg,
		lpszClassName: u16(name),
	}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(wc.lpszClassName)
}

func createControls(hwnd uintptr) {
	hFont, _, _ = pGetStockObject.Call(defaultGuiFont)
	hStatus = createCtl(hwnd, "STATIC", "Загрузка…", wsChild|wsVisible, 12, 12, 424, 20, idStatus)
	hList = createCtl(hwnd, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify|lbsHasStrings, 12, 38, 424, 150, idList)
	hEdit = createCtl(hwnd, "EDIT", "", wsChild|wsVisible|wsBorder|wsVScroll|esMultiline|esReadOnly|esAutoVScroll, 12, 194, 424, 96, idEdit)
	createCtl(hwnd, "STATIC", "Двойной клик по серверу — выбрать его.", wsChild|wsVisible, 12, 294, 424, 16, idHint)

	btn := func(text string, x, y, cx, cy int32, id uintptr) {
		createCtl(hwnd, "BUTTON", text, wsChild|wsVisible|wsTabStop, x, y, cx, cy, id)
	}
	btn("Обновить подписку", 12, 314, 130, 26, idUpdate)
	btn("Ввести ссылку", 148, 314, 130, 26, idAddSub)
	btn("Мой IP", 284, 314, 72, 26, idConn)
	btn("Пинг", 362, 314, 74, 26, idPing)
	hBtnToggle = createCtl(hwnd, "BUTTON", "Подключить", wsChild|wsVisible|wsTabStop, 12, 346, 338, 30, idToggle)
	btn("Выход", 356, 346, 80, 30, idQuit)
	btn("Настройки", 12, 382, 140, 28, idSettings)
}

// --- actions -------------------------------------------------------------

func showOut(s string) { setText(hEdit, strings.TrimRight(s, "\n")) }

// Public IP, cached and refreshed in background.
var cachedPublicIP string

func refreshStatus() {
	s, _ := loadState()
	state := "ВЫКЛ"
	if xrayRunning() {
		state = "ВКЛ"
	}
	name := "(нет сервера)"
	addr := ""
	if s.Selected >= 0 && s.Selected < len(s.Servers) {
		name = s.Servers[s.Selected].Name
		addr = s.Servers[s.Selected].Address
	}
	txt := fmt.Sprintf("%s  |  %s  |  серверов: %d", state, name, len(s.Servers))
	if addr != "" {
		txt += "  |  " + addr
	}
	if cachedPublicIP != "" {
		txt += "  |  IP: " + cachedPublicIP
	}
	if s.HwidLimit {
		txt += "  |  лимит устройств"
	}
	setText(hStatus, txt)

	// Toggle button follows the tunnel state.
	if hBtnToggle != 0 {
		if xrayRunning() {
			setText(hBtnToggle, "Отключить")
		} else {
			setText(hBtnToggle, "Подключить")
		}
	}
}

// refreshPublicIP fetches the public IP in background and updates the status.
func refreshPublicIP() {
	go func() {
		ip, err := getPublicIP()
		if err != nil {
			return
		}
		cachedPublicIP = ip
		refreshStatus()
	}()
}

func fillList() {
	sendW(hList, lbResetContent, 0, 0)
	s, err := loadState()
	if err != nil {
		return
	}
	for i, srv := range s.Servers {
		label := strconv.Itoa(i) + "  " + srv.Name
		if i == s.Selected {
			label = "> " + label
		}
		p := u16(label)
		sendW(hList, lbAddString, 0, uintptr(unsafe.Pointer(p)))
		runtime.KeepAlive(p)
	}
}

func actUpdate() {
	s, err := loadState()
	if err != nil {
		showOut("ошибка: " + err.Error())
		return
	}
	if s.SubURL == "" {
		showOut("Ссылка подписки не задана.\nНажмите «Ввести ссылку».")
		return
	}
	out, err := capture(func() error { return refreshSubscription(s) })
	if err != nil {
		showOut(out + "\nОШИБКА: " + err.Error())
	} else {
		showOut(out)
	}
	fillList()
	refreshStatus()
}

func actShow(kind string) {
	var out string
	var err error
	switch kind {
	case "hwid":
		out, err = capture(cmdHwid)
	case "status":
		out, err = capture(cmdStatus)
	}
	if err != nil {
		showOut("ошибка: " + err.Error())
		return
	}
	showOut(out)
}

func actUp() {
	out, err := capture(cmdUp)
	if err != nil {
		showOut("ошибка: " + err.Error() + "\n" + out)
	} else {
		showOut(out)
	}
	refreshStatus()
}

func actDown() {
	out, err := capture(cmdDown)
	if err != nil {
		showOut("ошибка: " + err.Error() + "\n" + out)
	} else {
		showOut(out)
	}
	refreshStatus()
}

func actConn() {
	out, err := capture(cmdConn)
	if err != nil {
		showOut("ошибка: " + err.Error() + "\n" + out)
	} else {
		showOut(out)
	}
	refreshPublicIP()
	refreshStatus()
}

func actPing() {
	out, err := capture(cmdPingThroughTunnel)
	if err != nil {
		showOut("ошибка: " + err.Error() + "\n" + out)
	} else {
		showOut(out)
	}
}

func actSelect() {
	sel := sendW(hList, lbGetCurSel, 0, 0)
	if sel == ^uintptr(0) { // LB_ERR
		return
	}
	s, err := loadState()
	if err != nil {
		return
	}
	i := int(sel)
	if i < 0 || i >= len(s.Servers) {
		return
	}
	s.Selected = i
	_ = saveState(s)
	fillList()
	refreshStatus()
}

func actAddWith(url string) {
	url = strings.TrimSpace(url)
	if url == "" {
		return
	}
	s, err := loadState()
	if err != nil {
		showOut("ошибка: " + err.Error())
		return
	}
	s.SubURL = url
	if _, err := ensureHwid(s); err != nil {
		showOut("ошибка: " + err.Error())
		return
	}
	if err := saveState(s); err != nil {
		showOut("ошибка: " + err.Error())
		return
	}
	actUpdate()
}

// --- input dialog --------------------------------------------------------

func openInput(prompt, def string) {
	pc := u16("VLessBarInputWnd")
	pt := u16("Ссылка подписки")
	hwndInput, _, _ = pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(pc)),
		uintptr(unsafe.Pointer(pt)),
		wsPopupWindow|wsCaption,
		uintptr(80), uintptr(80), 480, 150,
		hwndMain, 0, hInstance, 0,
	)
	runtime.KeepAlive(pc)
	runtime.KeepAlive(pt)
	if hwndInput == 0 {
		return
	}
	createCtl(hwndInput, "STATIC", prompt, wsChild|wsVisible, 12, 12, 456, 34, 0)
	hInputEdit = createCtl(hwndInput, "EDIT", def, wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll, 12, 50, 456, 24, idInputEdit)
	createCtl(hwndInput, "BUTTON", "OK", wsChild|wsVisible|wsTabStop, 288, 84, 86, 26, idInputOK)
	createCtl(hwndInput, "BUTTON", "Отмена", wsChild|wsVisible|wsTabStop, 382, 84, 86, 26, idInputCancel)

	// Center on the main window.
	// (Skip GetWindowRect; a fixed position near the parent is fine.)
	pEnableWindow.Call(hwndMain, 0)
	pShowWindow.Call(hwndInput, swShow)
	pSetFocus.Call(hInputEdit)
}

func closeInput() {
	pEnableWindow.Call(hwndMain, 1)
	pDestroyWindow.Call(hwndInput)
	hwndInput = 0
}

func inputWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmCommand:
		switch int(wp & 0xffff) {
		case idInputOK:
			buf := make([]uint16, 8192)
			pGetWindowTextW.Call(hInputEdit, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
			url := syscall.UTF16ToString(buf)
			closeInput()
			// Network fetch in background, otherwise the UI freezes.
			go actAddWith(url)
			return 0
		case idInputCancel:
			closeInput()
			return 0
		}
	case wmClose:
		closeInput()
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

// --- settings window -----------------------------------------------------

func lanInfoText(on bool) string {
	if !on {
		return "LAN выключен: доступ только с этого компьютера."
	}
	ip := localIPv4()
	if ip == "" {
		ip = "<IP>"
	}
	return "LAN включён. Другие устройства укажите HTTP-прокси " +
		lanGatewayURL() + " (SOCKS " + ip + ":10808). Разрешите в брандмауэре."
}

func openSettings() {
	if hwndSettings != 0 {
		pSetFocus.Call(hwndSettings)
		return
	}
	pc := u16("VLessBarSettingsWnd")
	pt := u16("Настройки")
	hwndSettings, _, _ = pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(pc)),
		uintptr(unsafe.Pointer(pt)),
		wsPopupWindow|wsCaption,
		0, 0, 480, 420,
		hwndMain, 0, hInstance, 0,
	)
	runtime.KeepAlive(pc)
	runtime.KeepAlive(pt)
	if hwndSettings == 0 {
		return
	}

	createCtl(hwndSettings, "STATIC", "Версия приложения: "+version, wsChild|wsVisible, 14, 14, 450, 18, idSetVer)
	createCtl(hwndSettings, "STATIC", "Ядро Xray: "+xrayVersion(), wsChild|wsVisible, 14, 34, 450, 18, idSetCore)

	s, _ := loadState()
	hSetLan = createCtl(hwndSettings, "BUTTON", "Разрешить LAN (доступ с других устройств)",
		wsChild|wsVisible|wsTabStop|bsAutoCheckbox, 14, 66, 450, 20, idSetLan)
	if s.AllowLAN {
		sendW(hSetLan, bmSetCheck, bstChecked, 0)
	}
	hSetLanInfo = createCtl(hwndSettings, "STATIC", lanInfoText(s.AllowLAN), wsChild|wsVisible, 14, 90, 450, 34, idSetLanInfo)

	createCtl(hwndSettings, "BUTTON", "Проверить обновление", wsChild|wsVisible|wsTabStop, 14, 130, 180, 26, idSetCheck)
	hSetUpdateBtn = createCtl(hwndSettings, "BUTTON", "Обновить приложение", wsChild|wsVisible|wsTabStop, 204, 130, 180, 26, idSetUpdate)
	pEnableWindow.Call(hSetUpdateBtn, 0)
	hSetStatus = createCtl(hwndSettings, "STATIC", "Обновление меняет только приложение; ядро Xray не затрагивается.",
		wsChild|wsVisible, 14, 166, 450, 56, idSetStatus)

	createCtl(hwndSettings, "BUTTON", "Проверить связь", wsChild|wsVisible|wsTabStop, 14, 232, 180, 26, idSetConn)
	createCtl(hwndSettings, "STATIC", "IP (напрямую) / IP (через туннель) / пинг сервера",
		wsChild|wsVisible, 204, 236, 264, 22, 0)
	hSetConnInfo = createCtl(hwndSettings, "EDIT", "",
		wsChild|wsVisible|wsBorder|esMultiline|esReadOnly|esAutoVScroll, 14, 264, 450, 64, idSetConnInfo)

	createCtl(hwndSettings, "BUTTON", "Закрыть", wsChild|wsVisible|wsTabStop, 350, 340, 114, 30, idSetClose)

	pEnableWindow.Call(hwndMain, 0)
	pShowWindow.Call(hwndSettings, swShow)
}

func closeSettings() {
	pEnableWindow.Call(hwndMain, 1)
	pDestroyWindow.Call(hwndSettings)
	hwndSettings = 0
}

func settingsWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmCommand:
		switch int(wp & 0xffff) {
		case idSetLan:
			s, _ := loadState()
			s.AllowLAN = sendW(hSetLan, bmGetCheck, 0, 0) == bstChecked
			_ = saveState(s)
			setText(hSetLanInfo, lanInfoText(s.AllowLAN))
			refreshStatus()
			return 0
		case idSetCheck:
			setText(hSetStatus, "Проверяю обновление…")
			go func() {
				updStatus, updURL, updVer = checkUpdate()
				pPostMessageW.Call(hwnd, wmAppUpdate, 0, 0)
			}()
			return 0
		case idSetUpdate:
			if updURL == "" {
				return 0
			}
			setText(hSetStatus, "Скачиваю обновление… приложение перезапустится.")
			pEnableWindow.Call(hSetUpdateBtn, 0)
			go func() {
				if err := applyUpdate(updURL); err != nil {
					updStatus = "Ошибка обновления: " + err.Error()
					pPostMessageW.Call(hwnd, wmAppUpdate, 0, 0)
					return
				}
				time.Sleep(400 * time.Millisecond)
				os.Exit(0)
			}()
			return 0
		case idSetClose:
			closeSettings()
			return 0
		case idSetConn:
			setText(hSetConnInfo, "Проверяю…")
			go func() {
				out, err := capture(cmdConn)
				if err != nil {
					out = "ошибка: " + err.Error()
				}
				setText(hSetConnInfo, out)
			}()
			return 0
		}
	case wmAppUpdate:
		setText(hSetStatus, updStatus)
		if updURL != "" {
			pEnableWindow.Call(hSetUpdateBtn, 1)
		}
		return 0
	case wmClose:
		closeSettings()
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

// --- main window ---------------------------------------------------------

func mainWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmCreate:
		hwndMain = hwnd
		createControls(hwnd)
		return 0
	case wmCommand:
		switch int(wp & 0xffff) {
		case idUpdate:
			go actUpdate()
		case idAddSub:
			openInput("Вставьте ссылку подписки (https://…):", "")
		case idConn:
			go actConn()
		case idPing:
			go actPing()
		case idToggle:
			if xrayRunning() {
				go actDown()
			} else {
				go actUp()
			}
		case idSettings:
			openSettings()
		case idQuit:
			pDestroyWindow.Call(hwnd)
		case idList:
			if int((wp>>16)&0xffff) == lbnDblClk {
				actSelect()
			}
		}
		return 0
	case wmClose:
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

// launchGUI is called when the engine is started with no arguments.
func launchGUI() error {
	hInstance, _, _ = pGetModuleHandleW.Call(0)

	mainProc := syscall.NewCallback(mainWndProc)
	inputProc := syscall.NewCallback(inputWndProc)
	settingsProc := syscall.NewCallback(settingsWndProc)
	registerClass("VLessBarMainWnd", mainProc)
	registerClass("VLessBarInputWnd", inputProc)
	registerClass("VLessBarSettingsWnd", settingsProc)

	pc := u16("VLessBarMainWnd")
	pt := u16("VLessBar")
	h, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(pc)),
		uintptr(unsafe.Pointer(pt)),
		wsOverlappedWindow,
		uintptr(cwUseDefault), uintptr(cwUseDefault), 460, 470,
		0, 0, hInstance, 0,
	)
	runtime.KeepAlive(pc)
	runtime.KeepAlive(pt)
	if h == 0 {
		return fmt.Errorf("CreateWindowExW failed")
	}
	hwndMain = h

	fillList()
	refreshStatus()
	refreshPublicIP()

	pShowWindow.Call(h, swShow)
	pUpdateWindow.Call(h)

	var m msgW
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

// Unused-import guard for potential future use.
var _ = strconv.Itoa
