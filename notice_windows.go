//go:build windows

package main

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	gdi32               = syscall.NewLazyDLL("gdi32.dll")
	getModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	registerClassExW    = user32.NewProc("RegisterClassExW")
	createWindowExW     = user32.NewProc("CreateWindowExW")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	setWindowPos        = user32.NewProc("SetWindowPos")
	destroyWindow       = user32.NewProc("DestroyWindow")
	loadIconW           = user32.NewProc("LoadIconW")
	sendMessageW        = user32.NewProc("SendMessageW")
	getMessageW         = user32.NewProc("GetMessageW")
	translateMessage    = user32.NewProc("TranslateMessage")
	dispatchMessageW    = user32.NewProc("DispatchMessageW")
	postQuitMessage     = user32.NewProc("PostQuitMessage")
	getSystemMetrics    = user32.NewProc("GetSystemMetrics")
	defWindowProcW      = user32.NewProc("DefWindowProcW")
	adjustWindowRectEx  = user32.NewProc("AdjustWindowRectEx")
	getClientRect       = user32.NewProc("GetClientRect")
	setFocus            = user32.NewProc("SetFocus")
	setBkMode           = gdi32.NewProc("SetBkMode")
	setTextColor        = gdi32.NewProc("SetTextColor")
	getStockObject      = gdi32.NewProc("GetStockObject")
)

const (
	// 客户区尺寸（不含标题栏/边框），按钮按客户区右下角定位
	noticeClientW = 440
	noticeClientH = 180

	noticePad       = 24
	noticeIconSize  = 32
	noticeBtnW      = 96
	noticeBtnH      = 32
	noticeBtnBottom = 20

	idiInformation = 32516
	idiError       = 32515
	idiApplication = 32512

	wsExTopmost       = 0x00000008
	wsExDlgModalFrame = 0x00000001
	wsCaption         = 0x00C00000
	wsSysMenu         = 0x00080000
	wsVisible         = 0x10000000
	wsChild           = 0x40000000
	wsTabStop         = 0x00010000

	ssIcon       = 0x00000003
	ssLeft       = 0x00000000
	ssNoprefix   = 0x00000080
	bsDefPushBtn = 0x00000001

	stmSetIcon = 0x0170

	wmCreate         = 0x0001
	wmCommand        = 0x0111
	wmClose          = 0x0010
	wmCtlColorStatic = 0x0138

	idNoticeOkBtn = 1
	idNoticeIcon  = 2
	idNoticeMsg   = 3

	smCxScreen  = 0
	smCyScreen  = 1
	hwndTopmost = ^uintptr(0)
	bnClicked   = 0
	colorWindow = 5
	whiteBrush  = 0 // WHITE_BRUSH
	transparent = 1
	rgbBlack    = 0x00000000
)

type wndClassEx struct {
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

type point struct {
	x int32
	y int32
}

type rect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type win32Msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type noticeDialogState struct {
	message string
	success bool
}

var (
	noticeClassOnce sync.Once
	noticeClassName *uint16
	noticeDlgProc   uintptr
	noticeDlgCtx    noticeDialogState
)

func showNativeNotice(title, msg string, success bool) {
	ensureNoticeWindowClass()

	noticeDlgCtx = noticeDialogState{
		message: msg,
		success: success,
	}

	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}

	exStyle := uint32(wsExTopmost | wsExDlgModalFrame)
	style := uint32(wsCaption | wsSysMenu | wsVisible)

	rc := rect{
		left:   0,
		top:    0,
		right:  noticeClientW,
		bottom: noticeClientH,
	}
	adjustWindowRectEx.Call(
		uintptr(unsafe.Pointer(&rc)),
		uintptr(style),
		0,
		uintptr(exStyle),
	)
	outerW := int(rc.right - rc.left)
	outerH := int(rc.bottom - rc.top)

	hInstance, _, _ := getModuleHandleW.Call(0)
	screenW, _, _ := getSystemMetrics.Call(smCxScreen)
	screenH, _, _ := getSystemMetrics.Call(smCyScreen)
	posX := (int(screenW) - outerW) / 2
	posY := (int(screenH) - outerH) / 2
	if posX < 0 {
		posX = 0
	}
	if posY < 0 {
		posY = 0
	}

	hwnd, _, _ := createWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(noticeClassName)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(style),
		uintptr(posX),
		uintptr(posY),
		uintptr(outerW),
		uintptr(outerH),
		0,
		0,
		hInstance,
		0,
	)
	if hwnd == 0 {
		return
	}

	setForegroundWindow.Call(hwnd)
	setWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, 0x0001|0x0002) // SWP_NOMOVE|SWP_NOSIZE

	var winMsg win32Msg
	for {
		ret, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&winMsg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&winMsg)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&winMsg)))
	}
}

func ensureNoticeWindowClass() {
	noticeClassOnce.Do(func() {
		noticeClassName, _ = syscall.UTF16PtrFromString("YksToolNoticeDialog")
		noticeDlgProc = syscall.NewCallback(noticeDialogWndProc)

		hInstance, _, _ := getModuleHandleW.Call(0)
		appIcon, _, _ := loadIconW.Call(hInstance, uintptr(idiApplication))

		wndClass := wndClassEx{
			cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			lpfnWndProc:   noticeDlgProc,
			hInstance:     hInstance,
			hIcon:         appIcon,
			hIconSm:       appIcon,
			hbrBackground: colorWindow + 1,
			lpszClassName: noticeClassName,
		}
		registerClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))
	})
}

func noticeDialogWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCreate:
		createNoticeDialogControls(hwnd)
		return 0
	case wmCtlColorStatic:
		// 去掉 Static 默认灰底，与窗口白底一致
		setBkMode.Call(wParam, transparent)
		setTextColor.Call(wParam, rgbBlack)
		brush, _, _ := getStockObject.Call(whiteBrush)
		return brush
	case wmCommand:
		if wParam&0xffff == idNoticeOkBtn && (wParam>>16)&0xffff == bnClicked {
			destroyWindow.Call(hwnd)
			postQuitMessage.Call(0)
			return 0
		}
	case wmClose:
		destroyWindow.Call(hwnd)
		postQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := defWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

func createNoticeDialogControls(hwnd uintptr) {
	hInstance, _, _ := getModuleHandleW.Call(0)

	var client rect
	getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	clientW := int(client.right - client.left)
	clientH := int(client.bottom - client.top)
	if clientW <= 0 {
		clientW = noticeClientW
	}
	if clientH <= 0 {
		clientH = noticeClientH
	}

	iconID := uintptr(idiInformation)
	if !noticeDlgCtx.success {
		iconID = uintptr(idiError)
	}
	hIcon, _, _ := loadIconW.Call(0, iconID)

	contentTop := noticePad
	iconY := contentTop + 8

	staticClass, _ := syscall.UTF16PtrFromString("Static")
	iconHwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		0,
		uintptr(wsChild|wsVisible|ssIcon),
		uintptr(noticePad),
		uintptr(iconY),
		uintptr(noticeIconSize),
		uintptr(noticeIconSize),
		hwnd,
		idNoticeIcon,
		hInstance,
		0,
	)
	if iconHwnd != 0 && hIcon != 0 {
		sendMessageW.Call(iconHwnd, stmSetIcon, hIcon, 0)
	}

	msgLeft := noticePad + noticeIconSize + 16
	msgWidth := clientW - msgLeft - noticePad
	if msgWidth < 120 {
		msgWidth = 120
	}
	msgText, _ := syscall.UTF16PtrFromString(noticeDlgCtx.message)
	createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(msgText)),
		uintptr(wsChild|wsVisible|ssLeft|ssNoprefix),
		uintptr(msgLeft),
		uintptr(contentTop+10),
		uintptr(msgWidth),
		48,
		hwnd,
		idNoticeMsg,
		hInstance,
		0,
	)

	btnX := clientW - noticePad - noticeBtnW
	btnY := clientH - noticeBtnBottom - noticeBtnH
	if btnX < noticePad {
		btnX = noticePad
	}
	if btnY < contentTop+60 {
		btnY = contentTop + 60
	}

	okText, _ := syscall.UTF16PtrFromString(noticeBtnOK)
	btnClass, _ := syscall.UTF16PtrFromString("Button")
	btnHwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(okText)),
		uintptr(wsChild|wsVisible|wsTabStop|bsDefPushBtn),
		uintptr(btnX),
		uintptr(btnY),
		uintptr(noticeBtnW),
		uintptr(noticeBtnH),
		hwnd,
		idNoticeOkBtn,
		hInstance,
		0,
	)
	if btnHwnd != 0 {
		setFocus.Call(btnHwnd)
	}
}
