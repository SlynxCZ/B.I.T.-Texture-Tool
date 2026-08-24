//go:build windows

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// Embedded UI artwork supplied for v10. The logo swaps with the app theme,
// while the small flag images make the language controls independent of emoji fonts.
//
//go:embed logo_light.png
var logoLightPNG []byte

//go:embed logo_dark.png
var logoDarkPNG []byte

//go:embed flag_gb.png
var flagGBPNG []byte

//go:embed flag_ru.png
var flagRUPNG []byte

//go:embed flag_cz.png
var flagCZPNG []byte

//go:embed flag_br.png
var flagBRPNG []byte

//go:embed flag_fr.png
var flagFRPNG []byte

//go:embed flag_de.png
var flagDEPNG []byte

//go:embed flag_es.png
var flagESPNG []byte

//go:embed flag_pl.png
var flagPLPNG []byte

//go:embed flag_tr.png
var flagTRPNG []byte

const (
	appTitle = "B.I.T. Texture Tool V0.17.16"

	WM_CREATE          = 0x0001
	WM_SIZE            = 0x0005
	WM_GETMINMAXINFO   = 0x0024
	WM_DESTROY         = 0x0002
	WM_CLOSE           = 0x0010
	WM_PAINT           = 0x000F
	WM_ERASEBKGND      = 0x0014
	WM_DRAWITEM        = 0x002B
	WM_COMMAND         = 0x0111
	WM_HSCROLL         = 0x0114
	WM_CTLCOLORBTN     = 0x0135
	WM_CTLCOLORDLG     = 0x0136
	WM_CTLCOLORSTATIC  = 0x0138
	WM_CTLCOLOREDIT    = 0x0133
	WM_CTLCOLORLISTBOX = 0x0134
	WM_SETFONT         = 0x0030
	WM_SETICON         = 0x0080
	WM_USER            = 0x0400
	WM_APP_STATUS      = 0x8001
	WM_APP_DONE        = 0x8002
	WM_APP_DETECT      = 0x8003
	WM_APP_IMAGE       = 0x8004
	WM_APP_ADDONS      = 0x8005
	WM_APP_AUTO_DONE   = 0x8006
	WM_KEYDOWN         = 0x0100

	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_THICKFRAME       = 0x00040000
	WS_MINIMIZEBOX      = 0x00020000
	WS_MAXIMIZEBOX      = 0x00010000
	WS_OVERLAPPEDWINDOW = WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX
	WS_POPUP            = 0x80000000
	WS_CHILD            = 0x40000000
	WS_VISIBLE          = 0x10000000
	WS_TABSTOP          = 0x00010000
	WS_GROUP            = 0x00020000
	WS_VSCROLL          = 0x00200000
	WS_BORDER           = 0x00800000

	ES_AUTOHSCROLL     = 0x0080
	BS_PUSHBUTTON      = 0x00000000
	BS_AUTOCHECKBOX    = 0x00000003
	BS_MULTILINE       = 0x00002000
	BS_AUTORADIOBUTTON = 0x00000009
	BS_OWNERDRAW       = 0x0000000B
	SS_CENTER          = 0x00000001
	SS_OWNERDRAW       = 0x0000000D

	CBS_DROPDOWN     = 0x0002
	CBS_DROPDOWNLIST = 0x0003

	SW_HIDE     = 0
	SW_SHOW     = 5
	SW_MAXIMIZE = 3
	SW_RESTORE  = 9

	SIZE_RESTORED  = 0
	SIZE_MAXIMIZED = 2

	VK_F11          = 0x7A
	SM_CXSCREEN     = 0
	SM_CYSCREEN     = 1
	GWL_STYLE_INDEX = 0xFFFFFFF0 // -16 as DWORD for Get/SetWindowLongW

	CW_USEDEFAULT = 0x80000000

	COLOR_WINDOW     = 5
	DEFAULT_GUI_FONT = 17
	HOLLOW_BRUSH     = 5
	IDC_ARROW        = 32512

	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_ICONERROR       = 0x00000010
	MB_YESNO           = 0x00000004
	MB_YESNOCANCEL     = 0x00000003
	MB_ICONQUESTION    = 0x00000020
	IDYES              = 6
	IDNO               = 7
	IDCANCEL           = 2

	OFN_FILEMUSTEXIST = 0x00001000
	OFN_PATHMUSTEXIST = 0x00000800
	OFN_EXPLORER      = 0x00080000

	BIF_RETURNONLYFSDIRS = 0x00000001
	BIF_NEWDIALOGSTYLE   = 0x00000040

	CB_ADDSTRING    = 0x0143
	CB_RESETCONTENT = 0x014B
	CB_SETCURSEL    = 0x014E
	CB_GETCURSEL    = 0x0147
	CBN_SELCHANGE   = 1
	BN_CLICKED      = 0

	BM_GETCHECK = 0x00F0
	BM_SETCHECK = 0x00F1
	BST_CHECKED = 1

	TBM_GETPOS   = WM_USER
	TBM_SETPOS   = WM_USER + 5
	TBM_SETRANGE = WM_USER + 6

	SRCCOPY        = 0x00CC0020
	DIB_RGB_COLORS = 0
	PS_SOLID       = 0

	DT_CENTER     = 0x00000001
	DT_VCENTER    = 0x00000004
	DT_SINGLELINE = 0x00000020

	ODT_BUTTON   = 4
	ODT_STATIC   = 5
	ODS_SELECTED = 0x0001
	ODS_DISABLED = 0x0004

	EDGE_RAISED = 0x0005
	EDGE_SUNKEN = 0x000A
	BF_RECT     = 0x000F

	SWP_NOMOVE       = 0x0002
	SWP_NOSIZE       = 0x0001
	SWP_NOZORDER     = 0x0004
	SWP_FRAMECHANGED = 0x0020

	ID_BTN_PNG              = 1001
	ID_BTN_CROP             = 1002
	ID_EDIT_ROOT            = 1003
	ID_BTN_ROOT             = 1004
	ID_COMBO_ADDON          = 1005
	ID_EDIT_MAT             = 1006
	ID_COMBO_SHADER         = 1007
	ID_CHECK_COMPILE        = 1008
	ID_BTN_CREATE           = 1009
	ID_BTN_OUTPUT           = 1010
	ID_BTN_REFRESH          = 1011
	ID_COMBO_QUALITY        = 1012
	ID_BTN_CONVERT          = 1013
	ID_BTN_DETECT           = 1014
	ID_QUALITY_ORIGINAL     = 1020
	ID_QUALITY_HD           = 1021
	ID_QUALITY_HIGH         = 1022
	ID_QUALITY_MID          = 1023
	ID_QUALITY_LOW          = 1024
	ID_BTN_SETTINGS         = 1030
	ID_COMBO_GAME           = 1031
	ID_COMBO_LANGUAGE       = 1033 // legacy; v10 uses language flag buttons
	ID_BTN_THEME            = 1032
	ID_MATERIAL_OPAQUE      = 1040
	ID_MATERIAL_CUTOUT      = 1041
	ID_MATERIAL_TRANS       = 1042
	ID_ALPHA_TRACK          = 1043
	ID_BTN_JUNK_OPEN        = 1044
	ID_BTN_JUNK_CLEAR       = 1045
	ID_BTN_LOG              = 1046
	ID_BTN_AUTONOMOUS       = 1047
	ID_BTN_FULLSCREEN       = 1048
	ID_BTN_STOP             = 1049
	ID_AUTO_SLOW            = 1065
	ID_AUTO_NORMAL          = 1066
	ID_AUTO_FAST            = 1067
	ID_AUTO_EXTREME         = 1068
	ID_AUTO_CUSTOM          = 1069
	ID_CHECK_RETRY          = 1070
	ID_CHECK_LOCK           = 1071
	ID_COMBO_OVERWRITE      = 1072
	ID_COMBO_CUSTOM_WORKERS = 1073
	ID_LANG_EN              = 1050
	ID_LANG_RU              = 1051
	ID_LANG_CS              = 1052
	ID_LANG_BR              = 1053
	ID_LANG_FR              = 1054
	ID_LANG_DE              = 1055
	ID_LANG_ES              = 1056
	ID_LANG_PL              = 1057
	ID_LANG_TR              = 1058
	ID_LOGO                 = 1060

	ID_CROP_TRACK  = 2001
	ID_CROP_CENTER = 2002
	ID_CROP_USE    = 2003
	ID_CROP_CANCEL = 2004
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")

	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procEnableWindow         = user32.NewProc("EnableWindow")
	procMessageBoxW          = user32.NewProc("MessageBoxW")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procBeginPaint           = user32.NewProc("BeginPaint")
	procEndPaint             = user32.NewProc("EndPaint")
	procInvalidateRect       = user32.NewProc("InvalidateRect")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procSetFocus             = user32.NewProc("SetFocus")
	procSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	procPostMessageW         = user32.NewProc("PostMessageW")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procFillRect             = user32.NewProc("FillRect")
	procDrawTextW            = user32.NewProc("DrawTextW")
	procDrawEdge             = user32.NewProc("DrawEdge")
	procGetWindowLongW       = user32.NewProc("GetWindowLongW")
	procSetWindowLongW       = user32.NewProc("SetWindowLongW")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procStretchDIBits    = gdi32.NewProc("StretchDIBits")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procRectangle        = gdi32.NewProc("Rectangle")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSetBkColor       = gdi32.NewProc("SetBkColor")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procTextOutW         = gdi32.NewProc("TextOutW")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procRoundRect        = gdi32.NewProc("RoundRect")

	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")

	procSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	procShellExecuteW        = shell32.NewProc("ShellExecuteW")

	procCoTaskMemFree  = ole32.NewProc("CoTaskMemFree")
	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
	procCoUninitialize = ole32.NewProc("CoUninitialize")

	procInitCommonControls = comctl32.NewProc("InitCommonControls")
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MINMAXINFO struct {
	PtReserved, PtMaxSize, PtMaxPosition, PtMinTrackSize, PtMaxTrackSize POINT
}
type MSG struct {
	Hwnd     syscall.Handle
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}
type PAINTSTRUCT struct {
	Hdc         syscall.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}
type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}
type RGBQUAD struct{ Blue, Green, Red, Reserved byte }
type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]RGBQUAD
}
type OPENFILENAME struct {
	LStructSize       uint32
	HwndOwner         syscall.Handle
	HInstance         syscall.Handle
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        unsafe.Pointer
	DwReserved        uint32
	FlagsEx           uint32
}
type BROWSEINFO struct {
	HwndOwner      syscall.Handle
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   syscall.Handle
	HDC        syscall.Handle
	RcItem     RECT
	ItemData   uintptr
}

type CropState struct {
	hwnd       syscall.Handle
	track      syscall.Handle
	img        image.Image
	pixels     []byte
	srcPath    string
	w, h       int
	size       int
	offset     int
	horizontal bool
	done       bool
	accepted   bool
}

type imageAsset struct {
	img    image.Image
	pixels []byte
	w, h   int
}

type AppSettings struct {
	Language      string `json:"language"`
	Game          string `json:"game"`
	DarkMode      bool   `json:"dark_mode"`
	CS2Root       string `json:"cs2_root"`
	LastAddon     string `json:"last_addon"`
	LastImageDir  string `json:"last_image_dir"`
	MaterialMode  int    `json:"material_mode"`
	Quality       int    `json:"quality"`
	AutoMode      int    `json:"auto_mode"`
	RetryCompile  bool   `json:"retry_compile"`
	CompilerLock  bool   `json:"compiler_lock"`
	OverwriteMode int    `json:"overwrite_mode"`
	CustomWorkers int    `json:"custom_workers"`
}

type imageLoadResult struct {
	img          image.Image
	path         string
	originalPath string
	format       string
	w, h         int
	hasAlpha     bool
	err          error
}

type addonLoadResult struct {
	root  string
	game  string
	names []string
	err   error
}

var (
	hInstance           syscall.Handle
	hwndMain            syscall.Handle
	hFileLabel          syscall.Handle
	hDimLabel           syscall.Handle
	hCropBtn            syscall.Handle
	hRootEdit           syscall.Handle
	hAddonCombo         syscall.Handle
	hMatEdit            syscall.Handle
	hShaderCombo        syscall.Handle
	hQualityCombo       syscall.Handle // retained for compatibility; v6 uses radio buttons
	hCompileCheck       syscall.Handle
	hStatusLabel        syscall.Handle
	hOutputBtn          syscall.Handle
	hCreateBtn          syscall.Handle
	hDetectBtn          syscall.Handle
	hSettingsBtn        syscall.Handle
	hLanguageCombo      syscall.Handle // legacy handle; not used by v10
	hThemeBtn           syscall.Handle
	hLogo               syscall.Handle
	hAlphaTrack         syscall.Handle
	hAlphaLabel         syscall.Handle
	hMaterialRadios     [3]syscall.Handle
	hQualityRadios      [5]syscall.Handle
	hLogBtn             syscall.Handle
	hAutoBtn            syscall.Handle
	hStopBtn            syscall.Handle
	hFullscreenBtn      syscall.Handle
	hAutoModeRadios     [5]syscall.Handle
	hChooseBtn          syscall.Handle
	hJunkOpen           syscall.Handle
	hJunkClear          syscall.Handle
	hStepImage          syscall.Handle
	hStepCS2            syscall.Handle
	hBrowseBtn          syscall.Handle
	hAddonLabel         syscall.Handle
	hRefreshBtn         syscall.Handle
	hStepVmat           syscall.Handle
	hStepType           syscall.Handle
	hStepQuality        syscall.Handle
	hQualityHint        syscall.Handle
	hOutputLabel        syscall.Handle
	hAutoSpeedLabel     syscall.Handle
	hRetryCompile       syscall.Handle
	hCompilerLock       syscall.Handle
	hOverwriteLabel     syscall.Handle
	hOverwriteCombo     syscall.Handle
	hCustomWorkersLabel syscall.Handle
	hCustomWorkersCombo syscall.Handle
	hSettingsTitle      syscall.Handle // legacy; V0.17 uses the Settings button as the drawer title
	hAppTitle           syscall.Handle
	hAppSubtitle        syscall.Handle
	hLanguageLabel      syscall.Handle
	hGameLabel          syscall.Handle
	hGameCombo          syscall.Handle
	hThemeLabel         syscall.Handle
	hVersionLabel       syscall.Handle
	hLanguageButtons    [9]syscall.Handle
	settingsControls    []syscall.Handle

	currentImage          image.Image
	currentPath           string
	currentOriginalPath   string
	currentW, currentH    int
	lastOutputDir         string
	lastLogPath           string
	addonNames            []string
	selectedAddon         string
	crop                  *CropState
	mainWndProcCallback   uintptr
	cropWndProcCallback   uintptr
	selectedQuality       int  = 0 // 0 original, 1 4096, 2 2048, 3 1024, 4 512
	selectedMaterialMode  int  = 0 // 0 opaque, 1 cutout, 2 translucent
	selectedAutoMode      int  = 1 // 0 slow, 1 normal, 2 fast, 3 extreme, 4 custom
	selectedCustomWorkers int  = 8 // Custom autonomous worker count (1-32)
	retryCompile          bool = true
	compilerLockEnabled   bool = true
	selectedOverwriteMode int  = 0  // 0 ask, 1 skip existing, 2 replace existing
	alphaThreshold        int  = 50 // percent, 1-99
	sharedVMATCompileMu   sync.RWMutex
	createBusy            bool
	imageBusy             bool
	jobMu                 sync.Mutex
	jobStatus             string
	jobResult             createJobResult
	detectMu              sync.Mutex
	detectResult          string
	detectResultGame      string
	detectBusy            bool
	imageMu               sync.Mutex
	imageResult           imageLoadResult
	addonMu               sync.Mutex
	addonResult           addonLoadResult
	addonBusy             bool
	autoBusy              bool
	autoMu                sync.Mutex
	autoResult            createJobResult
	autoCancelMu          sync.Mutex
	autoCancelCh          chan struct{}
	fullscreen            bool
	fullscreenOldStyle    uint32
	fullscreenOldRect     RECT
	appSettings           AppSettings
	logoLightAsset        imageAsset
	logoDarkAsset         imageAsset
	flagAssets            map[int]imageAsset

	selectedGame = "cs2"

	currentLang   = "en"
	darkMode      = false
	settingsOpen  = true
	themeBrushBg  syscall.Handle
	themeBrushCtl syscall.Handle
	hFontTitle    syscall.Handle
	hFontSection  syscall.Handle
	hFontBody     syscall.Handle
	hFontSmall    syscall.Handle
	hFontButton   syscall.Handle
	hFontSettings syscall.Handle
	hFontFooter   syscall.Handle

	translatedControls []translatedControl
)

type translatedControl struct {
	h   syscall.Handle
	key string
}

type GameProfile struct {
	Key         string
	Name        string
	ShortName   string
	SteamFolder string
}

var gameProfiles = []GameProfile{
	{Key: "cs2", Name: "CS2", ShortName: "CS2", SteamFolder: "Counter-Strike Global Offensive"},
	{Key: "tf2", Name: "Team Fortress 2", ShortName: "TF2", SteamFolder: "Team Fortress 2"},
	{Key: "deadlock", Name: "Deadlock", ShortName: "Deadlock", SteamFolder: "Deadlock"},
	{Key: "hla", Name: "Half-Life: Alyx", ShortName: "HLA", SteamFolder: "Half-Life Alyx"},
}

func validGameKey(k string) bool {
	for _, g := range gameProfiles {
		if g.Key == k {
			return true
		}
	}
	return false
}

func currentGameName() string {
	for _, g := range gameProfiles {
		if g.Key == selectedGame {
			return g.Name
		}
	}
	return "CS2"
}

func gameProfileForKey(key string) GameProfile {
	for _, g := range gameProfiles {
		if g.Key == key {
			return g
		}
	}
	return gameProfiles[0]
}

func currentGameShortName() string {
	return gameProfileForKey(selectedGame).ShortName
}

func updateGameToolsUI() {
	short := currentGameShortName()
	if hStepCS2 != 0 {
		setText(hStepCS2, strings.ReplaceAll(tr("step_cs2"), "CS2", short))
	}
	if hDetectBtn != 0 {
		setText(hDetectBtn, strings.ReplaceAll(tr("detect_cs2"), "CS2", short))
	}
}

func gameAddonDirectory(root, gameKey string) string {
	switch gameKey {
	case "tf2":
		return filepath.Join(root, "tf", "custom")
	case "hla":
		return filepath.Join(root, "content", "hlvr_addons")
	case "deadlock":
		// Deadlock's internal Source 2 project name is commonly citadel. If this
		// folder is not present, the UI simply shows no addon folders.
		return filepath.Join(root, "content", "citadel_addons")
	default:
		return filepath.Join(root, "content", "csgo_addons")
	}
}

func validGameRoot(root, gameKey string) bool {
	if root == "" {
		return false
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return false
	}
	if gameKey == "cs2" {
		a, err1 := os.Stat(filepath.Join(root, "content", "csgo_addons"))
		b, err2 := os.Stat(filepath.Join(root, "game"))
		return err1 == nil && err2 == nil && a.IsDir() && b.IsDir()
	}
	return true
}

func gameComboIndex(key string) int {
	for i, g := range gameProfiles {
		if g.Key == key {
			return i
		}
	}
	return 0
}

func populateGameCombo() {
	if hGameCombo == 0 {
		return
	}
	procSendMessageW.Call(uintptr(hGameCombo), CB_RESETCONTENT, 0, 0)
	for _, g := range gameProfiles {
		sendMessageString(hGameCombo, CB_ADDSTRING, g.Name)
	}
	procSendMessageW.Call(uintptr(hGameCombo), CB_SETCURSEL, uintptr(gameComboIndex(selectedGame)), 0)
}

type languageDef struct {
	code string
	name string
}

var languageDefs = []languageDef{
	{"cs", "Čeština"},
	{"de", "Deutsch"},
	{"en", "English"},
	{"es", "Español"},
	{"fr", "Français"},
	{"pl", "Polski"},
	{"pt-BR", "Português (Brasil)"},
	{"ru", "Русский"},
	{"tr", "Türkçe"},
}

var translations = map[string]map[string]string{
	"en": {
		"settings": "Settings", "game": "Game", "language": "Language", "theme": "Appearance", "light": "Light mode", "dark": "Dark mode",
		"step_image": "1. Choose image", "choose_image": "Choose image...", "no_image": "No image selected",
		"image_waiting": "Choose a PNG, JPG/JPEG, GIF, BMP or TGA. Conversion and square checking are automatic.",
		"step_cs2":      "2. CS2 Workshop Tools / Addon", "detect_cs2": "Detect CS2", "browse": "Browse...", "addon": "Addon", "refresh": "Refresh",
		"step_vmat": "3. VMAT name (subfolders allowed)",
		"step_type": "4. Material type", "mat_opaque": "Normal / Opaque", "mat_cutout": "Cutout transparency", "mat_translucent": "See-through transparency", "alpha_threshold": "Alpha threshold: %d%%",
		"step_quality": "5. Choose output quality", "quality_hint": "Selection is instant. Resizing happens only after Create VMAT.",
		"quality_original": "Original", "quality_hd": "4096 Highly Detailed", "quality_high": "2048 High", "quality_mid": "1024 Mid", "quality_low": "512 Low",
		"compile_after": "Compile with CS2 Resource Compiler after creation", "output": "Status", "create_vmat": "6. Create VMAT", "open_output": "Open output folder", "open_log": "Open compile log",
		"open_junk": "Open Junk Folder", "clear_junk": "Clear Junk Folder", "fullscreen": "Maximize", "exit_fullscreen": "Restore window", "autonomous": "Autonomous Production", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Autonomous speed", "auto_slow": "Experimental Slow", "auto_normal": "Normal", "auto_fast": "Experimental Fast", "auto_extreme": "Experimental Extreme", "stop_auto": "Stop", "stopping": "Stopping Autonomous Production... current texture(s) will finish.",
		"center": "Center", "use_crop": "Use crop", "cancel": "Cancel",
		"ready":       "Ready. Nothing is scanned on startup. Choose an image or detect CS2 when you are ready.",
		"quality_set": "Quality set to %s. No resizing happens until Create VMAT.", "material_set": "Material mode: %s",
		"detecting": "Detecting CS2 in the background...", "detected": "CS2 detected. Addons are loading in the background.", "not_found": "CS2 was not found automatically. Use Browse and choose the Counter-Strike Global Offensive folder.",
		"invalid_root": "Invalid CS2 root. Choose the Counter-Strike Global Offensive folder first.", "no_addons": "No addons found under content\\csgo_addons.", "found_addons": "Found %d addon(s). Selected: %s",
		"loading_image": "Loading and preparing image in the background...", "image_ready": "Image ready: %d x %d", "alpha_found": "Transparency detected in this image.", "alpha_none": "No transparency detected.",
		"junk_cleared": "Temporary image folder cleared.", "junk_empty": "Junk folder is already empty.", "clear_confirm": "Delete all temporary images created by B.I.T. Texture Tool?",
	},
	"ru": {
		"settings": "Настройки", "game": "Игра", "language": "Язык", "theme": "Оформление", "light": "Светлая тема", "dark": "Тёмная тема",
		"step_image": "1. Выберите изображение", "choose_image": "Выбрать изображение...", "no_image": "Изображение не выбрано",
		"image_waiting": "Выберите PNG, JPG/JPEG, GIF, BMP или TGA. Конвертация и проверка квадрата выполняются автоматически.",
		"step_cs2":      "2. CS2 Workshop Tools / Аддон", "detect_cs2": "Найти CS2", "browse": "Обзор...", "addon": "Аддон", "refresh": "Обновить",
		"step_vmat": "3. Имя VMAT (можно подпапки)",
		"step_type": "4. Тип материала", "mat_opaque": "Обычный / Непрозрачный", "mat_cutout": "Прозрачность с вырезом", "mat_translucent": "Полупрозрачный", "alpha_threshold": "Порог альфа: %d%%",
		"step_quality": "5. Качество вывода", "quality_hint": "Выбор мгновенный. Размер изменяется только после создания VMAT.",
		"quality_original": "Оригинал", "quality_hd": "4096 Максимальная детализация", "quality_high": "2048 Высокое", "quality_mid": "1024 Среднее", "quality_low": "512 Низкое",
		"compile_after": "Компилировать через CS2 Resource Compiler", "output": "Статус", "create_vmat": "6. Создать VMAT", "open_output": "Открыть папку", "open_log": "Открыть журнал компиляции",
		"open_junk": "Открыть Junk-папку", "clear_junk": "Очистить Junk-папку", "fullscreen": "Развернуть", "exit_fullscreen": "Восстановить окно", "autonomous": "Автономное производство", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Скорость автономного режима", "auto_slow": "Экспериментально медленно", "auto_normal": "Обычно", "auto_fast": "Экспериментально быстро", "auto_extreme": "Экспериментальный экстрим", "stop_auto": "Стоп", "stopping": "Остановка автономного режима... текущие текстуры будут завершены.",
		"center": "По центру", "use_crop": "Применить", "cancel": "Отмена",
		"ready":       "Готово. При запуске ничего не сканируется. Выберите изображение или найдите CS2.",
		"quality_set": "Качество: %s. Размер изменится только при создании VMAT.", "material_set": "Режим материала: %s",
		"detecting": "Поиск CS2 в фоне...", "detected": "CS2 найден. Аддоны загружаются в фоне.", "not_found": "CS2 не найден автоматически. Нажмите Обзор и выберите папку Counter-Strike Global Offensive.",
		"invalid_root": "Неверная папка CS2.", "no_addons": "Аддоны в content\\csgo_addons не найдены.", "found_addons": "Найдено аддонов: %d. Выбран: %s",
		"loading_image": "Изображение загружается и подготавливается в фоне...", "image_ready": "Изображение готово: %d x %d", "alpha_found": "В изображении обнаружена прозрачность.", "alpha_none": "Прозрачность не обнаружена.",
		"junk_cleared": "Временная папка очищена.", "junk_empty": "Junk-папка уже пуста.", "clear_confirm": "Удалить все временные изображения, созданные B.I.T. Texture Tool?",
	},
	"cs": {
		"settings": "Nastavení", "game": "Hra", "language": "Jazyk", "theme": "Vzhled", "light": "Světlý režim", "dark": "Tmavý režim",
		"step_image": "1. Vyber obrázek", "choose_image": "Vybrat obrázek...", "no_image": "Není vybrán obrázek",
		"image_waiting": "Vyber PNG, JPG/JPEG, GIF, BMP nebo TGA. Převod a kontrola čtverce proběhnou automaticky.",
		"step_cs2":      "2. CS2 Workshop Tools / Addon", "detect_cs2": "Najít CS2", "browse": "Procházet...", "addon": "Addon", "refresh": "Obnovit",
		"step_vmat": "3. Název VMAT (lze podsložky)",
		"step_type": "4. Typ materiálu", "mat_opaque": "Normální / Neprůhledný", "mat_cutout": "Výřezová průhlednost", "mat_translucent": "Průhledný / průsvitný", "alpha_threshold": "Práh alfa: %d%%",
		"step_quality": "5. Vyber kvalitu výstupu", "quality_hint": "Výběr je okamžitý. Změna velikosti proběhne až při vytvoření VMAT.",
		"quality_original": "Původní", "quality_hd": "4096 Velmi detailní", "quality_high": "2048 Vysoká", "quality_mid": "1024 Střední", "quality_low": "512 Nízká",
		"compile_after": "Po vytvoření zkompilovat přes CS2 Resource Compiler", "output": "Stav", "create_vmat": "6. Vytvořit VMAT", "open_output": "Otevřít výstupní složku", "open_log": "Otevřít log kompilace",
		"open_junk": "Otevřít Junk složku", "clear_junk": "Vyčistit Junk složku", "fullscreen": "Maximalizovat", "exit_fullscreen": "Obnovit okno", "autonomous": "Autonomní výroba", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Rychlost autonomního režimu", "auto_slow": "Experimentálně pomalé", "auto_normal": "Normální", "auto_fast": "Experimentálně rychlé", "auto_extreme": "Experimentální extrém", "stop_auto": "Zastavit", "stopping": "Zastavuji autonomní výrobu... aktuální textury se dokončí.",
		"center": "Vycentrovat", "use_crop": "Použít ořez", "cancel": "Zrušit",
		"ready":       "Připraveno. Při spuštění se nic neskenuje. Vyber obrázek nebo najdi CS2.",
		"quality_set": "Kvalita nastavena na %s. Velikost se změní až při vytvoření VMAT.", "material_set": "Režim materiálu: %s",
		"detecting": "Hledám CS2 na pozadí...", "detected": "CS2 nalezeno. Addony se načítají na pozadí.", "not_found": "CS2 se nepodařilo najít automaticky. Použij Procházet a vyber složku Counter-Strike Global Offensive.",
		"invalid_root": "Neplatná složka CS2.", "no_addons": "V content\\csgo_addons nebyly nalezeny žádné addony.", "found_addons": "Nalezeno addonů: %d. Vybráno: %s",
		"loading_image": "Obrázek se načítá a připravuje na pozadí...", "image_ready": "Obrázek připraven: %d x %d", "alpha_found": "V obrázku byla zjištěna průhlednost.", "alpha_none": "Průhlednost nebyla zjištěna.",
		"junk_cleared": "Dočasná složka byla vyčištěna.", "junk_empty": "Junk složka je už prázdná.", "clear_confirm": "Smazat všechny dočasné obrázky vytvořené B.I.T. Texture Tool?",
	},
	"pt-BR": {
		"settings": "Configurações", "game": "Jogo", "language": "Idioma", "theme": "Aparência", "light": "Modo claro", "dark": "Modo escuro",
		"step_image": "1. Escolha a imagem", "choose_image": "Escolher imagem...", "no_image": "Nenhuma imagem selecionada",
		"step_status": "2. Status da imagem", "image_waiting": "Escolha PNG, JPG/JPEG, GIF, BMP ou TGA. Conversão e verificação quadrada são automáticas.",
		"step_cs2": "2. CS2 Workshop Tools / Addon", "detect_cs2": "Detectar CS2", "browse": "Procurar...", "addon": "Addon", "refresh": "Atualizar",
		"step_vmat": "3. Nome do VMAT (aceita subpastas)",
		"step_type": "4. Tipo de material", "mat_opaque": "Normal / Opaco", "mat_cutout": "Transparência recortada", "mat_translucent": "Transparente / semitransparente", "alpha_threshold": "Limite alfa: %d%%",
		"step_quality": "5. Qualidade de saída", "quality_hint": "A seleção é instantânea. O redimensionamento só acontece ao criar o VMAT.",
		"quality_original": "Original", "quality_hd": "4096 Altamente detalhado", "quality_high": "2048 Alta", "quality_mid": "1024 Média", "quality_low": "512 Baixa",
		"compile_after": "Compilar com CS2 Resource Compiler", "output": "Status", "create_vmat": "6. Criar VMAT", "open_output": "Abrir pasta de saída", "open_log": "Abrir log de compilação",
		"open_junk": "Abrir pasta Junk", "clear_junk": "Limpar pasta Junk", "fullscreen": "Maximizar", "exit_fullscreen": "Restaurar janela", "autonomous": "Produção autônoma", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Velocidade autônoma", "auto_slow": "Experimental lento", "auto_normal": "Normal", "auto_fast": "Experimental rápido", "auto_extreme": "Experimental extremo", "stop_auto": "Parar", "stopping": "Parando a produção autônoma... as texturas atuais serão concluídas.",
		"center": "Centralizar", "use_crop": "Usar corte", "cancel": "Cancelar",
		"ready":       "Pronto. Nada é verificado ao iniciar. Escolha uma imagem ou detecte o CS2.",
		"quality_set": "Qualidade definida para %s. O tamanho só muda ao criar o VMAT.", "material_set": "Modo de material: %s",
		"detecting": "Detectando CS2 em segundo plano...", "detected": "CS2 detectado. Os addons estão sendo carregados em segundo plano.", "not_found": "CS2 não foi encontrado automaticamente. Use Procurar e selecione a pasta Counter-Strike Global Offensive.",
		"invalid_root": "Pasta raiz do CS2 inválida.", "no_addons": "Nenhum addon encontrado em content\\csgo_addons.", "found_addons": "%d addon(s) encontrado(s). Selecionado: %s",
		"loading_image": "Carregando e preparando a imagem em segundo plano...", "image_ready": "Imagem pronta: %d x %d", "alpha_found": "Transparência detectada nesta imagem.", "alpha_none": "Nenhuma transparência detectada.",
		"junk_cleared": "Pasta temporária limpa.", "junk_empty": "A pasta Junk já está vazia.", "clear_confirm": "Excluir todas as imagens temporárias criadas pelo B.I.T. Texture Tool?",
	},
	"fr": {
		"settings": "Paramètres", "game": "Jeu", "language": "Langue", "theme": "Apparence", "light": "Mode clair", "dark": "Mode sombre",
		"step_image": "1. Choisir l’image", "choose_image": "Choisir une image...", "no_image": "Aucune image sélectionnée",
		"step_status": "2. État de l’image", "image_waiting": "Choisissez PNG, JPG/JPEG, GIF, BMP ou TGA. Conversion et vérification carrée sont automatiques.",
		"step_cs2": "2. CS2 Workshop Tools / Addon", "detect_cs2": "Détecter CS2", "browse": "Parcourir...", "addon": "Addon", "refresh": "Actualiser",
		"step_vmat": "3. Nom VMAT (sous-dossiers autorisés)",
		"step_type": "4. Type de matériau", "mat_opaque": "Normal / Opaque", "mat_cutout": "Transparence découpée", "mat_translucent": "Transparent / translucide", "alpha_threshold": "Seuil alpha : %d%%",
		"step_quality": "5. Qualité de sortie", "quality_hint": "La sélection est instantanée. Le redimensionnement se fait seulement lors de la création du VMAT.",
		"quality_original": "Original", "quality_hd": "4096 Très détaillé", "quality_high": "2048 Haute", "quality_mid": "1024 Moyenne", "quality_low": "512 Basse",
		"compile_after": "Compiler avec CS2 Resource Compiler", "output": "État", "create_vmat": "6. Créer VMAT", "open_output": "Ouvrir le dossier de sortie", "open_log": "Ouvrir le journal de compilation",
		"open_junk": "Ouvrir le dossier Junk", "clear_junk": "Vider le dossier Junk", "fullscreen": "Maximiser", "exit_fullscreen": "Restaurer la fenêtre", "autonomous": "Production autonome", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Vitesse autonome", "auto_slow": "Expérimental lent", "auto_normal": "Normal", "auto_fast": "Expérimental rapide", "auto_extreme": "Expérimental extrême", "stop_auto": "Arrêter", "stopping": "Arrêt de la production autonome... les textures en cours seront terminées.",
		"center": "Centrer", "use_crop": "Utiliser le recadrage", "cancel": "Annuler",
		"ready":       "Prêt. Rien n’est analysé au démarrage. Choisissez une image ou détectez CS2.",
		"quality_set": "Qualité réglée sur %s. Le redimensionnement se fera seulement lors de la création du VMAT.", "material_set": "Mode du matériau : %s",
		"detecting": "Détection de CS2 en arrière-plan...", "detected": "CS2 détecté. Les addons se chargent en arrière-plan.", "not_found": "CS2 n’a pas été trouvé automatiquement. Utilisez Parcourir et choisissez le dossier Counter-Strike Global Offensive.",
		"invalid_root": "Dossier racine CS2 invalide.", "no_addons": "Aucun addon trouvé dans content\\csgo_addons.", "found_addons": "%d addon(s) trouvé(s). Sélectionné : %s",
		"loading_image": "Chargement et préparation de l’image en arrière-plan...", "image_ready": "Image prête : %d x %d", "alpha_found": "Transparence détectée dans cette image.", "alpha_none": "Aucune transparence détectée.",
		"junk_cleared": "Dossier temporaire vidé.", "junk_empty": "Le dossier Junk est déjà vide.", "clear_confirm": "Supprimer toutes les images temporaires créées par B.I.T. Texture Tool ?",
	},
	"de": {
		"settings": "Einstellungen", "game": "Spiel", "language": "Sprache", "theme": "Darstellung", "light": "Heller Modus", "dark": "Dunkler Modus",
		"step_image": "1. Bild auswählen", "choose_image": "Bild auswählen...", "no_image": "Kein Bild ausgewählt",
		"image_waiting": "PNG, JPG/JPEG, GIF, BMP oder TGA auswählen. Konvertierung und Quadratprüfung erfolgen automatisch.",
		"step_cs2":      "2. CS2 Workshop Tools / Addon", "detect_cs2": "CS2 erkennen", "browse": "Durchsuchen...", "addon": "Addon", "refresh": "Aktualisieren",
		"step_vmat": "3. VMAT-Name (Unterordner erlaubt)",
		"step_type": "4. Materialtyp", "mat_opaque": "Normal / Undurchsichtig", "mat_cutout": "Ausschnitt-Transparenz", "mat_translucent": "Durchsichtige Transparenz", "alpha_threshold": "Alpha-Schwelle: %d%%",
		"step_quality": "5. Ausgabequalität wählen", "quality_hint": "Die Auswahl ist sofort. Die Größenänderung erfolgt erst beim Erstellen des VMAT.",
		"quality_original": "Original", "quality_hd": "4096 Sehr detailliert", "quality_high": "2048 Hoch", "quality_mid": "1024 Mittel", "quality_low": "512 Niedrig",
		"compile_after": "Nach Erstellung mit CS2 Resource Compiler kompilieren", "output": "Status", "create_vmat": "6. VMAT erstellen", "open_output": "Ausgabeordner öffnen", "open_log": "Kompilierungsprotokoll öffnen",
		"open_junk": "Junk-Ordner öffnen", "clear_junk": "Junk-Ordner leeren", "fullscreen": "Maximieren", "exit_fullscreen": "Fenster wiederherstellen", "autonomous": "Autonome Produktion", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Autonome Geschwindigkeit", "auto_slow": "Experimentell langsam", "auto_normal": "Normal", "auto_fast": "Experimentell schnell", "auto_extreme": "Experimentell extrem", "stop_auto": "Stopp", "stopping": "Autonome Produktion wird gestoppt... aktuelle Texturen werden fertiggestellt.",
		"center": "Zentrieren", "use_crop": "Zuschnitt verwenden", "cancel": "Abbrechen",
		"ready":       "Bereit. Beim Start wird nichts gescannt. Wähle ein Bild oder erkenne CS2.",
		"quality_set": "Qualität auf %s gesetzt. Größenänderung erst beim Erstellen des VMAT.", "material_set": "Materialmodus: %s",
		"detecting": "CS2 wird im Hintergrund gesucht...", "detected": "CS2 erkannt. Addons werden im Hintergrund geladen.", "not_found": "CS2 wurde nicht automatisch gefunden. Durchsuchen verwenden und den Ordner Counter-Strike Global Offensive wählen.",
		"invalid_root": "Ungültiger CS2-Stammordner.", "no_addons": "Keine Addons unter content\\csgo_addons gefunden.", "found_addons": "%d Addon(s) gefunden. Ausgewählt: %s",
		"loading_image": "Bild wird im Hintergrund geladen und vorbereitet...", "image_ready": "Bild bereit: %d x %d", "alpha_found": "Transparenz im Bild erkannt.", "alpha_none": "Keine Transparenz erkannt.",
		"junk_cleared": "Temporärer Bildordner geleert.", "junk_empty": "Der Junk-Ordner ist bereits leer.", "clear_confirm": "Alle von B.I.T. Texture Tool erstellten temporären Bilder löschen?",
	},
	"es": {
		"settings": "Configuración", "game": "Juego", "language": "Idioma", "theme": "Apariencia", "light": "Modo claro", "dark": "Modo oscuro",
		"step_image": "1. Elegir imagen", "choose_image": "Elegir imagen...", "no_image": "Ninguna imagen seleccionada",
		"image_waiting": "Elige PNG, JPG/JPEG, GIF, BMP o TGA. La conversión y comprobación cuadrada son automáticas.",
		"step_cs2":      "2. CS2 Workshop Tools / Addon", "detect_cs2": "Detectar CS2", "browse": "Examinar...", "addon": "Addon", "refresh": "Actualizar",
		"step_vmat": "3. Nombre VMAT (se permiten subcarpetas)",
		"step_type": "4. Tipo de material", "mat_opaque": "Normal / Opaco", "mat_cutout": "Transparencia recortada", "mat_translucent": "Transparencia translúcida", "alpha_threshold": "Umbral alfa: %d%%",
		"step_quality": "5. Elegir calidad de salida", "quality_hint": "La selección es instantánea. El cambio de tamaño solo ocurre al crear el VMAT.",
		"quality_original": "Original", "quality_hd": "4096 Muy detallada", "quality_high": "2048 Alta", "quality_mid": "1024 Media", "quality_low": "512 Baja",
		"compile_after": "Compilar con CS2 Resource Compiler después de crear", "output": "Estado", "create_vmat": "6. Crear VMAT", "open_output": "Abrir carpeta de salida", "open_log": "Abrir registro de compilación",
		"open_junk": "Abrir carpeta Junk", "clear_junk": "Vaciar carpeta Junk", "fullscreen": "Maximizar", "exit_fullscreen": "Restaurar ventana", "autonomous": "Producción autónoma", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Velocidad autónoma", "auto_slow": "Experimental lento", "auto_normal": "Normal", "auto_fast": "Experimental rápido", "auto_extreme": "Experimental extremo", "stop_auto": "Detener", "stopping": "Deteniendo la producción autónoma... las texturas actuales terminarán.",
		"center": "Centrar", "use_crop": "Usar recorte", "cancel": "Cancelar",
		"ready":       "Listo. No se analiza nada al iniciar. Elige una imagen o detecta CS2.",
		"quality_set": "Calidad establecida en %s. El tamaño no cambia hasta crear el VMAT.", "material_set": "Modo de material: %s",
		"detecting": "Detectando CS2 en segundo plano...", "detected": "CS2 detectado. Los addons se cargan en segundo plano.", "not_found": "CS2 no se encontró automáticamente. Usa Examinar y elige la carpeta Counter-Strike Global Offensive.",
		"invalid_root": "Carpeta raíz de CS2 no válida.", "no_addons": "No se encontraron addons en content\\csgo_addons.", "found_addons": "Se encontraron %d addon(s). Seleccionado: %s",
		"loading_image": "Cargando y preparando la imagen en segundo plano...", "image_ready": "Imagen lista: %d x %d", "alpha_found": "Se detectó transparencia en esta imagen.", "alpha_none": "No se detectó transparencia.",
		"junk_cleared": "Carpeta temporal vaciada.", "junk_empty": "La carpeta Junk ya está vacía.", "clear_confirm": "¿Eliminar todas las imágenes temporales creadas por B.I.T. Texture Tool?",
	},

	"pl": {
		"settings": "Ustawienia", "game": "Gra", "language": "Język", "theme": "Wygląd", "light": "Tryb jasny", "dark": "Tryb ciemny",
		"step_image": "1. Wybierz obraz", "choose_image": "Wybierz obraz...", "no_image": "Nie wybrano obrazu",
		"image_waiting": "Wybierz PNG, JPG/JPEG, GIF, BMP lub TGA. Konwersja i sprawdzanie kwadratu są automatyczne.",
		"step_cs2":      "2. CS2 Workshop Tools / Dodatek", "detect_cs2": "Wykryj CS2", "browse": "Przeglądaj...", "addon": "Dodatek", "refresh": "Odśwież",
		"step_vmat": "3. Nazwa VMAT (podfoldery dozwolone)",
		"step_type": "4. Typ materiału", "mat_opaque": "Normalny / Nieprzezroczysty", "mat_cutout": "Przezroczystość wycinana", "mat_translucent": "Przezroczysty / półprzezroczysty", "alpha_threshold": "Próg alfa: %d%%",
		"step_quality": "5. Jakość wyjściowa", "quality_hint": "Wybór jest natychmiastowy. Zmiana rozmiaru następuje dopiero po utworzeniu VMAT.",
		"quality_original": "Oryginalna", "quality_hd": "4096 Bardzo szczegółowa", "quality_high": "2048 Wysoka", "quality_mid": "1024 Średnia", "quality_low": "512 Niska",
		"compile_after": "Kompiluj przez CS2 Resource Compiler", "output": "Stan", "create_vmat": "6. Utwórz VMAT", "open_output": "Otwórz folder wyjściowy", "open_log": "Otwórz log kompilacji",
		"open_junk": "Otwórz folder Junk", "clear_junk": "Wyczyść folder Junk", "fullscreen": "Maksymalizuj", "exit_fullscreen": "Przywróć okno", "autonomous": "Produkcja autonomiczna", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Prędkość autonomiczna", "auto_slow": "Eksperymentalnie wolno", "auto_normal": "Normalnie", "auto_fast": "Eksperymentalnie szybko", "auto_extreme": "Eksperymentalnie ekstremalnie", "stop_auto": "Stop", "stopping": "Zatrzymywanie produkcji autonomicznej... bieżące tekstury zostaną dokończone.",
		"center": "Wyśrodkuj", "use_crop": "Użyj kadru", "cancel": "Anuluj",
		"ready":       "Gotowe. Przy starcie nic nie jest skanowane. Wybierz obraz lub wykryj CS2.",
		"quality_set": "Ustawiono jakość %s. Rozmiar zmieni się dopiero podczas tworzenia VMAT.", "material_set": "Tryb materiału: %s",
		"detecting": "Wykrywanie CS2 w tle...", "detected": "Wykryto CS2. Dodatki są ładowane w tle.", "not_found": "Nie znaleziono CS2 automatycznie. Użyj Przeglądaj i wybierz folder Counter-Strike Global Offensive.",
		"invalid_root": "Nieprawidłowy folder główny CS2.", "no_addons": "Nie znaleziono dodatków w content\\csgo_addons.", "found_addons": "Znaleziono dodatków: %d. Wybrano: %s",
		"loading_image": "Ładowanie i przygotowywanie obrazu w tle...", "image_ready": "Obraz gotowy: %d x %d", "alpha_found": "Wykryto przezroczystość w obrazie.", "alpha_none": "Nie wykryto przezroczystości.",
		"junk_cleared": "Folder tymczasowy został wyczyszczony.", "junk_empty": "Folder Junk jest już pusty.", "clear_confirm": "Usunąć wszystkie tymczasowe obrazy utworzone przez B.I.T. Texture Tool?",
	},
	"tr": {
		"settings": "Ayarlar", "game": "Oyun", "language": "Dil", "theme": "Görünüm", "light": "Açık mod", "dark": "Koyu mod",
		"step_image": "1. Görsel seç", "choose_image": "Görsel seç...", "no_image": "Görsel seçilmedi",
		"image_waiting": "PNG, JPG/JPEG, GIF, BMP veya TGA seçin. Dönüştürme ve kare kontrolü otomatik yapılır.",
		"step_cs2":      "2. CS2 Workshop Tools / Eklenti", "detect_cs2": "CS2'yi bul", "browse": "Gözat...", "addon": "Eklenti", "refresh": "Yenile",
		"step_vmat": "3. VMAT adı (alt klasörlere izin verilir)",
		"step_type": "4. Malzeme türü", "mat_opaque": "Normal / Opak", "mat_cutout": "Kesik saydamlık", "mat_translucent": "Saydam / yarı saydam", "alpha_threshold": "Alfa eşiği: %d%%",
		"step_quality": "5. Çıktı kalitesi", "quality_hint": "Seçim anlıktır. Yeniden boyutlandırma yalnızca VMAT oluşturulurken yapılır.",
		"quality_original": "Orijinal", "quality_hd": "4096 Çok ayrıntılı", "quality_high": "2048 Yüksek", "quality_mid": "1024 Orta", "quality_low": "512 Düşük",
		"compile_after": "Oluşturduktan sonra CS2 Resource Compiler ile derle", "output": "Durum", "create_vmat": "6. VMAT oluştur", "open_output": "Çıktı klasörünü aç", "open_log": "Derleme günlüğünü aç",
		"open_junk": "Junk klasörünü aç", "clear_junk": "Junk klasörünü temizle", "fullscreen": "Büyüt", "exit_fullscreen": "Pencereyi geri yükle", "autonomous": "Otonom Üretim", "version": "V0.17.16 • Made by Tabo using ChatGPT",
		"auto_speed": "Otonom hız", "auto_slow": "Deneysel Yavaş", "auto_normal": "Normal", "auto_fast": "Deneysel Hızlı", "auto_extreme": "Deneysel Ekstrem", "stop_auto": "Durdur", "stopping": "Otonom Üretim durduruluyor... mevcut dokular tamamlanacak.",
		"center": "Ortala", "use_crop": "Kırpmayı kullan", "cancel": "İptal",
		"ready":       "Hazır. Başlangıçta hiçbir şey taranmaz. Bir görsel seçin veya CS2'yi algılayın.",
		"quality_set": "Kalite %s olarak ayarlandı. Boyut yalnızca VMAT oluşturulurken değişir.", "material_set": "Malzeme modu: %s",
		"detecting": "CS2 arka planda aranıyor...", "detected": "CS2 bulundu. Eklentiler arka planda yükleniyor.", "not_found": "CS2 otomatik bulunamadı. Gözat'ı kullanıp Counter-Strike Global Offensive klasörünü seçin.",
		"invalid_root": "Geçersiz CS2 kök klasörü.", "no_addons": "content\\csgo_addons altında eklenti bulunamadı.", "found_addons": "%d eklenti bulundu. Seçili: %s",
		"loading_image": "Görsel arka planda yükleniyor ve hazırlanıyor...", "image_ready": "Görsel hazır: %d x %d", "alpha_found": "Bu görselde saydamlık algılandı.", "alpha_none": "Saydamlık algılanmadı.",
		"junk_cleared": "Geçici klasör temizlendi.", "junk_empty": "Junk klasörü zaten boş.", "clear_confirm": "B.I.T. Texture Tool tarafından oluşturulan tüm geçici görseller silinsin mi?",
	}}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func u16multi(s string) []uint16 {
	r := []rune(s)
	out := utf16.Encode(r)
	out = append(out, 0)
	return out
}

func loword(v uintptr) uint16 { return uint16(v & 0xffff) }
func hiword(v uintptr) uint16 { return uint16((v >> 16) & 0xffff) }

func makeLPARAM(lo, hi int32) uintptr {
	return uintptr(uint32(uint16(lo)) | (uint32(uint16(hi)) << 16))
}

func setText(hwnd syscall.Handle, s string) {
	if hwnd == 0 {
		return
	}
	p := u16(s)
	procSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

func getText(hwnd syscall.Handle) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	buf := make([]uint16, n+2)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func msgBox(owner syscall.Handle, text, title string, flags uintptr) uintptr {
	pt := u16(text)
	pc := u16(title)
	r, _, _ := procMessageBoxW.Call(uintptr(owner), uintptr(unsafe.Pointer(pt)), uintptr(unsafe.Pointer(pc)), flags)
	runtime.KeepAlive(pt)
	runtime.KeepAlive(pc)
	return r
}

func createUIFontFace(height int32, weight int32, family string) syscall.Handle {
	face := u16(family)
	f, _, _ := procCreateFontW.Call(
		uintptr(height), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)),
	)
	runtime.KeepAlive(face)
	return syscall.Handle(f)
}

func createAppFonts() {
	// Use the standard Segoe UI families explicitly. They are available across
	// supported Windows versions and avoid falling back to the dated Win32 GUI font.
	// The body scale is intentionally a little larger than older B.I.T. builds so
	// radios, checks, combo boxes and helper text visibly share one modern style.
	hFontTitle = createUIFontFace(-28, 700, "Segoe UI")
	hFontSection = createUIFontFace(-17, 600, "Segoe UI Semibold")
	hFontBody = createUIFontFace(-16, 400, "Segoe UI")
	hFontSmall = createUIFontFace(-14, 400, "Segoe UI")
	hFontButton = createUIFontFace(-15, 600, "Segoe UI Semibold")
	hFontSettings = createUIFontFace(-14, 400, "Segoe UI")
	hFontFooter = createUIFontFace(-12, 400, "Segoe UI")
}

func deleteAppFonts() {
	for _, h := range []syscall.Handle{hFontTitle, hFontSection, hFontBody, hFontSmall, hFontButton, hFontSettings, hFontFooter} {
		if h != 0 {
			procDeleteObject.Call(uintptr(h))
		}
	}
}

func setControlFont(hwnd, font syscall.Handle) {
	if hwnd == 0 {
		return
	}
	if font == 0 {
		f, _, _ := procGetStockObject.Call(DEFAULT_GUI_FONT)
		font = syscall.Handle(f)
	}
	procSendMessageW.Call(uintptr(hwnd), WM_SETFONT, uintptr(font), 1)
}

func setFont(hwnd syscall.Handle) {
	// Every ordinary control gets the app font immediately; avoid Win32's
	// dated DEFAULT_GUI_FONT wherever we control the typography.
	setControlFont(hwnd, hFontBody)
}

func createControl(class, text string, style uint32, x, y, w, h int32, parent syscall.Handle, id int) syscall.Handle {
	pc := u16(class)
	pt := u16(text)
	hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(pc)), uintptr(unsafe.Pointer(pt)), uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(parent), uintptr(id), uintptr(hInstance), 0)
	runtime.KeepAlive(pc)
	runtime.KeepAlive(pt)
	hnd := syscall.Handle(hwnd)
	if hnd != 0 {
		setFont(hnd)
	}
	return hnd
}

func tr(key string) string {
	if m, ok := translations[currentLang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if m, ok := translations["en"]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	return key
}

func registerTranslated(h syscall.Handle, key string) syscall.Handle {
	if h != 0 {
		translatedControls = append(translatedControls, translatedControl{h: h, key: key})
	}
	return h
}

func createTranslatedControl(class, key string, style uint32, x, y, w, h int32, parent syscall.Handle, id int) syscall.Handle {
	return registerTranslated(createControl(class, tr(key), style, x, y, w, h, parent, id), key)
}

func create3DButton(parent syscall.Handle, key string, x, y, w, h int32, id int) syscall.Handle {
	return registerTranslated(createControl("BUTTON", tr(key), WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, x, y, w, h, parent, id), key)
}

func create3DButtonText(parent syscall.Handle, text string, x, y, w, h int32, id int) syscall.Handle {
	return createControl("BUTTON", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, x, y, w, h, parent, id)
}

func themeColors() (bg, ctl, text, edgeLight, edgeDark uintptr) {
	if darkMode {
		return rgb(20, 23, 28), rgb(34, 38, 45), rgb(241, 244, 248), rgb(72, 79, 91), rgb(11, 13, 16)
	}
	return rgb(247, 249, 252), rgb(255, 255, 255), rgb(25, 32, 43), rgb(226, 231, 238), rgb(176, 184, 196)
}

func accentColor() uintptr {
	if darkMode {
		return rgb(96, 165, 250)
	}
	return rgb(37, 99, 235)
}

func accentSoftColor() uintptr {
	if darkMode {
		return rgb(39, 55, 78)
	}
	return rgb(229, 238, 255)
}

func mutedTextColor() uintptr {
	if darkMode {
		return rgb(164, 174, 188)
	}
	return rgb(100, 112, 128)
}

func borderColor() uintptr {
	if darkMode {
		return rgb(61, 68, 79)
	}
	return rgb(216, 222, 231)
}

func dangerColor() uintptr {
	if darkMode {
		return rgb(220, 70, 84)
	}
	return rgb(205, 45, 62)
}

func rebuildThemeBrushes() {
	if themeBrushBg != 0 {
		procDeleteObject.Call(uintptr(themeBrushBg))
		themeBrushBg = 0
	}
	if themeBrushCtl != 0 {
		procDeleteObject.Call(uintptr(themeBrushCtl))
		themeBrushCtl = 0
	}
	bg, ctl, _, _, _ := themeColors()
	b1, _, _ := procCreateSolidBrush.Call(bg)
	b2, _, _ := procCreateSolidBrush.Call(ctl)
	themeBrushBg = syscall.Handle(b1)
	themeBrushCtl = syscall.Handle(b2)
}

func applyTheme() {
	rebuildThemeBrushes()
	if hwndMain != 0 {
		procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
	}
	for _, tc := range translatedControls {
		if tc.h != 0 {
			procInvalidateRect.Call(uintptr(tc.h), 0, 1)
		}
	}
	for _, h := range hQualityRadios {
		if h != 0 {
			procInvalidateRect.Call(uintptr(h), 0, 1)
		}
	}
	for _, h := range hMaterialRadios {
		if h != 0 {
			procInvalidateRect.Call(uintptr(h), 0, 1)
		}
	}
	for _, h := range []syscall.Handle{hRootEdit, hAddonCombo, hMatEdit, hCompileCheck, hStatusLabel, hFileLabel, hDimLabel, hAlphaLabel, hAlphaTrack, hLogo} {
		if h != 0 {
			procInvalidateRect.Call(uintptr(h), 0, 1)
		}
	}
	for _, h := range settingsControls {
		if h != 0 {
			procInvalidateRect.Call(uintptr(h), 0, 1)
		}
	}
}

func draw3DButton(dis *DRAWITEMSTRUCT) {
	if dis == nil || dis.HDC == 0 || dis.HwndItem == 0 {
		return
	}
	_, ctlColor, baseText, _, _ := themeColors()
	fillColor := ctlColor
	outline := borderColor()
	textColor := baseText

	isPrimary := dis.CtlID == ID_BTN_CREATE || dis.CtlID == ID_BTN_AUTONOMOUS
	isDanger := dis.CtlID == ID_BTN_STOP
	isSelectedLang := false
	if code := languageButtonCode(dis.CtlID); code != "" && code == currentLang {
		isSelectedLang = true
	}
	if isPrimary {
		fillColor = accentColor()
		outline = accentColor()
		textColor = rgb(255, 255, 255)
	} else if isDanger {
		fillColor = dangerColor()
		outline = dangerColor()
		textColor = rgb(255, 255, 255)
	} else if isSelectedLang {
		fillColor = accentSoftColor()
		outline = accentColor()
		textColor = baseText
	}

	if dis.ItemState&ODS_DISABLED != 0 {
		if darkMode {
			fillColor = rgb(43, 47, 55)
			outline = rgb(55, 60, 70)
			textColor = rgb(105, 112, 124)
		} else {
			fillColor = rgb(236, 239, 244)
			outline = rgb(221, 226, 233)
			textColor = rgb(151, 159, 171)
		}
	}

	rc := dis.RcItem
	pressed := dis.ItemState&ODS_SELECTED != 0
	if !pressed && dis.ItemState&ODS_DISABLED == 0 {
		// Small solid shadow keeps the user's requested 3D feel without the old Win95 bevel.
		shadow := rc
		shadow.Top += 2
		shadow.Bottom += 2
		shadowColor := borderColor()
		br, _, _ := procCreateSolidBrush.Call(shadowColor)
		pen, _, _ := procCreatePen.Call(PS_SOLID, 1, shadowColor)
		if br != 0 && pen != 0 {
			oldB, _, _ := procSelectObject.Call(uintptr(dis.HDC), br)
			oldP, _, _ := procSelectObject.Call(uintptr(dis.HDC), pen)
			procRoundRect.Call(uintptr(dis.HDC), uintptr(shadow.Left), uintptr(shadow.Top), uintptr(shadow.Right), uintptr(shadow.Bottom), 10, 10)
			procSelectObject.Call(uintptr(dis.HDC), oldB)
			procSelectObject.Call(uintptr(dis.HDC), oldP)
		}
		if br != 0 {
			procDeleteObject.Call(br)
		}
		if pen != 0 {
			procDeleteObject.Call(pen)
		}
	}
	if pressed {
		rc.Left++
		rc.Right++
		rc.Top += 2
		rc.Bottom += 2
	}
	br, _, _ := procCreateSolidBrush.Call(fillColor)
	pen, _, _ := procCreatePen.Call(PS_SOLID, 1, outline)
	if br != 0 && pen != 0 {
		oldB, _, _ := procSelectObject.Call(uintptr(dis.HDC), br)
		oldP, _, _ := procSelectObject.Call(uintptr(dis.HDC), pen)
		procRoundRect.Call(uintptr(dis.HDC), uintptr(rc.Left), uintptr(rc.Top), uintptr(rc.Right), uintptr(rc.Bottom-2), 10, 10)
		procSelectObject.Call(uintptr(dis.HDC), oldB)
		procSelectObject.Call(uintptr(dis.HDC), oldP)
	}
	if br != 0 {
		procDeleteObject.Call(br)
	}
	if pen != 0 {
		procDeleteObject.Call(pen)
	}

	text := getText(dis.HwndItem)
	if dis.CtlID == ID_BTN_SETTINGS {
		arrow := "  ▸"
		if settingsOpen {
			arrow = "  ▾"
		}
		text = tr("settings") + arrow
	}
	if text == "" {
		return
	}

	trc := rc
	trc.Bottom -= 2
	if asset, ok := flagAssets[int(dis.CtlID)]; ok && asset.w > 0 {
		fr := RECT{Left: trc.Left + 9, Top: trc.Top + 6, Right: trc.Left + 39, Bottom: trc.Bottom - 6}
		drawAsset(dis.HDC, asset, fr, 0)
		trc.Left += 42
	}
	procSetBkMode.Call(uintptr(dis.HDC), 1)
	procSetTextColor.Call(uintptr(dis.HDC), textColor)
	if hFontButton != 0 {
		old, _, _ := procSelectObject.Call(uintptr(dis.HDC), uintptr(hFontButton))
		defer procSelectObject.Call(uintptr(dis.HDC), old)
	}
	buf := append(utf16.Encode([]rune(text)), 0)
	procDrawTextW.Call(uintptr(dis.HDC), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)-1), uintptr(unsafe.Pointer(&trc)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	runtime.KeepAlive(buf)
}

func drawLogoStatic(dis *DRAWITEMSTRUCT) {
	if dis == nil || dis.HDC == 0 {
		return
	}
	bg, _, _, _, _ := themeColors()
	brush, _, _ := procCreateSolidBrush.Call(bg)
	if brush != 0 {
		procFillRect.Call(uintptr(dis.HDC), uintptr(unsafe.Pointer(&dis.RcItem)), brush)
		procDeleteObject.Call(brush)
	}
	asset := logoLightAsset
	if darkMode {
		asset = logoDarkAsset
	}
	drawAsset(dis.HDC, asset, dis.RcItem, 4)
}

func isStepHandle(h syscall.Handle) bool {
	return h == hStepImage || h == hStepCS2 || h == hStepVmat || h == hStepType || h == hStepQuality
}

func isMutedHandle(h syscall.Handle) bool {
	return h == hAppSubtitle || h == hDimLabel || h == hFileLabel || h == hQualityHint ||
		h == hVersionLabel || h == hGameLabel || h == hLanguageLabel || h == hCustomWorkersLabel || h == hOverwriteLabel || h == hThemeLabel ||
		h == hOutputLabel || h == hAutoSpeedLabel
}

func controlColor(hdc syscall.Handle, msg uint32, ctlHandle syscall.Handle) uintptr {
	if themeBrushBg == 0 || themeBrushCtl == 0 {
		rebuildThemeBrushes()
	}
	bg, ctl, text, _, _ := themeColors()
	if isStepHandle(ctlHandle) {
		text = accentColor()
	} else if isMutedHandle(ctlHandle) {
		text = mutedTextColor()
	}
	procSetTextColor.Call(uintptr(hdc), text)
	switch msg {
	case WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		procSetBkColor.Call(uintptr(hdc), ctl)
		return uintptr(themeBrushCtl)
	default:
		procSetBkMode.Call(uintptr(hdc), 1)
		procSetBkColor.Call(uintptr(hdc), bg)
		return uintptr(themeBrushBg)
	}
}

func fillLanguageCombo() {
	if hLanguageCombo == 0 {
		return
	}
	procSendMessageW.Call(uintptr(hLanguageCombo), CB_RESETCONTENT, 0, 0)
	selected := 0
	for i, ld := range languageDefs {
		sendMessageString(hLanguageCombo, CB_ADDSTRING, ld.name)
		if ld.code == currentLang {
			selected = i
		}
	}
	procSendMessageW.Call(uintptr(hLanguageCombo), CB_SETCURSEL, uintptr(selected), 0)
}

func fillShaderCombo() {
	if hShaderCombo == 0 {
		return
	}
	sel, _, _ := procSendMessageW.Call(uintptr(hShaderCombo), CB_GETCURSEL, 0, 0)
	if int(sel) < 0 || int(sel) > 1 {
		sel = 0
	}
	procSendMessageW.Call(uintptr(hShaderCombo), CB_RESETCONTENT, 0, 0)
	sendMessageString(hShaderCombo, CB_ADDSTRING, tr("shader_world"))
	sendMessageString(hShaderCombo, CB_ADDSTRING, tr("shader_prop"))
	procSendMessageW.Call(uintptr(hShaderCombo), CB_SETCURSEL, sel, 0)
}

func installV0175Translations() {
	vals := map[string]map[string]string{
		"en":    {"retry_compile": "Retry failed compile once", "compiler_lock": "Compiler locking for shared VMAT resources", "overwrite": "Overwrite mode", "overwrite_ask": "Ask", "overwrite_skip": "Skip existing", "overwrite_replace": "Replace existing", "auto_custom": "Custom", "custom_workers": "Custom workers"},
		"cs":    {"retry_compile": "Jednou opakovat neúspěšnou kompilaci", "compiler_lock": "Zamknout kompilaci sdílených VMAT prostředků", "overwrite": "Režim přepsání", "overwrite_ask": "Zeptat se", "overwrite_skip": "Přeskočit existující", "overwrite_replace": "Nahradit existující", "auto_custom": "Vlastní", "custom_workers": "Vlastní pracovníci"},
		"de":    {"retry_compile": "Fehlgeschlagene Kompilierung einmal wiederholen", "compiler_lock": "Compiler-Sperre für gemeinsam genutzte VMAT-Ressourcen", "overwrite": "Überschreibmodus", "overwrite_ask": "Nachfragen", "overwrite_skip": "Vorhandene überspringen", "overwrite_replace": "Vorhandene ersetzen", "auto_custom": "Benutzerdefiniert", "custom_workers": "Eigene Worker"},
		"es":    {"retry_compile": "Reintentar una vez si falla la compilación", "compiler_lock": "Bloqueo del compilador para recursos VMAT compartidos", "overwrite": "Modo de sobrescritura", "overwrite_ask": "Preguntar", "overwrite_skip": "Omitir existentes", "overwrite_replace": "Reemplazar existentes", "auto_custom": "Personalizado", "custom_workers": "Trabajadores personalizados"},
		"fr":    {"retry_compile": "Réessayer une fois si la compilation échoue", "compiler_lock": "Verrouillage du compilateur pour les ressources VMAT partagées", "overwrite": "Mode d’écrasement", "overwrite_ask": "Demander", "overwrite_skip": "Ignorer les existants", "overwrite_replace": "Remplacer les existants", "auto_custom": "Personnalisé", "custom_workers": "Workers personnalisés"},
		"pl":    {"retry_compile": "Ponów nieudaną kompilację jeden raz", "compiler_lock": "Blokada kompilatora dla współdzielonych zasobów VMAT", "overwrite": "Tryb nadpisywania", "overwrite_ask": "Pytaj", "overwrite_skip": "Pomiń istniejące", "overwrite_replace": "Zastąp istniejące", "auto_custom": "Własne", "custom_workers": "Własna liczba workerów"},
		"pt-BR": {"retry_compile": "Tentar novamente uma vez se a compilação falhar", "compiler_lock": "Bloqueio do compilador para recursos VMAT compartilhados", "overwrite": "Modo de substituição", "overwrite_ask": "Perguntar", "overwrite_skip": "Ignorar existentes", "overwrite_replace": "Substituir existentes", "auto_custom": "Personalizado", "custom_workers": "Workers personalizados"},
		"ru":    {"retry_compile": "Повторить неудачную компиляцию один раз", "compiler_lock": "Блокировка компилятора для общих ресурсов VMAT", "overwrite": "Режим перезаписи", "overwrite_ask": "Спрашивать", "overwrite_skip": "Пропускать существующие", "overwrite_replace": "Заменять существующие", "auto_custom": "Пользовательский", "custom_workers": "Число воркеров"},
		"tr":    {"retry_compile": "Başarısız derlemeyi bir kez yeniden dene", "compiler_lock": "Paylaşılan VMAT kaynakları için derleyici kilidi", "overwrite": "Üzerine yazma modu", "overwrite_ask": "Sor", "overwrite_skip": "Var olanları atla", "overwrite_replace": "Var olanları değiştir", "auto_custom": "Özel", "custom_workers": "Özel worker sayısı"},
	}
	for lang, m := range vals {
		if translations[lang] == nil {
			translations[lang] = map[string]string{}
		}
		for k, v := range m {
			translations[lang][k] = v
		}
	}
}

func populateOverwriteCombo() {
	if hOverwriteCombo == 0 {
		return
	}
	procSendMessageW.Call(uintptr(hOverwriteCombo), CB_RESETCONTENT, 0, 0)
	for _, key := range []string{"overwrite_ask", "overwrite_skip", "overwrite_replace"} {
		sendMessageString(hOverwriteCombo, CB_ADDSTRING, tr(key))
	}
	if selectedOverwriteMode < 0 || selectedOverwriteMode > 2 {
		selectedOverwriteMode = 0
	}
	procSendMessageW.Call(uintptr(hOverwriteCombo), CB_SETCURSEL, uintptr(selectedOverwriteMode), 0)
}

func clampCustomWorkers(n int) int {
	if n < 1 {
		return 1
	}
	if n > 32 {
		return 32
	}
	return n
}

func populateCustomWorkersCombo() {
	if hCustomWorkersCombo == 0 {
		return
	}
	procSendMessageW.Call(uintptr(hCustomWorkersCombo), CB_RESETCONTENT, 0, 0)
	for i := 1; i <= 32; i++ {
		sendMessageString(hCustomWorkersCombo, CB_ADDSTRING, fmt.Sprintf("%d", i))
	}
	selectedCustomWorkers = clampCustomWorkers(selectedCustomWorkers)
	procSendMessageW.Call(uintptr(hCustomWorkersCombo), CB_SETCURSEL, uintptr(selectedCustomWorkers-1), 0)
}

func syncCustomWorkersFromUI() int {
	if hCustomWorkersCombo != 0 {
		idx, _, _ := procSendMessageW.Call(uintptr(hCustomWorkersCombo), CB_GETCURSEL, 0, 0)
		if int(idx) >= 0 && int(idx) < 32 {
			selectedCustomWorkers = int(idx) + 1
			appSettings.CustomWorkers = selectedCustomWorkers
		}
	}
	return clampCustomWorkers(selectedCustomWorkers)
}

func applyLanguage() {
	for _, tc := range translatedControls {
		if tc.h != 0 {
			setText(tc.h, tr(tc.key))
		}
	}
	fillLanguageCombo()
	fillShaderCombo()
	populateOverwriteCombo()
	updateSettingsButtonText()
	if hThemeBtn != 0 {
		if darkMode {
			setText(hThemeBtn, tr("light"))
		} else {
			setText(hThemeBtn, tr("dark"))
		}
	}
	if hFullscreenBtn != 0 {
		if fullscreen {
			setText(hFullscreenBtn, tr("exit_fullscreen"))
		} else {
			setText(hFullscreenBtn, tr("fullscreen"))
		}
	}
	if currentImage == nil && hFileLabel != 0 {
		setText(hFileLabel, tr("no_image"))
		if hDimLabel != 0 {
			setText(hDimLabel, tr("image_waiting"))
		}
	}
	updateAlphaLabel()
	updateGameToolsUI()
	if hwndMain != 0 {
		var rc RECT
		procGetClientRect.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(&rc)))
		layoutMainUI(rc.Right-rc.Left, rc.Bottom-rc.Top)
	}
	applyTheme()
	saveSettings()
}

func updateSettingsButtonText() {
	if hSettingsBtn == 0 {
		return
	}
	arrow := "  ▸"
	if settingsOpen {
		arrow = "  ▾"
	}
	setText(hSettingsBtn, tr("settings")+arrow)
}

func showSettings(open bool) {
	settingsOpen = open
	for _, h := range settingsControls {
		if h != 0 {
			if open {
				procShowWindow.Call(uintptr(h), SW_SHOW)
			} else {
				procShowWindow.Call(uintptr(h), SW_HIDE)
			}
		}
	}
	// In the normal window, grow/shrink the frame with the Settings drawer.
	// When maximized, keep the frame maximized and simply relayout the client area.
	if !fullscreen && hwndMain != 0 {
		width := int32(880)
		if open {
			width = 1150
		}
		procSetWindowPos.Call(uintptr(hwndMain), 0, 0, 0, uintptr(width), uintptr(int32(850)), SWP_NOMOVE|SWP_NOZORDER)
	}
	updateSettingsButtonText()
	var rc RECT
	if hwndMain != 0 {
		procGetClientRect.Call(uintptr(hwndMain), uintptr(unsafe.Pointer(&rc)))
		layoutMainUI(rc.Right-rc.Left, rc.Bottom-rc.Top)
	}
	applyTheme()
}

func toggleFullscreen() {
	if hwndMain == 0 {
		return
	}
	// v13 uses the normal Windows maximize/restore behavior. This keeps the
	// title bar with minimize / restore / close instead of switching to a
	// borderless popup, and WM_SIZE handles the responsive layout.
	if fullscreen {
		procShowWindow.Call(uintptr(hwndMain), SW_RESTORE)
	} else {
		procShowWindow.Call(uintptr(hwndMain), SW_MAXIMIZE)
	}
}

func fillRectColor(hdc syscall.Handle, rc RECT, c uintptr) {
	br, _, _ := procCreateSolidBrush.Call(c)
	if br != 0 {
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rc)), br)
		procDeleteObject.Call(br)
	}
}

func drawMainChrome(hdc syscall.Handle, hwnd syscall.Handle) {
	if hdc == 0 {
		return
	}
	var rc RECT
	procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	bg, _, _, _, _ := themeColors()
	fillRectColor(hdc, rc, bg)
	fillRectColor(hdc, RECT{Left: 0, Top: 0, Right: rc.Right, Bottom: 4}, accentColor())

	settingsW := int32(280)
	settingsX := rc.Right - settingsW - 16
	leftRight := rc.Right - 20
	if settingsOpen {
		leftRight = settingsX - 14
	}
	if leftRight < 650 {
		leftRight = 650
	}
	// Stop the header divider before the Settings drawer so it cannot run
	// underneath the B.I.T. logo.
	fillRectColor(hdc, RECT{Left: 16, Top: 72, Right: leftRight, Bottom: 73}, borderColor())
	if settingsOpen {
		fillRectColor(hdc, RECT{Left: settingsX - 8, Top: 74, Right: settingsX - 7, Bottom: rc.Bottom - 14}, borderColor())
	}
	workR := leftRight
	// Autonomous speed now has its title on its own line and a fixed 3-column
	// grid beneath it. The divider therefore stays stable in every language.
	speedDivider := int32(686)
	// Compact, consistent section dividers. Autonomous speed is intentionally
	// outside section 5 and gets its own divider below it.
	for _, y := range []int32{160, 276, 358, 464, 584, speedDivider} {
		fillRectColor(hdc, RECT{Left: 24, Top: y, Right: workR, Bottom: y + 1}, borderColor())
	}
}

func moveControl(h syscall.Handle, x, y, w, height int32) {
	if h == 0 || w <= 0 || height <= 0 {
		return
	}
	procSetWindowPos.Call(uintptr(h), 0, uintptr(x), uintptr(y), uintptr(w), uintptr(height), SWP_NOZORDER)
}

func translatedPreferredWidth(text string, minW, maxW int32, charPx int32) int32 {
	// Simple DPI-friendly text width estimate. It intentionally over-allocates a
	// little space so German/Russian/Czech/etc. do not lose the end of labels.
	n := int32(len([]rune(strings.TrimSpace(text))))
	w := n*charPx + 28
	if w < minW {
		w = minW
	}
	if maxW > 0 && w > maxW {
		w = maxW
	}
	return w
}

func translatedRowWidths(keys []string, totalW, gap, minEach int32) []int32 {
	out := make([]int32, len(keys))
	if len(keys) == 0 {
		return out
	}
	usable := totalW - gap*int32(len(keys)-1)
	if usable < minEach*int32(len(keys)) {
		usable = minEach * int32(len(keys))
	}
	weights := make([]int32, len(keys))
	var totalWeight int32
	for i, key := range keys {
		// +5 prevents very short words such as "Normal" from becoming tiny.
		w := int32(len([]rune(tr(key)))) + 5
		if w < 8 {
			w = 8
		}
		weights[i] = w
		totalWeight += w
	}
	remaining := usable - minEach*int32(len(keys))
	for i := range out {
		out[i] = minEach
		if remaining > 0 && totalWeight > 0 {
			out[i] += remaining * weights[i] / totalWeight
		}
	}
	// Give rounding leftovers to the last control.
	var sum int32
	for _, w := range out {
		sum += w
	}
	if diff := usable - sum; diff > 0 {
		out[len(out)-1] += diff
	}
	return out
}

func layoutMainUI(clientW, clientH int32) {
	if clientW <= 0 || clientH <= 0 {
		return
	}

	// V0.17.4 uses a denser responsive grid. The normal window is smaller,
	// but controls still shrink cleanly instead of drifting off-screen.
	settingsW := int32(280)
	settingsX := clientW - settingsW - 16
	leftRight := clientW - 20
	if settingsOpen {
		leftRight = settingsX - 14
	}
	if leftRight < 650 {
		leftRight = 650
	}
	workW := leftRight - 24
	if workW > 820 {
		workW = 820
	}
	if workW < 600 {
		workW = 600
	}

	// Header / branding.
	moveControl(hAppTitle, 24, 12, min32(540, workW), 32)
	moveControl(hAppSubtitle, 24, 44, min32(620, workW), 20)
	moveControl(hSettingsBtn, clientW-142, 15, 118, 36)

	// 1. Image
	moveControl(hStepImage, 24, 82, 145, 24)
	moveControl(hDimLabel, 172, 85, workW-148, 24)
	chooseW := translatedPreferredWidth(tr("choose_png"), 166, 245, 8)
	moveControl(hChooseBtn, 24, 111, chooseW, 36)
	moveControl(hFileLabel, 24+chooseW+12, 118, workW-chooseW-12, 24)

	// 2. Game tools / Addon
	moveControl(hStepCS2, 24, 171, workW, 24)
	detectW := translatedPreferredWidth(getText(hDetectBtn), 102, 165, 8)
	browseW := translatedPreferredWidth(tr("browse"), 100, 155, 8)
	rootW := workW - detectW - browseW - 20
	if rootW < 250 {
		rootW = 250
	}
	moveControl(hRootEdit, 24, 199, rootW, 28)
	moveControl(hDetectBtn, 34+rootW, 197, detectW, 32)
	moveControl(hBrowseBtn, 44+rootW+detectW, 197, browseW, 32)
	moveControl(hAddonLabel, 24, 238, 52, 22)
	refreshW := translatedPreferredWidth(tr("refresh"), 88, 145, 8)
	refreshX := 24 + workW - refreshW
	comboW := refreshX - 8 - 82
	if comboW < 220 {
		comboW = 220
	}
	moveControl(hAddonCombo, 82, 234, comboW, 190)
	moveControl(hRefreshBtn, refreshX, 232, refreshW, 32)

	// 3. VMAT
	moveControl(hStepVmat, 24, 287, workW, 24)
	moveControl(hMatEdit, 24, 315, workW, 29)

	// 4. Material
	moveControl(hStepType, 24, 369, workW, 24)
	matWidths := translatedRowWidths([]string{"mat_opaque", "mat_cutout", "mat_translucent"}, workW, 8, 118)
	x := int32(24)
	for i, h := range hMaterialRadios {
		moveControl(h, x, 397, matWidths[i], 28)
		x += matWidths[i] + 8
	}
	moveControl(hAlphaLabel, 24, 428, 170, 22)
	moveControl(hAlphaTrack, 198, 424, min32(300, workW-174), 30)

	// 5. Quality
	moveControl(hStepQuality, 24, 475, workW, 24)
	qWidths := translatedRowWidths([]string{"quality_original", "quality_hd", "quality_high", "quality_mid", "quality_low"}, workW, 6, 78)
	x = 24
	for i, h := range hQualityRadios {
		moveControl(h, x, 503, qWidths[i], 28)
		x += qWidths[i] + 6
	}
	moveControl(hQualityHint, 24, 532, workW, 20)
	moveControl(hCompileCheck, 24, 555, workW, 28)

	// Autonomous speed uses a language-safe grid. The label gets its own row,
	// then the choices use the full workspace width instead of being squeezed
	// into whatever remains to the right of a translated label.
	//
	//   Slow      Normal      Fast
	//   Extreme               Custom
	//
	// This keeps English compact while giving German, Portuguese, Russian, etc.
	// enough room without overlaps or awkward drifting between columns.
	speedTop := int32(594)
	moveControl(hAutoSpeedLabel, 24, speedTop, workW, 20)
	colGap := int32(10)
	colW := (workW - 2*colGap) / 3
	row1Y := speedTop + 23
	row2Y := speedTop + 55
	moveControl(hAutoModeRadios[0], 24, row1Y, colW, 30)
	moveControl(hAutoModeRadios[1], 24+colW+colGap, row1Y, colW, 30)
	moveControl(hAutoModeRadios[2], 24+2*(colW+colGap), row1Y, colW, 30)
	moveControl(hAutoModeRadios[3], 24, row2Y, colW, 30)
	moveControl(hAutoModeRadios[4], 24+2*(colW+colGap), row2Y, colW, 30)

	// Bottom action strip stays anchored to the window bottom. Status is kept
	// below the complete two-row speed grid at every supported window size.
	bottomY := clientH - 54
	if bottomY < 756 {
		bottomY = 756
	}
	speedDivider := int32(686)
	statusY := speedDivider + 9
	moveControl(hOutputLabel, 24, statusY, 100, 20)
	moveControl(hStatusLabel, 24, statusY+22, workW, 24)

	gap := int32(7)
	controls := []syscall.Handle{hCreateBtn, hOutputBtn, hLogBtn, hAutoBtn, hStopBtn}
	keys := []string{"create_vmat", "open_output", "open_log", "autonomous", "stop_auto"}
	widths := translatedRowWidths(keys, workW, gap, 92)
	x = 24
	for i, h := range controls {
		moveControl(h, x, bottomY, widths[i], 40)
		x += widths[i] + gap
	}

	if settingsOpen {
		xs := settingsX + 8
		settingsCtlW := int32(260)
		moveControl(hLogo, xs+47, 74, 166, 106)
		moveControl(hGameLabel, xs, 187, settingsCtlW, 20)
		moveControl(hGameCombo, xs, 207, settingsCtlW, 126)
		moveControl(hLanguageLabel, xs, 236, settingsCtlW, 20)
		for i, h := range hLanguageButtons {
			moveControl(h, xs, 256+int32(i)*25, settingsCtlW, 23)
		}
		// Long localized checkbox labels need both width and a second line.
		moveControl(hRetryCompile, xs, 482, settingsCtlW, 32)
		moveControl(hCompilerLock, xs, 515, settingsCtlW, 34)

		// Custom workers follows the exact same Settings pattern as Game and
		// Overwrite mode: muted category label above a full-width selector.
		moveControl(hCustomWorkersLabel, xs, 553, settingsCtlW, 18)
		moveControl(hCustomWorkersCombo, xs, 572, settingsCtlW, 180)
		moveControl(hOverwriteLabel, xs, 605, settingsCtlW, 18)
		moveControl(hOverwriteCombo, xs, 624, settingsCtlW, 92)
		moveControl(hThemeLabel, xs, 657, settingsCtlW, 18)
		moveControl(hThemeBtn, xs, 676, settingsCtlW, 28)
		moveControl(hFullscreenBtn, xs, 708, settingsCtlW, 26)
		moveControl(hJunkOpen, xs, 738, settingsCtlW, 26)
		moveControl(hJunkClear, xs, 768, settingsCtlW, 26)
		moveControl(hVersionLabel, xs, 801, settingsCtlW, 18)
	}
	procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
}

func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func handleLanguageChange() {
	if hLanguageCombo == 0 {
		return
	}
	r, _, _ := procSendMessageW.Call(uintptr(hLanguageCombo), CB_GETCURSEL, 0, 0)
	i := int(r)
	if i >= 0 && i < len(languageDefs) {
		currentLang = languageDefs[i].code
		applyLanguage()
	}
}

func sendMessageString(hwnd syscall.Handle, msg uint32, s string) uintptr {
	p := u16(s)
	r, _, _ := procSendMessageW.Call(uintptr(hwnd), uintptr(msg), 0, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
	return r
}

func appDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if strings.TrimSpace(base) == "" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			base = filepath.Join(home, "AppData", "Local")
		} else {
			base = "."
		}
	}
	p := filepath.Join(base, "BITTextureTool")
	_ = os.MkdirAll(p, 0755)
	return p
}

func junkDir() string {
	p := filepath.Join(appDataDir(), "Junk")
	_ = os.MkdirAll(p, 0755)
	return p
}

func settingsPath() string { return filepath.Join(appDataDir(), "settings.json") }

func legacySettingsPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if strings.TrimSpace(base) == "" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			base = filepath.Join(home, "AppData", "Local")
		} else {
			return ""
		}
	}
	return filepath.Join(base, "CS2MaterialDropper", "settings.json")
}

func loadSettings() {
	appSettings = AppSettings{Language: "en", Game: "cs2", DarkMode: false, MaterialMode: 0, Quality: 0, AutoMode: 1, RetryCompile: true, CompilerLock: true, OverwriteMode: 0, CustomWorkers: 8}
	b, err := os.ReadFile(settingsPath())
	// One-time compatibility with the previous CS2 Material Dropper name so users
	// keep their CS2 path/addon/theme when moving to B.I.T. Texture Tool.
	if err != nil {
		if legacy := legacySettingsPath(); legacy != "" {
			b, err = os.ReadFile(legacy)
		}
	}
	if err == nil {
		var st AppSettings
		if json.Unmarshal(b, &st) == nil {
			if _, ok := translations[st.Language]; ok {
				appSettings.Language = st.Language
			}
			if validGameKey(st.Game) {
				appSettings.Game = st.Game
			}
			appSettings.DarkMode = st.DarkMode
			appSettings.CS2Root = st.CS2Root
			appSettings.LastAddon = st.LastAddon
			appSettings.LastImageDir = st.LastImageDir
			if st.MaterialMode >= 0 && st.MaterialMode <= 2 {
				appSettings.MaterialMode = st.MaterialMode
			}
			if st.Quality >= 0 && st.Quality <= 4 {
				appSettings.Quality = st.Quality
			}
			// Preserve safe defaults when upgrading from older settings files that do not contain these keys.
			if bytes.Contains(b, []byte("\"retry_compile\"")) {
				appSettings.RetryCompile = st.RetryCompile
			}
			if bytes.Contains(b, []byte("\"compiler_lock\"")) {
				appSettings.CompilerLock = st.CompilerLock
			}
			if st.OverwriteMode >= 0 && st.OverwriteMode <= 2 {
				appSettings.OverwriteMode = st.OverwriteMode
			}
			if st.CustomWorkers >= 1 && st.CustomWorkers <= 32 {
				appSettings.CustomWorkers = st.CustomWorkers
			}
		}
	}
	currentLang = appSettings.Language
	selectedGame = appSettings.Game
	if !validGameKey(selectedGame) {
		selectedGame = "cs2"
		appSettings.Game = "cs2"
	}
	darkMode = appSettings.DarkMode
	selectedMaterialMode = appSettings.MaterialMode
	// V0.17.4 intentionally starts on Original every launch.
	appSettings.Quality = 0
	selectedQuality = 0
	// v13 always starts Autonomous Production in Normal mode.
	appSettings.AutoMode = 1
	selectedAutoMode = 1
	retryCompile = appSettings.RetryCompile
	compilerLockEnabled = appSettings.CompilerLock
	selectedOverwriteMode = appSettings.OverwriteMode
	selectedCustomWorkers = clampCustomWorkers(appSettings.CustomWorkers)
}

func saveSettings() {
	appSettings.Language = currentLang
	appSettings.Game = selectedGame
	appSettings.DarkMode = darkMode
	appSettings.MaterialMode = selectedMaterialMode
	appSettings.Quality = selectedQuality
	appSettings.AutoMode = selectedAutoMode
	appSettings.RetryCompile = retryCompile
	appSettings.CompilerLock = compilerLockEnabled
	appSettings.OverwriteMode = selectedOverwriteMode
	appSettings.CustomWorkers = selectedCustomWorkers
	if hRootEdit != 0 {
		appSettings.CS2Root = strings.TrimSpace(getText(hRootEdit))
	}
	if selectedAddon != "" {
		appSettings.LastAddon = selectedAddon
	}
	b, err := json.MarshalIndent(appSettings, "", "  ")
	if err == nil {
		_ = os.WriteFile(settingsPath(), b, 0644)
	}
}

func decodeEmbeddedAsset(data []byte, bg color.NRGBA) imageAsset {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return imageAsset{}
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pixels := make([]byte, w*h*4)
	p := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r16, g16, bl16, a16 := img.At(x, y).RGBA()
			a := float64(a16) / 65535.0
			r := uint8(float64(r16>>8)*a + float64(bg.R)*(1-a) + 0.5)
			g := uint8(float64(g16>>8)*a + float64(bg.G)*(1-a) + 0.5)
			bl := uint8(float64(bl16>>8)*a + float64(bg.B)*(1-a) + 0.5)
			pixels[p+0] = bl
			pixels[p+1] = g
			pixels[p+2] = r
			pixels[p+3] = 255
			p += 4
		}
	}
	return imageAsset{img: img, pixels: pixels, w: w, h: h}
}

func loadEmbeddedAssets() {
	logoLightAsset = decodeEmbeddedAsset(logoLightPNG, color.NRGBA{R: 247, G: 249, B: 252, A: 255})
	logoDarkAsset = decodeEmbeddedAsset(logoDarkPNG, color.NRGBA{R: 20, G: 23, B: 28, A: 255})
	flagAssets = map[int]imageAsset{
		ID_LANG_EN: decodeEmbeddedAsset(flagGBPNG, color.NRGBA{A: 255}),
		ID_LANG_RU: decodeEmbeddedAsset(flagRUPNG, color.NRGBA{A: 255}),
		ID_LANG_CS: decodeEmbeddedAsset(flagCZPNG, color.NRGBA{A: 255}),
		ID_LANG_BR: decodeEmbeddedAsset(flagBRPNG, color.NRGBA{A: 255}),
		ID_LANG_FR: decodeEmbeddedAsset(flagFRPNG, color.NRGBA{A: 255}),
		ID_LANG_DE: decodeEmbeddedAsset(flagDEPNG, color.NRGBA{A: 255}),
		ID_LANG_ES: decodeEmbeddedAsset(flagESPNG, color.NRGBA{A: 255}),
		ID_LANG_PL: decodeEmbeddedAsset(flagPLPNG, color.NRGBA{A: 255}),
		ID_LANG_TR: decodeEmbeddedAsset(flagTRPNG, color.NRGBA{A: 255}),
	}
}

func drawAsset(hdc syscall.Handle, asset imageAsset, rc RECT, pad int32) {
	if hdc == 0 || asset.w <= 0 || asset.h <= 0 || len(asset.pixels) == 0 {
		return
	}
	availW := rc.Right - rc.Left - 2*pad
	availH := rc.Bottom - rc.Top - 2*pad
	if availW <= 0 || availH <= 0 {
		return
	}
	scale := math.Min(float64(availW)/float64(asset.w), float64(availH)/float64(asset.h))
	dw := int32(float64(asset.w) * scale)
	dh := int32(float64(asset.h) * scale)
	dx := rc.Left + (rc.Right-rc.Left-dw)/2
	dy := rc.Top + (rc.Bottom-rc.Top-dh)/2
	bmi := BITMAPINFO{BmiHeader: BITMAPINFOHEADER{
		BiSize: uint32(unsafe.Sizeof(BITMAPINFOHEADER{})), BiWidth: int32(asset.w), BiHeight: -int32(asset.h),
		BiPlanes: 1, BiBitCount: 32, BiCompression: 0, BiSizeImage: uint32(len(asset.pixels)),
	}}
	procStretchDIBits.Call(uintptr(hdc), uintptr(dx), uintptr(dy), uintptr(dw), uintptr(dh), 0, 0,
		uintptr(asset.w), uintptr(asset.h), uintptr(unsafe.Pointer(&asset.pixels[0])), uintptr(unsafe.Pointer(&bmi)), DIB_RGB_COLORS, SRCCOPY)
}

func languageButtonCode(id uint32) string {
	switch id {
	case ID_LANG_EN:
		return "en"
	case ID_LANG_RU:
		return "ru"
	case ID_LANG_CS:
		return "cs"
	case ID_LANG_BR:
		return "pt-BR"
	case ID_LANG_FR:
		return "fr"
	case ID_LANG_DE:
		return "de"
	case ID_LANG_ES:
		return "es"
	case ID_LANG_PL:
		return "pl"
	case ID_LANG_TR:
		return "tr"
	}
	return ""
}

func languageButtonName(id uint32) string {
	switch id {
	case ID_LANG_EN:
		return "English"
	case ID_LANG_RU:
		return "Русский"
	case ID_LANG_CS:
		return "Čeština"
	case ID_LANG_BR:
		return "Português (Brasil)"
	case ID_LANG_FR:
		return "Français"
	case ID_LANG_DE:
		return "Deutsch"
	case ID_LANG_ES:
		return "Español"
	case ID_LANG_PL:
		return "Polski"
	case ID_LANG_TR:
		return "Türkçe"
	}
	return ""
}

func main() {
	// Win32 windows and their message queues are owned by the creating OS thread.
	// A Go goroutine may otherwise migrate between OS threads, which can make the
	// window stop receiving messages and appear frozen. Keep the GUI on one thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	loadSettings()
	loadEmbeddedAssets()
	defer func() {
		if r := recover(); r != nil {
			writeCrashLog("main", r)
		}
	}()
	procCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED
	defer procCoUninitialize.Call()
	procInitCommonControls.Call()
	hi, _, _ := procGetModuleHandleW.Call(0)
	hInstance = syscall.Handle(hi)
	createAppFonts()
	defer deleteAppFonts()

	mainClass := u16("CS2MaterialDropperMain")
	cursor, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	mainWndProcCallback = syscall.NewCallback(mainWndProc)
	cropWndProcCallback = syscall.NewCallback(cropWndProc)
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   mainWndProcCallback,
		HInstance:     hInstance,
		HCursor:       syscall.Handle(cursor),
		HbrBackground: 0, // app paints the full themed background itself
		LpszClassName: mainClass,
	}
	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return
	}

	cropClass := u16("CS2MaterialDropperCrop")
	wc2 := wc
	wc2.LpfnWndProc = cropWndProcCallback
	wc2.LpszClassName = cropClass
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc2)))

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(mainClass)),
		uintptr(unsafe.Pointer(u16(appTitle))),
		uintptr(WS_OVERLAPPEDWINDOW),
		uintptr(int32(100)), uintptr(int32(35)), uintptr(int32(1150)), uintptr(int32(890)),
		0, 0, uintptr(hInstance), 0,
	)
	hwndMain = syscall.Handle(hwnd)
	if hwndMain == 0 {
		return
	}
	procShowWindow.Call(uintptr(hwndMain), SW_SHOW)
	procUpdateWindow.Call(uintptr(hwndMain))

	var m MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if m.Message == WM_KEYDOWN && m.WParam == VK_F11 {
			toggleFullscreen()
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func mainWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			writeCrashLog("mainWndProc", r)
			msgBox(hwnd, "The app caught an internal error instead of closing.\n\nA crash log was written to the B.I.T. Texture Tool app-data folder.", appTitle, MB_OK|MB_ICONERROR)
			ret = 0
		}
	}()
	switch msg {
	case WM_ERASEBKGND:
		if hwnd == hwndMain {
			if themeBrushBg == 0 {
				rebuildThemeBrushes()
			}
			var rc RECT
			procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
			procFillRect.Call(wParam, uintptr(unsafe.Pointer(&rc)), uintptr(themeBrushBg))
			return 1
		}
	case WM_PAINT:
		if hwnd == hwndMain {
			var ps PAINTSTRUCT
			hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
			drawMainChrome(syscall.Handle(hdc), hwnd)
			procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
			return 0
		}
	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis != nil {
			if dis.CtlType == ODT_BUTTON {
				draw3DButton(dis)
				return 1
			}
			if dis.CtlType == ODT_STATIC && dis.CtlID == ID_LOGO {
				drawLogoStatic(dis)
				return 1
			}
		}
	case WM_CTLCOLORSTATIC, WM_CTLCOLOREDIT, WM_CTLCOLORBTN, WM_CTLCOLORLISTBOX:
		return controlColor(syscall.Handle(wParam), msg, syscall.Handle(lParam))
	case WM_APP_DETECT:
		detectMu.Lock()
		root := detectResult
		gameKey := detectResultGame
		detectBusy = false
		detectMu.Unlock()
		if gameKey != selectedGame {
			// The user changed games while detection was running. Start detection
			// again for the newly selected game instead of applying stale results.
			beginDetectCS2()
			return 0
		}
		if hDetectBtn != 0 {
			procEnableWindow.Call(uintptr(hDetectBtn), 1)
		}
		if root == "" {
			msg := currentGameName() + " was not found automatically. Use Browse and choose its Steam installation folder."
			setText(hStatusLabel, msg)
			msgBox(hwnd, msg, appTitle, MB_OK|MB_ICONINFORMATION)
		} else {
			setText(hRootEdit, root)
			appSettings.CS2Root = root
			saveSettings()
			setText(hStatusLabel, currentGameName()+" detected. Loading addon/custom folders...")
			beginPopulateAddons(root)
		}
		return 0
	case WM_APP_IMAGE:
		imageMu.Lock()
		res := imageResult
		imageBusy = false
		imageMu.Unlock()
		if hChooseBtn != 0 {
			procEnableWindow.Call(uintptr(hChooseBtn), 1)
		}
		if res.err != nil {
			setText(hStatusLabel, "Image load failed.")
			msgBox(hwnd, "Could not read or convert that image:\n"+res.err.Error(), appTitle, MB_OK|MB_ICONERROR)
			return 0
		}
		currentImage, currentPath, currentW, currentH = res.img, res.path, res.w, res.h
		currentOriginalPath = res.originalPath
		updateImageUIV10(res.hasAlpha)
		base := strings.TrimSuffix(filepath.Base(res.originalPath), filepath.Ext(res.originalPath))
		setText(hMatEdit, sanitizeSegment(base))
		if currentW != currentH {
			setText(hStatusLabel, fmt.Sprintf("%d x %d is not square. Opening crop automatically...", currentW, currentH))
			openCropDialog(hwnd)
		} else {
			setText(hStatusLabel, fmt.Sprintf(tr("image_ready"), currentW, currentH)+"  "+map[bool]string{true: tr("alpha_found"), false: tr("alpha_none")}[res.hasAlpha])
		}
		return 0
	case WM_APP_ADDONS:
		addonMu.Lock()
		ar := addonResult
		addonBusy = false
		addonMu.Unlock()
		if ar.game != "" && ar.game != selectedGame {
			return 0
		}
		if ar.err != nil {
			setText(hStatusLabel, "Could not read addon folder: "+ar.err.Error())
			return 0
		}
		addonNames = append([]string(nil), ar.names...)
		selectedAddon = ""
		procSendMessageW.Call(uintptr(hAddonCombo), CB_RESETCONTENT, 0, 0)
		for _, n := range addonNames {
			sendMessageString(hAddonCombo, CB_ADDSTRING, n)
		}
		if len(addonNames) == 0 {
			if selectedGame == "cs2" {
				setText(hStatusLabel, tr("no_addons"))
			} else {
				setText(hStatusLabel, "No addon/custom folders found for "+currentGameName()+".")
			}
			return 0
		}
		idx := 0
		want := appSettings.LastAddon
		for i, n := range addonNames {
			if n == want {
				idx = i
				break
			}
		}
		procSendMessageW.Call(uintptr(hAddonCombo), CB_SETCURSEL, uintptr(idx), 0)
		selectedAddon = addonNames[idx]
		setText(hStatusLabel, fmt.Sprintf(tr("found_addons"), len(addonNames), selectedAddon))
		saveSettings()
		return 0
	case WM_APP_STATUS:
		jobMu.Lock()
		st := jobStatus
		jobMu.Unlock()
		if st != "" {
			setText(hStatusLabel, st)
		}
		return 0
	case WM_APP_DONE:
		jobMu.Lock()
		res := jobResult
		jobMu.Unlock()
		createBusy = false
		if hCreateBtn != 0 {
			procEnableWindow.Call(uintptr(hCreateBtn), 1)
		}
		if res.OutputDir != "" {
			lastOutputDir = res.OutputDir
			procEnableWindow.Call(uintptr(hOutputBtn), 1)
		}
		if res.LogPath != "" {
			lastLogPath = res.LogPath
			procEnableWindow.Call(uintptr(hLogBtn), 1)
		}
		setText(hStatusLabel, res.Status)
		if res.ErrText != "" {
			msgBox(hwnd, res.ErrText, res.DialogTitle, MB_OK|MB_ICONERROR)
		} else {
			msgBox(hwnd, res.Message, res.DialogTitle, MB_OK|MB_ICONINFORMATION)
		}
		return 0
	case WM_APP_AUTO_DONE:
		autoMu.Lock()
		res := autoResult
		autoMu.Unlock()
		autoBusy = false
		autoCancelMu.Lock()
		autoCancelCh = nil
		autoCancelMu.Unlock()
		if hAutoBtn != 0 {
			procEnableWindow.Call(uintptr(hAutoBtn), 1)
		}
		if hStopBtn != 0 {
			procEnableWindow.Call(uintptr(hStopBtn), 0)
		}
		if res.OutputDir != "" {
			lastOutputDir = res.OutputDir
			procEnableWindow.Call(uintptr(hOutputBtn), 1)
		}
		if res.LogPath != "" {
			lastLogPath = res.LogPath
			procEnableWindow.Call(uintptr(hLogBtn), 1)
		}
		setText(hStatusLabel, res.Status)
		return 0
	case WM_GETMINMAXINFO:
		if lParam != 0 {
			mmi := (*MINMAXINFO)(unsafe.Pointer(lParam))
			// Keep enough room for the compact responsive layout while still
			// allowing a substantially smaller window than older V0.17 builds.
			mmi.PtMinTrackSize.X = 880
			mmi.PtMinTrackSize.Y = 850
		}
		return 0
	case WM_SIZE:
		cw := int32(loword(lParam))
		ch := int32(hiword(lParam))
		if wParam == SIZE_MAXIMIZED {
			fullscreen = true
		} else if wParam == SIZE_RESTORED {
			fullscreen = false
		}
		layoutMainUI(cw, ch)
		if hFullscreenBtn != 0 {
			if fullscreen {
				setText(hFullscreenBtn, tr("exit_fullscreen"))
			} else {
				setText(hFullscreenBtn, tr("fullscreen"))
			}
		}
		return 0
	case WM_KEYDOWN:
		if wParam == VK_F11 {
			toggleFullscreen()
			return 0
		}
	case WM_CREATE:
		createMainUI(hwnd)
		return 0
	case WM_HSCROLL:
		if hAlphaTrack != 0 && syscall.Handle(lParam) == hAlphaTrack {
			pos, _, _ := procSendMessageW.Call(uintptr(hAlphaTrack), TBM_GETPOS, 0, 0)
			alphaThreshold = clampInt(int(pos), 1, 99)
			updateAlphaLabel()
		}
		return 0
	case WM_COMMAND:
		id := int(loword(wParam))
		notify := int(hiword(wParam))
		if id == ID_COMBO_GAME && notify == CBN_SELCHANGE {
			idx, _, _ := procSendMessageW.Call(uintptr(hGameCombo), CB_GETCURSEL, 0, 0)
			if int(idx) >= 0 && int(idx) < len(gameProfiles) {
				selectedGame = gameProfiles[int(idx)].Key
				appSettings.Game = selectedGame
				appSettings.CS2Root = ""
				appSettings.LastAddon = ""
				selectedAddon = ""
				addonNames = nil
				setText(hRootEdit, "")
				procSendMessageW.Call(uintptr(hAddonCombo), CB_RESETCONTENT, 0, 0)
				updateGameToolsUI()
				setText(hStatusLabel, currentGameName()+" selected. Detecting its installation...")
				saveSettings()
				beginDetectCS2()
			}
			return 0
		}
		if id == ID_COMBO_ADDON && notify == CBN_SELCHANGE {
			selectedAddon = getSelectedAddon()
			saveSettings()
			return 0
		}
		if id == ID_COMBO_CUSTOM_WORKERS && notify == CBN_SELCHANGE {
			syncCustomWorkersFromUI()
			saveSettings()
			return 0
		}
		if id == ID_COMBO_OVERWRITE && notify == CBN_SELCHANGE {
			idx, _, _ := procSendMessageW.Call(uintptr(hOverwriteCombo), CB_GETCURSEL, 0, 0)
			if int(idx) >= 0 && int(idx) <= 2 {
				selectedOverwriteMode = int(idx)
			}
			saveSettings()
			return 0
		}
		if notify == BN_CLICKED && id >= ID_QUALITY_ORIGINAL && id <= ID_QUALITY_LOW {
			selectedQuality = id - ID_QUALITY_ORIGINAL
			setText(hStatusLabel, fmt.Sprintf(tr("quality_set"), qualityLabel(selectedQuality)))
			saveSettings()
			return 0
		}
		if notify == BN_CLICKED && id >= ID_MATERIAL_OPAQUE && id <= ID_MATERIAL_TRANS {
			setMaterialMode(id - ID_MATERIAL_OPAQUE)
			return 0
		}
		if notify == BN_CLICKED && id >= ID_AUTO_SLOW && id <= ID_AUTO_CUSTOM {
			selectedAutoMode = id - ID_AUTO_SLOW
			for i, h := range hAutoModeRadios {
				state := uintptr(0)
				if i == selectedAutoMode {
					state = BST_CHECKED
				}
				procSendMessageW.Call(uintptr(h), BM_SETCHECK, state, 0)
			}
			saveSettings()
			return 0
		}
		if notify == BN_CLICKED && id == ID_CHECK_RETRY {
			st, _, _ := procSendMessageW.Call(uintptr(hRetryCompile), BM_GETCHECK, 0, 0)
			retryCompile = st == BST_CHECKED
			saveSettings()
			return 0
		}
		if notify == BN_CLICKED && id == ID_CHECK_LOCK {
			st, _, _ := procSendMessageW.Call(uintptr(hCompilerLock), BM_GETCHECK, 0, 0)
			compilerLockEnabled = st == BST_CHECKED
			saveSettings()
			return 0
		}
		if notify == BN_CLICKED {
			if code := languageButtonCode(uint32(id)); code != "" {
				currentLang = code
				applyLanguage()
				saveSettings()
				return 0
			}
		}
		switch id {
		case ID_BTN_PNG:
			chooseImageV10(hwnd)
		case ID_BTN_ROOT:
			gp := gameProfileForKey(selectedGame)
			if p := browseFolder(hwnd, "Choose the "+gp.Name+" installation folder"); p != "" {
				setText(hRootEdit, p)
				appSettings.CS2Root = p
				saveSettings()
				beginPopulateAddons(p)
			}
		case ID_BTN_DETECT:
			beginDetectCS2()
		case ID_BTN_REFRESH:
			beginPopulateAddons(strings.TrimSpace(getText(hRootEdit)))
		case ID_BTN_CREATE:
			if selectedGame != "cs2" {
				msgBox(hwnd, currentGameName()+" is selected. This V0.17-based patch keeps the proven VMAT creation pipeline enabled for CS2 only.", appTitle, MB_OK|MB_ICONINFORMATION)
				break
			}
			beginCreateMaterial(hwnd)
		case ID_BTN_OUTPUT:
			if lastOutputDir != "" {
				openFolder(lastOutputDir)
			}
		case ID_BTN_LOG:
			if lastLogPath != "" {
				openFile(lastLogPath)
			}
		case ID_BTN_AUTONOMOUS:
			if selectedGame != "cs2" {
				msgBox(hwnd, currentGameName()+" is selected. Autonomous VMAT production is enabled for CS2 only in this V0.17-based patch.", appTitle, MB_OK|MB_ICONINFORMATION)
				break
			}
			beginAutonomousProduction(hwnd)
		case ID_BTN_STOP:
			requestStopAutonomous()
		case ID_BTN_FULLSCREEN:
			toggleFullscreen()
		case ID_BTN_SETTINGS:
			showSettings(!settingsOpen)
		case ID_BTN_THEME:
			darkMode = !darkMode
			if darkMode {
				setText(hThemeBtn, tr("light"))
			} else {
				setText(hThemeBtn, tr("dark"))
			}
			applyTheme()
			saveSettings()
		case ID_BTN_JUNK_OPEN:
			openFolder(junkDir())
		case ID_BTN_JUNK_CLEAR:
			if msgBox(hwnd, tr("clear_confirm"), appTitle, MB_YESNO|MB_ICONQUESTION) == IDYES {
				count, err := clearJunkFolder()
				if err != nil {
					msgBox(hwnd, "Could not clear Junk folder:\n"+err.Error(), appTitle, MB_OK|MB_ICONERROR)
				} else if count == 0 {
					msgBox(hwnd, tr("junk_empty"), appTitle, MB_OK|MB_ICONINFORMATION)
				} else {
					msgBox(hwnd, tr("junk_cleared"), appTitle, MB_OK|MB_ICONINFORMATION)
				}
			}
		}
		return 0
	case WM_CLOSE:
		saveSettings()
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func createMainUI(hwnd syscall.Handle) {
	installV0175Translations()
	translatedControls = nil
	settingsControls = nil
	rebuildThemeBrushes()
	settingsOpen = true

	// Settings starts open. V0.17 uses one stable owner-drawn Settings control; there is no duplicate title label.
	hSettingsBtn = create3DButtonText(hwnd, tr("settings")+"  ▾", 1010, 18, 122, 38, ID_BTN_SETTINGS)
	hAppTitle = createControl("STATIC", "B.I.T. Texture Tool", WS_CHILD|WS_VISIBLE, 28, 14, 560, 34, hwnd, 0)
	hAppSubtitle = createControl("STATIC", "Texture → VMAT", WS_CHILD|WS_VISIBLE, 28, 48, 650, 20, hwnd, 0)

	// Step 1: helper/status text sits directly beside the heading.
	hStepImage = createTranslatedControl("STATIC", "step_image", WS_CHILD|WS_VISIBLE, 24, 18, 145, 24, hwnd, 0)
	hDimLabel = createControl("STATIC", tr("image_waiting"), WS_CHILD|WS_VISIBLE, 176, 18, 678, 24, hwnd, 0)
	hChooseBtn = create3DButton(hwnd, "choose_image", 24, 45, 165, 34, ID_BTN_PNG)
	hFileLabel = createControl("STATIC", tr("no_image"), WS_CHILD|WS_VISIBLE, 200, 50, 654, 24, hwnd, 0)

	hStepCS2 = createTranslatedControl("STATIC", "step_cs2", WS_CHILD|WS_VISIBLE, 24, 94, 360, 24, hwnd, 0)
	hRootEdit = createControl("EDIT", appSettings.CS2Root, WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL, 24, 120, 440, 26, hwnd, ID_EDIT_ROOT)
	hDetectBtn = create3DButton(hwnd, "detect_cs2", 472, 118, 108, 30, ID_BTN_DETECT)
	hBrowseBtn = create3DButton(hwnd, "browse", 588, 118, 112, 30, ID_BTN_ROOT)
	hAddonLabel = createTranslatedControl("STATIC", "addon", WS_CHILD|WS_VISIBLE, 24, 158, 60, 22, hwnd, 0)
	hAddonCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST, 88, 154, 300, 190, hwnd, ID_COMBO_ADDON)
	hRefreshBtn = create3DButton(hwnd, "refresh", 396, 153, 88, 31, ID_BTN_REFRESH)

	hStepVmat = createTranslatedControl("STATIC", "step_vmat", WS_CHILD|WS_VISIBLE, 24, 198, 350, 22, hwnd, 0)
	hMatEdit = createControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL, 24, 223, 760, 27, hwnd, ID_EDIT_MAT)

	hStepType = createTranslatedControl("STATIC", "step_type", WS_CHILD|WS_VISIBLE, 24, 267, 260, 22, hwnd, 0)
	mDefs := []struct {
		id   int
		key  string
		x, w int32
	}{
		{ID_MATERIAL_OPAQUE, "mat_opaque", 24, 175},
		{ID_MATERIAL_CUTOUT, "mat_cutout", 207, 200},
		{ID_MATERIAL_TRANS, "mat_translucent", 415, 230},
	}
	for i, m := range mDefs {
		style := uint32(WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_AUTORADIOBUTTON | BS_MULTILINE)
		if i == 0 {
			style |= WS_GROUP
		}
		hMaterialRadios[i] = registerTranslated(createControl("BUTTON", tr(m.key), style, m.x, 292, m.w, 26, hwnd, m.id), m.key)
	}
	procSendMessageW.Call(uintptr(hMaterialRadios[selectedMaterialMode]), BM_SETCHECK, BST_CHECKED, 0)
	hAlphaLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE, 24, 326, 180, 22, hwnd, 0)
	hAlphaTrack = createControl("msctls_trackbar32", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 208, 322, 300, 32, hwnd, ID_ALPHA_TRACK)
	procSendMessageW.Call(uintptr(hAlphaTrack), TBM_SETRANGE, 1, makeLPARAM(1, 99))
	procSendMessageW.Call(uintptr(hAlphaTrack), TBM_SETPOS, 1, uintptr(alphaThreshold))
	updateAlphaLabel()
	setMaterialMode(selectedMaterialMode)

	hStepQuality = createTranslatedControl("STATIC", "step_quality", WS_CHILD|WS_VISIBLE, 24, 365, 300, 24, hwnd, 0)
	qDefs := []struct {
		id   int
		key  string
		x, w int32
	}{
		{ID_QUALITY_ORIGINAL, "quality_original", 24, 100}, {ID_QUALITY_HD, "quality_hd", 132, 155},
		{ID_QUALITY_HIGH, "quality_high", 295, 100}, {ID_QUALITY_MID, "quality_mid", 403, 100}, {ID_QUALITY_LOW, "quality_low", 511, 100},
	}
	for i, q := range qDefs {
		style := uint32(WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_AUTORADIOBUTTON | BS_MULTILINE)
		if i == 0 {
			style |= WS_GROUP
		}
		hQualityRadios[i] = registerTranslated(createControl("BUTTON", tr(q.key), style, q.x, 391, q.w, 27, hwnd, q.id), q.key)
	}
	procSendMessageW.Call(uintptr(hQualityRadios[selectedQuality]), BM_SETCHECK, BST_CHECKED, 0)
	hQualityHint = createTranslatedControl("STATIC", "quality_hint", WS_CHILD|WS_VISIBLE, 24, 422, 760, 22, hwnd, 0)

	hCompileCheck = createTranslatedControl("BUTTON", "compile_after", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX|BS_MULTILINE, 24, 449, 500, 28, hwnd, ID_CHECK_COMPILE)
	procSendMessageW.Call(uintptr(hCompileCheck), BM_SETCHECK, BST_CHECKED, 0)

	// Autonomous Production modes. v13 always starts on Normal speed.
	hAutoSpeedLabel = createTranslatedControl("STATIC", "auto_speed", WS_CHILD|WS_VISIBLE, 24, 496, 160, 22, hwnd, 0)
	autoDefs := []struct {
		id  int
		key string
	}{
		{ID_AUTO_SLOW, "auto_slow"}, {ID_AUTO_NORMAL, "auto_normal"}, {ID_AUTO_FAST, "auto_fast"}, {ID_AUTO_EXTREME, "auto_extreme"}, {ID_AUTO_CUSTOM, "auto_custom"},
	}
	for i, a := range autoDefs {
		style := uint32(WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_AUTORADIOBUTTON | BS_MULTILINE)
		if i == 0 {
			style |= WS_GROUP
		}
		hAutoModeRadios[i] = registerTranslated(createControl("BUTTON", tr(a.key), style, 190+int32(i)*132, 492, 124, 28, hwnd, a.id), a.key)
	}
	selectedAutoMode = 1
	procSendMessageW.Call(uintptr(hAutoModeRadios[selectedAutoMode]), BM_SETCHECK, BST_CHECKED, 0)

	hOutputLabel = createTranslatedControl("STATIC", "output", WS_CHILD|WS_VISIBLE, 24, 555, 100, 22, hwnd, 0)
	hStatusLabel = createControl("STATIC", tr("ready"), WS_CHILD|WS_VISIBLE, 24, 578, 830, 30, hwnd, 0)

	hCreateBtn = create3DButton(hwnd, "create_vmat", 24, 650, 145, 44, ID_BTN_CREATE)
	hOutputBtn = create3DButton(hwnd, "open_output", 177, 650, 158, 44, ID_BTN_OUTPUT)
	hLogBtn = create3DButton(hwnd, "open_log", 343, 650, 158, 44, ID_BTN_LOG)
	hAutoBtn = create3DButton(hwnd, "autonomous", 509, 650, 215, 44, ID_BTN_AUTONOMOUS)
	hStopBtn = create3DButton(hwnd, "stop_auto", 732, 650, 88, 44, ID_BTN_STOP)
	procEnableWindow.Call(uintptr(hOutputBtn), 0)
	procEnableWindow.Call(uintptr(hLogBtn), 0)
	procEnableWindow.Call(uintptr(hStopBtn), 0)

	// Right-side Settings drawer. The logo sits directly under the Settings control.
	hSettingsTitle = 0
	hLogo = createControl("STATIC", "", WS_CHILD|WS_VISIBLE|SS_OWNERDRAW, 940, 76, 176, 112, hwnd, ID_LOGO)
	hGameLabel = createTranslatedControl("STATIC", "game", WS_CHILD|WS_VISIBLE, 928, 196, 200, 22, hwnd, 0)
	hGameCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST, 928, 218, 200, 150, hwnd, ID_COMBO_GAME)
	populateGameCombo()
	hLanguageLabel = createTranslatedControl("STATIC", "language", WS_CHILD|WS_VISIBLE, 928, 258, 200, 22, hwnd, 0)
	// Sorted by displayed/native language name for a cleaner predictable list.
	langButtons := []struct {
		id   int
		text string
	}{
		{ID_LANG_CS, "Čeština"}, {ID_LANG_DE, "Deutsch"}, {ID_LANG_EN, "English"},
		{ID_LANG_ES, "Español"}, {ID_LANG_FR, "Français"}, {ID_LANG_PL, "Polski"},
		{ID_LANG_BR, "Português (Brasil)"}, {ID_LANG_RU, "Русский"}, {ID_LANG_TR, "Türkçe"},
	}
	for i, lb := range langButtons {
		hLanguageButtons[i] = create3DButtonText(hwnd, lb.text, 928, 282+int32(i)*31, 200, 27, lb.id)
	}
	hRetryCompile = createTranslatedControl("BUTTON", "retry_compile", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX|BS_MULTILINE, 928, 520, 260, 34, hwnd, ID_CHECK_RETRY)
	hCompilerLock = createTranslatedControl("BUTTON", "compiler_lock", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX|BS_MULTILINE, 928, 555, 260, 34, hwnd, ID_CHECK_LOCK)
	if retryCompile {
		procSendMessageW.Call(uintptr(hRetryCompile), BM_SETCHECK, BST_CHECKED, 0)
	}
	if compilerLockEnabled {
		procSendMessageW.Call(uintptr(hCompilerLock), BM_SETCHECK, BST_CHECKED, 0)
	}
	hCustomWorkersLabel = createTranslatedControl("STATIC", "custom_workers", WS_CHILD|WS_VISIBLE, 928, 553, 260, 20, hwnd, 0)
	hCustomWorkersCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST, 928, 572, 260, 180, hwnd, ID_COMBO_CUSTOM_WORKERS)
	populateCustomWorkersCombo()
	hOverwriteLabel = createTranslatedControl("STATIC", "overwrite", WS_CHILD|WS_VISIBLE, 928, 624, 260, 20, hwnd, 0)
	hOverwriteCombo = createControl("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|CBS_DROPDOWNLIST, 928, 612, 260, 100, hwnd, ID_COMBO_OVERWRITE)
	populateOverwriteCombo()
	hThemeLabel = createTranslatedControl("STATIC", "theme", WS_CHILD|WS_VISIBLE, 928, 620, 200, 22, hwnd, 0)
	themeText := tr("dark")
	if darkMode {
		themeText = tr("light")
	}
	hThemeBtn = create3DButtonText(hwnd, themeText, 928, 642, 200, 32, ID_BTN_THEME)
	hFullscreenBtn = create3DButton(hwnd, "fullscreen", 928, 678, 200, 30, ID_BTN_FULLSCREEN)
	hJunkOpen = create3DButton(hwnd, "open_junk", 928, 712, 200, 30, ID_BTN_JUNK_OPEN)
	hJunkClear = create3DButton(hwnd, "clear_junk", 928, 746, 200, 30, ID_BTN_JUNK_CLEAR)
	hVersionLabel = createTranslatedControl("STATIC", "version", WS_CHILD|WS_VISIBLE|SS_CENTER, 928, 801, 260, 18, hwnd, 0)
	settingsControls = []syscall.Handle{hLogo, hGameLabel, hGameCombo, hLanguageLabel, hRetryCompile, hCompilerLock, hCustomWorkersLabel, hCustomWorkersCombo, hOverwriteLabel, hOverwriteCombo, hThemeLabel, hThemeBtn, hFullscreenBtn, hJunkOpen, hJunkClear, hVersionLabel}
	settingsControls = append(settingsControls, hLanguageButtons[:]...)

	// Typography hierarchy is one of the biggest V0.17 visual changes.
	setControlFont(hAppTitle, hFontTitle)
	setControlFont(hAppSubtitle, hFontSmall)
	for _, h := range []syscall.Handle{hStepImage, hStepCS2, hStepVmat, hStepType, hStepQuality} {
		setControlFont(h, hFontSection)
	}
	// Game, Language, Overwrite mode and Appearance deliberately share the exact
	// same Settings-label typography. Do not give Game/Overwrite a bold treatment.
	for _, h := range []syscall.Handle{hGameLabel, hLanguageLabel, hOverwriteLabel, hThemeLabel, hCustomWorkersLabel} {
		setControlFont(h, hFontSettings)
	}
	for _, h := range []syscall.Handle{hRetryCompile, hCompilerLock} {
		setControlFont(h, hFontBody)
	}
	for _, h := range []syscall.Handle{hOutputLabel, hAutoSpeedLabel, hQualityHint, hDimLabel, hFileLabel} {
		setControlFont(h, hFontSmall)
	}
	setControlFont(hVersionLabel, hFontFooter)
	for _, h := range hMaterialRadios {
		setControlFont(h, hFontBody)
	}
	for _, h := range hQualityRadios {
		setControlFont(h, hFontBody)
	}
	for _, h := range hAutoModeRadios {
		setControlFont(h, hFontBody)
	}
	setControlFont(hCompileCheck, hFontBody)
	for _, h := range []syscall.Handle{hSettingsBtn, hChooseBtn, hDetectBtn, hBrowseBtn, hRefreshBtn, hCreateBtn, hOutputBtn, hLogBtn, hAutoBtn, hStopBtn, hThemeBtn, hFullscreenBtn, hJunkOpen, hJunkClear} {
		setControlFont(h, hFontButton)
	}
	for _, h := range hLanguageButtons {
		setControlFont(h, hFontButton)
	}
	setControlFont(hGameCombo, hFontBody)
	setControlFont(hOverwriteCombo, hFontBody)
	setControlFont(hCustomWorkersCombo, hFontBody)

	updateSettingsButtonText()
	updateGameToolsUI()
	setText(hStatusLabel, tr("ready"))
	var rc RECT
	procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	layoutMainUI(rc.Right-rc.Left, rc.Bottom-rc.Top)
	applyTheme()
}

func updateAlphaLabel() {
	if hAlphaLabel != 0 {
		setText(hAlphaLabel, fmt.Sprintf(tr("alpha_threshold"), alphaThreshold))
	}
}

func materialModeLabel(mode int) string {
	switch mode {
	case 1:
		return tr("mat_cutout")
	case 2:
		return tr("mat_translucent")
	default:
		return tr("mat_opaque")
	}
}

func setMaterialMode(mode int) {
	if mode < 0 || mode > 2 {
		mode = 0
	}
	selectedMaterialMode = mode
	for i, h := range hMaterialRadios {
		if h != 0 {
			state := uintptr(0)
			if i == mode {
				state = BST_CHECKED
			}
			procSendMessageW.Call(uintptr(h), BM_SETCHECK, state, 0)
		}
	}
	if hAlphaTrack != 0 {
		if mode == 1 {
			procEnableWindow.Call(uintptr(hAlphaTrack), 1)
		} else {
			procEnableWindow.Call(uintptr(hAlphaTrack), 0)
		}
	}
	if hAlphaLabel != 0 {
		if mode == 1 {
			procShowWindow.Call(uintptr(hAlphaLabel), SW_SHOW)
			procShowWindow.Call(uintptr(hAlphaTrack), SW_SHOW)
		} else {
			procShowWindow.Call(uintptr(hAlphaLabel), SW_HIDE)
			procShowWindow.Call(uintptr(hAlphaTrack), SW_HIDE)
		}
	}
	if hStatusLabel != 0 {
		setText(hStatusLabel, fmt.Sprintf(tr("material_set"), materialModeLabel(mode)))
	}
	saveSettings()
}

func chooseImageV10(owner syscall.Handle) {
	if imageBusy {
		return
	}
	path := openImageDialog(owner)
	if path == "" {
		return
	}
	imageBusy = true
	if hChooseBtn != 0 {
		procEnableWindow.Call(uintptr(hChooseBtn), 0)
	}
	setText(hStatusLabel, tr("loading_image"))
	appSettings.LastImageDir = filepath.Dir(path)
	saveSettings()
	go loadImageWorker(path)
}

func loadImageWorker(path string) {
	defer func() {
		if r := recover(); r != nil {
			writeCrashLog("loadImageWorker", r)
			imageMu.Lock()
			imageResult = imageLoadResult{err: fmt.Errorf("internal image loader error: %v", r)}
			imageMu.Unlock()
			procPostMessageW.Call(uintptr(hwndMain), WM_APP_IMAGE, 0, 0)
		}
	}()
	img, format, err := decodeSupportedImage(path)
	if err != nil {
		imageMu.Lock()
		imageResult = imageLoadResult{err: err}
		imageMu.Unlock()
		procPostMessageW.Call(uintptr(hwndMain), WM_APP_IMAGE, 0, 0)
		return
	}
	usePath := path
	if strings.ToLower(filepath.Ext(path)) != ".png" {
		usePath = uniqueJunkPNGPath(path, "converted")
		if err := savePNG(usePath, img); err != nil {
			imageMu.Lock()
			imageResult = imageLoadResult{err: fmt.Errorf("automatic PNG conversion failed: %w", err)}
			imageMu.Unlock()
			procPostMessageW.Call(uintptr(hwndMain), WM_APP_IMAGE, 0, 0)
			return
		}
	}
	b := img.Bounds()
	res := imageLoadResult{img: img, path: usePath, originalPath: path, format: format, w: b.Dx(), h: b.Dy(), hasAlpha: imageHasAlpha(img)}
	imageMu.Lock()
	imageResult = res
	imageMu.Unlock()
	procPostMessageW.Call(uintptr(hwndMain), WM_APP_IMAGE, 0, 0)
}

func imageHasAlpha(img image.Image) bool {
	if img == nil {
		return false
	}
	b := img.Bounds()
	// Sample every pixel for modest images and stride across very large ones.
	step := 1
	if b.Dx()*b.Dy() > 16_000_000 {
		step = 2
	}
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return true
			}
		}
	}
	return false
}

func uniqueJunkPNGPath(src, suffix string) string {
	base := sanitizeSegment(strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)))
	if base == "" {
		base = "texture"
	}
	stamp := time.Now().Format("20060102_150405.000")
	return filepath.Join(junkDir(), fmt.Sprintf("%s_%s_%s.png", base, suffix, stamp))
}

func updateImageUIV10(hasAlpha bool) {
	if currentImage == nil {
		setText(hFileLabel, tr("no_image"))
		setText(hDimLabel, tr("image_waiting"))
		return
	}
	displayPath := currentOriginalPath
	if displayPath == "" {
		displayPath = currentPath
	}
	setText(hFileLabel, filepath.Base(displayPath))
	shape := "square / 1:1"
	if currentW != currentH {
		shape = "NOT square — crop required"
	}
	pow := ""
	if currentW == currentH && isPowerOfTwo(currentW) {
		pow = ", power-of-two"
	}
	alpha := tr("alpha_none")
	if hasAlpha {
		alpha = tr("alpha_found")
	}
	setText(hDimLabel, fmt.Sprintf("%d x %d — %s%s — %s", currentW, currentH, shape, pow, alpha))
}

func openImageDialog(owner syscall.Handle) string {
	fileBuf := make([]uint16, 32768)
	filter := u16multi("Supported images (*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.tga)\x00*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.tga\x00PNG (*.png)\x00*.png\x00JPEG (*.jpg;*.jpeg)\x00*.jpg;*.jpeg\x00GIF (*.gif)\x00*.gif\x00BMP (*.bmp)\x00*.bmp\x00TGA (*.tga)\x00*.tga\x00All files (*.*)\x00*.*\x00\x00")
	title := u16("Choose a texture image")
	ofn := OPENFILENAME{
		LStructSize:  uint32(unsafe.Sizeof(OPENFILENAME{})),
		HwndOwner:    owner,
		LpstrFilter:  &filter[0],
		NFilterIndex: 1,
		LpstrFile:    &fileBuf[0],
		NMaxFile:     uint32(len(fileBuf)),
		LpstrTitle:   title,
		Flags:        OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_EXPLORER,
	}
	r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(fileBuf)
}

func decodeSupportedImage(path string) (image.Image, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		img, err := png.Decode(f)
		return img, "PNG", err
	case ".jpg", ".jpeg", ".jpe":
		img, err := jpeg.Decode(f)
		return img, "JPEG", err
	case ".gif":
		img, err := gif.Decode(f)
		return img, "GIF", err
	case ".bmp":
		img, err := decodeBMP(f)
		return img, "BMP", err
	case ".tga":
		img, err := decodeTGA(f)
		return img, "TGA", err
	default:
		// Try signatures for files with a wrong or missing extension.
		head := make([]byte, 12)
		n, _ := io.ReadFull(f, head)
		_, _ = f.Seek(0, io.SeekStart)
		if n >= 8 && string(head[:8]) == "\x89PNG\r\n\x1a\n" {
			img, err := png.Decode(f)
			return img, "PNG", err
		}
		if n >= 2 && head[0] == 0xff && head[1] == 0xd8 {
			img, err := jpeg.Decode(f)
			return img, "JPEG", err
		}
		if n >= 6 && (string(head[:6]) == "GIF87a" || string(head[:6]) == "GIF89a") {
			img, err := gif.Decode(f)
			return img, "GIF", err
		}
		if n >= 2 && string(head[:2]) == "BM" {
			img, err := decodeBMP(f)
			return img, "BMP", err
		}
		return nil, "", fmt.Errorf("unsupported image format")
	}
}

func decodeBMP(r io.ReadSeeker) (image.Image, error) {
	header := make([]byte, 54)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	if string(header[:2]) != "BM" {
		return nil, fmt.Errorf("not a BMP file")
	}
	offset := int64(binary.LittleEndian.Uint32(header[10:14]))
	dib := binary.LittleEndian.Uint32(header[14:18])
	if dib < 40 {
		return nil, fmt.Errorf("unsupported BMP header")
	}
	w := int(int32(binary.LittleEndian.Uint32(header[18:22])))
	rawH := int32(binary.LittleEndian.Uint32(header[22:26]))
	if w <= 0 || rawH == 0 {
		return nil, fmt.Errorf("invalid BMP dimensions")
	}
	h := int(rawH)
	topDown := false
	if h < 0 {
		h = -h
		topDown = true
	}
	bpp := int(binary.LittleEndian.Uint16(header[28:30]))
	compression := binary.LittleEndian.Uint32(header[30:34])
	if compression != 0 {
		return nil, fmt.Errorf("compressed BMP is not supported; save as standard 24/32-bit BMP first")
	}
	if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("only 24-bit and 32-bit BMP are supported")
	}
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	rowBytes := ((w*bpp + 31) / 32) * 4
	row := make([]byte, rowBytes)
	for fileY := 0; fileY < h; fileY++ {
		if _, err := io.ReadFull(r, row); err != nil {
			return nil, err
		}
		y := fileY
		if !topDown {
			y = h - 1 - fileY
		}
		for x := 0; x < w; x++ {
			si := x * (bpp / 8)
			di := y*img.Stride + x*4
			img.Pix[di+0] = row[si+2]
			img.Pix[di+1] = row[si+1]
			img.Pix[di+2] = row[si+0]
			if bpp == 32 {
				img.Pix[di+3] = row[si+3]
			} else {
				img.Pix[di+3] = 255
			}
		}
	}
	return img, nil
}

func decodeTGA(r io.Reader) (image.Image, error) {
	hdr := make([]byte, 18)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}
	idLen := int(hdr[0])
	colorMapType := hdr[1]
	imageType := hdr[2]
	if colorMapType != 0 {
		return nil, fmt.Errorf("color-mapped TGA is not supported")
	}
	w := int(binary.LittleEndian.Uint16(hdr[12:14]))
	h := int(binary.LittleEndian.Uint16(hdr[14:16]))
	bpp := int(hdr[16])
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid TGA dimensions")
	}
	if imageType != 2 && imageType != 3 && imageType != 10 && imageType != 11 {
		return nil, fmt.Errorf("unsupported TGA type %d", imageType)
	}
	if imageType == 3 || imageType == 11 {
		if bpp != 8 {
			return nil, fmt.Errorf("only 8-bit grayscale TGA is supported")
		}
	} else if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("only 24-bit and 32-bit true-color TGA are supported")
	}
	if idLen > 0 {
		if _, err := io.CopyN(io.Discard, r, int64(idLen)); err != nil {
			return nil, err
		}
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	topOrigin := hdr[17]&0x20 != 0
	rightOrigin := hdr[17]&0x10 != 0
	bytesPerPixel := bpp / 8
	pixelCount := w * h
	writePixel := func(index int, px []byte) {
		x := index % w
		y := index / w
		if rightOrigin {
			x = w - 1 - x
		}
		if !topOrigin {
			y = h - 1 - y
		}
		di := y*img.Stride + x*4
		if bytesPerPixel == 1 {
			img.Pix[di+0], img.Pix[di+1], img.Pix[di+2], img.Pix[di+3] = px[0], px[0], px[0], 255
		} else {
			img.Pix[di+0], img.Pix[di+1], img.Pix[di+2] = px[2], px[1], px[0]
			if bytesPerPixel == 4 {
				img.Pix[di+3] = px[3]
			} else {
				img.Pix[di+3] = 255
			}
		}
	}
	if imageType == 2 || imageType == 3 {
		px := make([]byte, bytesPerPixel)
		for i := 0; i < pixelCount; i++ {
			if _, err := io.ReadFull(r, px); err != nil {
				return nil, err
			}
			writePixel(i, px)
		}
		return img, nil
	}
	px := make([]byte, bytesPerPixel)
	idx := 0
	for idx < pixelCount {
		var ph [1]byte
		if _, err := io.ReadFull(r, ph[:]); err != nil {
			return nil, err
		}
		count := int(ph[0]&0x7f) + 1
		if ph[0]&0x80 != 0 {
			if _, err := io.ReadFull(r, px); err != nil {
				return nil, err
			}
			for j := 0; j < count && idx < pixelCount; j++ {
				writePixel(idx, px)
				idx++
			}
		} else {
			for j := 0; j < count && idx < pixelCount; j++ {
				if _, err := io.ReadFull(r, px); err != nil {
					return nil, err
				}
				writePixel(idx, px)
				idx++
			}
		}
	}
	return img, nil
}

func browseFolder(owner syscall.Handle, title string) string {
	display := make([]uint16, 260)
	bi := BROWSEINFO{
		HwndOwner:      owner,
		PszDisplayName: &display[0],
		LpszTitle:      u16(title),
		UlFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE,
	}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)
	out := make([]uint16, 32768)
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&out[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(out)
}

func openCropDialog(owner syscall.Handle) {
	if currentImage == nil || currentW == currentH {
		return
	}
	size := currentW
	if currentH < size {
		size = currentH
	}
	st := &CropState{
		img:        currentImage,
		srcPath:    currentPath,
		w:          currentW,
		h:          currentH,
		size:       size,
		horizontal: currentW > currentH,
	}
	st.offset = abs(currentW-currentH) / 2
	st.pixels = toBGRA(currentImage)
	crop = st

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(u16("CS2MaterialDropperCrop"))),
		uintptr(unsafe.Pointer(u16("Square crop"))),
		uintptr(WS_CAPTION|WS_SYSMENU),
		uintptr(int32(280)), uintptr(int32(150)), uintptr(int32(820)), uintptr(int32(610)),
		uintptr(owner), 0, uintptr(hInstance), 0,
	)
	st.hwnd = syscall.Handle(hwnd)
	if st.hwnd == 0 {
		crop = nil
		return
	}
	procEnableWindow.Call(uintptr(owner), 0)
	procShowWindow.Call(uintptr(st.hwnd), SW_SHOW)
	procUpdateWindow.Call(uintptr(st.hwnd))
	procSetForegroundWindow.Call(uintptr(st.hwnd))

	var m MSG
	for !st.done {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	procEnableWindow.Call(uintptr(owner), 1)
	procSetForegroundWindow.Call(uintptr(owner))
	crop = nil
}

func cropWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			writeCrashLog("cropWndProc", r)
			if crop != nil {
				crop.done = true
			}
			ret = 0
		}
	}()
	st := crop
	switch msg {
	case WM_ERASEBKGND:
		if themeBrushBg == 0 {
			rebuildThemeBrushes()
		}
		var rc RECT
		procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
		procFillRect.Call(wParam, uintptr(unsafe.Pointer(&rc)), uintptr(themeBrushBg))
		return 1
	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis != nil && dis.CtlType == ODT_BUTTON {
			draw3DButton(dis)
			return 1
		}
	case WM_CTLCOLORSTATIC, WM_CTLCOLOREDIT, WM_CTLCOLORBTN, WM_CTLCOLORLISTBOX:
		return controlColor(syscall.Handle(wParam), msg, syscall.Handle(lParam))
	case WM_APP_DETECT:
		detectMu.Lock()
		root := detectResult
		detectBusy = false
		detectMu.Unlock()
		if hDetectBtn != 0 {
			procEnableWindow.Call(uintptr(hDetectBtn), 1)
		}
		if root == "" {
			setText(hStatusLabel, "CS2 was not found automatically. Click Browse and choose your Counter-Strike Global Offensive folder.")
			msgBox(hwnd, "Automatic detection did not find a CS2 Workshop Tools installation.\n\nUse Browse and select the Counter-Strike Global Offensive folder containing both content and game.", appTitle, MB_OK|MB_ICONINFORMATION)
		} else {
			setText(hRootEdit, root)
			beginPopulateAddons(root)
			setText(hStatusLabel, "CS2 detected. Choose an image, addon and VMAT name.")
		}
		return 0
	case WM_APP_STATUS:
		jobMu.Lock()
		st := jobStatus
		jobMu.Unlock()
		if st != "" {
			setText(hStatusLabel, st)
		}
		return 0
	case WM_APP_DONE:
		jobMu.Lock()
		res := jobResult
		jobMu.Unlock()
		createBusy = false
		if hCreateBtn != 0 {
			procEnableWindow.Call(uintptr(hCreateBtn), 1)
		}
		if res.OutputDir != "" {
			lastOutputDir = res.OutputDir
			procEnableWindow.Call(uintptr(hOutputBtn), 1)
		}
		if res.ErrText != "" {
			setText(hStatusLabel, res.Status)
			msgBox(hwnd, res.ErrText, res.DialogTitle, MB_OK|MB_ICONERROR)
		} else {
			setText(hStatusLabel, res.Status)
			msgBox(hwnd, res.Message, res.DialogTitle, MB_OK|MB_ICONINFORMATION)
		}
		return 0
	case WM_CREATE:
		if st == nil {
			return 0
		}
		axis := "Move crop left / right"
		if !st.horizontal {
			axis = "Move crop up / down"
		}
		createControl("STATIC", fmt.Sprintf("Original: %d x %d    Crop: %d x %d", st.w, st.h, st.size, st.size), WS_CHILD|WS_VISIBLE, 24, 430, 470, 22, hwnd, 0)
		createControl("STATIC", axis, WS_CHILD|WS_VISIBLE, 24, 460, 220, 22, hwnd, 0)
		st.track = createControl("msctls_trackbar32", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 24, 485, 520, 34, hwnd, ID_CROP_TRACK)
		maxOffset := abs(st.w - st.h)
		procSendMessageW.Call(uintptr(st.track), TBM_SETRANGE, 1, makeLPARAM(0, int32(maxOffset)))
		procSendMessageW.Call(uintptr(st.track), TBM_SETPOS, 1, uintptr(st.offset))
		create3DButtonText(hwnd, tr("center"), 570, 476, 92, 32, ID_CROP_CENTER)
		create3DButtonText(hwnd, tr("use_crop"), 570, 520, 92, 34, ID_CROP_USE)
		create3DButtonText(hwnd, tr("cancel"), 672, 520, 92, 34, ID_CROP_CANCEL)
		return 0
	case WM_HSCROLL:
		if st != nil && syscall.Handle(lParam) == st.track {
			p, _, _ := procSendMessageW.Call(uintptr(st.track), TBM_GETPOS, 0, 0)
			st.offset = int(p)
			procInvalidateRect.Call(uintptr(hwnd), 0, 1)
		}
		return 0
	case WM_COMMAND:
		if st == nil {
			return 0
		}
		switch int(loword(wParam)) {
		case ID_CROP_CENTER:
			st.offset = abs(st.w-st.h) / 2
			procSendMessageW.Call(uintptr(st.track), TBM_SETPOS, 1, uintptr(st.offset))
			procInvalidateRect.Call(uintptr(hwnd), 0, 1)
		case ID_CROP_USE:
			if err := applyCrop(st); err != nil {
				msgBox(hwnd, "Could not create the square PNG:\n"+err.Error(), appTitle, MB_OK|MB_ICONERROR)
				return 0
			}
			st.accepted = true
			st.done = true
			procDestroyWindow.Call(uintptr(hwnd))
		case ID_CROP_CANCEL:
			st.done = true
			procDestroyWindow.Call(uintptr(hwnd))
		}
		return 0
	case WM_PAINT:
		if st != nil {
			paintCrop(hwnd, st)
		}
		return 0
	case WM_CLOSE:
		if st != nil {
			st.done = true
		}
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case WM_DESTROY:
		if st != nil {
			st.done = true
		}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func paintCrop(hwnd syscall.Handle, st *CropState) {
	var ps PAINTSTRUCT
	hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	areaX, areaY, areaW, areaH := int32(24), int32(20), int32(740), int32(390)
	scaleX := float64(areaW) / float64(st.w)
	scaleY := float64(areaH) / float64(st.h)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	dw := int32(float64(st.w) * scale)
	dh := int32(float64(st.h) * scale)
	dx := areaX + (areaW-dw)/2
	dy := areaY + (areaH-dh)/2

	bmi := BITMAPINFO{BmiHeader: BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		BiWidth:       int32(st.w),
		BiHeight:      -int32(st.h),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0,
		BiSizeImage:   uint32(len(st.pixels)),
	}}
	if len(st.pixels) > 0 {
		procStretchDIBits.Call(
			hdc,
			uintptr(dx), uintptr(dy), uintptr(dw), uintptr(dh),
			0, 0, uintptr(st.w), uintptr(st.h),
			uintptr(unsafe.Pointer(&st.pixels[0])),
			uintptr(unsafe.Pointer(&bmi)),
			DIB_RGB_COLORS, SRCCOPY,
		)
	}

	var x1, y1, x2, y2 int32
	if st.horizontal {
		x1 = dx + int32(float64(st.offset)*scale)
		y1 = dy
		x2 = x1 + int32(float64(st.size)*scale)
		y2 = dy + dh
	} else {
		x1 = dx
		y1 = dy + int32(float64(st.offset)*scale)
		x2 = dx + dw
		y2 = y1 + int32(float64(st.size)*scale)
	}
	pen, _, _ := procCreatePen.Call(PS_SOLID, 3, rgb(255, 70, 70))
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	hollow, _, _ := procGetStockObject.Call(HOLLOW_BRUSH)
	oldBrush, _, _ := procSelectObject.Call(hdc, hollow)
	procRectangle.Call(hdc, uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2))
	procSelectObject.Call(hdc, oldBrush)
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(pen)
}

func applyCrop(st *CropState) error {
	x, y := 0, 0
	if st.horizontal {
		x = st.offset
	} else {
		y = st.offset
	}
	dst := image.NewRGBA(image.Rect(0, 0, st.size, st.size))
	draw.Draw(dst, dst.Bounds(), st.img, image.Point{X: st.img.Bounds().Min.X + x, Y: st.img.Bounds().Min.Y + y}, draw.Src)

	newPath := uniqueJunkPNGPath(st.srcPath, "square")
	f, err := os.Create(newPath)
	if err != nil {
		return err
	}
	err = png.Encode(f, dst)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}

	currentImage = dst
	currentPath = newPath
	currentW, currentH = st.size, st.size
	updateImageUIV10(imageHasAlpha(dst))
	displayBase := currentOriginalPath
	if displayBase == "" {
		displayBase = st.srcPath
	}
	name := strings.TrimSuffix(filepath.Base(displayBase), filepath.Ext(displayBase)) + ".png"
	setText(hStatusLabel, "Crop completed • "+name)
	if !isPowerOfTwo(st.size) {
		msgBox(st.hwnd, fmt.Sprintf("Square crop saved as %d x %d. The size is not a power of two; 512/1024/2048 are typically safer.", st.size, st.size), "Crop created", MB_OK|MB_ICONWARNING)
	}
	return nil
}

func toBGRA(img image.Image) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]byte, w*h*4)
	p := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			out[p+0] = byte(bl >> 8)
			out[p+1] = byte(g >> 8)
			out[p+2] = byte(r >> 8)
			out[p+3] = byte(a >> 8)
			p += 4
		}
	}
	return out
}

type MaterialVariant struct {
	Label string
	Size  int
}

type outputSet struct {
	variant  MaterialVariant
	base     string
	pngPath  string
	vtexPath string
	vmatPath string
}

type createJob struct {
	Root, DestDir, Leaf, RelPrefix string
	Source                         image.Image
	SourceW, SourceH               int
	Sets                           []outputSet
	Compile                        bool
	MaterialMode                   int
	AlphaThreshold                 float64
	RetryCompile                   bool
	CompilerLock                   bool
	OverwriteMode                  int
	Replacing                      bool
}

type autonomousJob struct {
	Root, Addon, Folder string
	Quality             int
	Compile             bool
	MaterialMode        int
	AlphaThreshold      float64
	Mode                int // 0 slow, 1 normal, 2 experimental fast, 3 extreme, 4 custom
	CustomWorkers       int
	Cancel              <-chan struct{}
	RetryCompile        bool
	CompilerLock        bool
	OverwriteMode       int
	CompileGate         chan struct{}
}

type autonomousItem struct {
	Index      int
	Path       string
	Name       string
	Base       string
	SkipReason string
}

type autonomousItemResult struct {
	Index   int
	Success bool
	Skipped bool
	Log     string
}

type createJobResult struct {
	OutputDir, LogPath, Status, DialogTitle, Message, ErrText string
}

func qualitySelection() int {
	if selectedQuality < 0 || selectedQuality > 4 {
		return 0
	}
	return selectedQuality
}

func qualityLabel(q int) string {
	switch q {
	case 1:
		return tr("quality_hd")
	case 2:
		return tr("quality_high")
	case 3:
		return tr("quality_mid")
	case 4:
		return tr("quality_low")
	default:
		return tr("quality_original")
	}
}

func prepareVariants(owner syscall.Handle) ([]MaterialVariant, bool) {
	q := qualitySelection()
	if q == 0 {
		return []MaterialVariant{{Label: "Original", Size: currentW}}, true
	}

	presets := []MaterialVariant{
		{Label: "Highly Detailed", Size: 4096},
		{Label: "High", Size: 2048},
		{Label: "Mid", Size: 1024},
		{Label: "Low", Size: 512},
	}

	// v5 always creates exactly one quality. No filename suffixes are used.
	v := presets[q-1]
	if v.Size > currentW {
		r := msgBox(owner,
			fmt.Sprintf("Your source texture is %d x %d, but you selected %d x %d.\n\nThe app can upscale it, but upscaling cannot create real extra detail. Continue?", currentW, currentH, v.Size, v.Size),
			"Upscaling warning", MB_YESNO|MB_ICONWARNING)
		if r != IDYES {
			return nil, false
		}
	}
	return []MaterialVariant{v}, true
}

func syncOverwriteModeFromUI() int {
	if hOverwriteCombo != 0 {
		idx, _, _ := procSendMessageW.Call(uintptr(hOverwriteCombo), CB_GETCURSEL, 0, 0)
		if int(idx) >= 0 && int(idx) <= 2 {
			selectedOverwriteMode = int(idx)
			appSettings.OverwriteMode = selectedOverwriteMode
			appSettings.CustomWorkers = selectedCustomWorkers
		}
	}
	return selectedOverwriteMode
}

func compiledOutputForSource(root, sourcePath string) string {
	contentRoot := filepath.Join(root, "content")
	rel, err := filepath.Rel(contentRoot, sourcePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return filepath.Join(root, "game", rel) + "_c"
}

func removeIfExists(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

// Compiled resources may already be loaded by Hammer. V0.17.16 never deletes
// .vtex_c/.vmat_c before recompiling, because doing so can make an already
// placed material become the Source 2 error texture. We keep the old compiled
// files in place until Resource Compiler successfully replaces them.
type compiledBackup struct {
	Original string
	Backup   string
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, cpErr := io.Copy(out, in)
	closeErr := out.Close()
	if cpErr != nil {
		return cpErr
	}
	return closeErr
}

func backupCompiledResources(root string, sourcePaths ...string) ([]compiledBackup, string, error) {
	backupDir, err := os.MkdirTemp(junkDir(), "compiled_backup_")
	if err != nil {
		return nil, "", err
	}
	var backups []compiledBackup
	for i, src := range sourcePaths {
		compiled := compiledOutputForSource(root, src)
		if compiled == "" {
			continue
		}
		if _, err := os.Stat(compiled); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			_ = os.RemoveAll(backupDir)
			return nil, "", err
		}
		backup := filepath.Join(backupDir, fmt.Sprintf("%02d_%s", i, filepath.Base(compiled)))
		if err := copyFile(compiled, backup); err != nil {
			_ = os.RemoveAll(backupDir)
			return nil, "", err
		}
		backups = append(backups, compiledBackup{Original: compiled, Backup: backup})
	}
	return backups, backupDir, nil
}

func filesEqual(a, b string) bool {
	aa, errA := os.ReadFile(a)
	bb, errB := os.ReadFile(b)
	if errA != nil || errB != nil {
		return false
	}
	return bytes.Equal(aa, bb)
}

func restoreCompiledResources(backups []compiledBackup) error {
	var errs []string
	for _, b := range backups {
		// If Resource Compiler failed before touching the old compiled file, leave
		// that live Hammer resource completely alone.
		if filesEqual(b.Backup, b.Original) {
			continue
		}
		if err := copyFile(b.Backup, b.Original); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", filepath.Base(b.Original), err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func cleanupCompiledBackup(dir string) {
	if strings.TrimSpace(dir) != "" {
		_ = os.RemoveAll(dir)
	}
}

// Kept as compatibility helpers for older call sites. They intentionally remove
// only editable source files; compiled resources stay alive for Hammer.
func purgeMaterialOutputs(root string, sets []outputSet) error {
	for _, set := range sets {
		for _, p := range []string{set.pngPath, set.vtexPath, set.vmatPath} {
			if err := removeIfExists(p); err != nil {
				return fmt.Errorf("could not replace existing file %s: %w", p, err)
			}
		}
	}
	return nil
}

func purgeAutonomousOutputs(root, pngPath, vtexPath, vmatPath string) error {
	for _, p := range []string{pngPath, vtexPath, vmatPath} {
		if err := removeIfExists(p); err != nil {
			return err
		}
	}
	return nil
}

func beginCreateMaterial(owner syscall.Handle) {
	overwriteMode := syncOverwriteModeFromUI()
	if createBusy || autoBusy {
		msgBox(owner, "A texture job is already running. The window stays usable while it finishes.", appTitle, MB_OK|MB_ICONINFORMATION)
		return
	}
	if currentImage == nil {
		msgBox(owner, "Choose an image first.", appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	if currentW != currentH {
		msgBox(owner, "The selected image is not square. Use Crop to square first.", appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	root := strings.TrimSpace(getText(hRootEdit))
	if !validCS2Root(root) {
		msgBox(owner, "That folder does not look like a CS2 Workshop Tools installation.\n\nChoose the folder that contains both 'content' and 'game'.", appTitle, MB_OK|MB_ICONERROR)
		return
	}
	addon := sanitizeAddon(getSelectedAddon())
	if addon == "" {
		msgBox(owner, "Choose an addon from the list. If your addon is missing, click Refresh.", appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	addonContent := filepath.Join(root, "content", "csgo_addons", addon)
	if st, err := os.Stat(addonContent); err != nil || !st.IsDir() {
		msgBox(owner, "Addon not found in Workshop Tools:\n"+addonContent+"\n\nClick Refresh or open/create the addon in CS2 Workshop Tools first.", appTitle, MB_OK|MB_ICONERROR)
		return
	}

	matName, err := cleanMaterialName(getText(hMatEdit))
	if err != nil {
		msgBox(owner, err.Error(), appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	parts := strings.Split(matName, "/")
	leaf := parts[len(parts)-1]
	relDirParts := parts[:len(parts)-1]
	destDir := filepath.Join(append([]string{addonContent, "materials"}, relDirParts...)...)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		msgBox(owner, "Could not create material folder:\n"+err.Error(), appTitle, MB_OK|MB_ICONERROR)
		return
	}

	variants, ok := prepareVariants(owner)
	if !ok {
		setText(hStatusLabel, "Resize cancelled.")
		return
	}

	relPrefix := "materials"
	if len(relDirParts) > 0 {
		relPrefix += "/" + strings.Join(relDirParts, "/")
	}
	sets := make([]outputSet, 0, len(variants))
	var allPaths []string
	for _, v := range variants {
		base := leaf
		set := outputSet{
			variant:  v,
			base:     base,
			pngPath:  filepath.Join(destDir, base+"_color.png"),
			vtexPath: filepath.Join(destDir, base+"_color.vtex"),
			vmatPath: filepath.Join(destDir, base+".vmat"),
		}
		sets = append(sets, set)
		allPaths = append(allPaths, set.pngPath, set.vtexPath, set.vmatPath)
	}

	replacingExisting := existsAny(allPaths...)
	if replacingExisting {
		switch overwriteMode {
		case 1: // Skip existing
			setText(hStatusLabel, "Skipped: one or more material source files already exist.")
			return
		case 2: // Replace existing
			setText(hStatusLabel, "Replacing existing material safely; keeping the current compiled material live until the new compile succeeds...")
		default: // Ask
			r := msgBox(owner, fmt.Sprintf("One or more output files already exist. Overwrite them?\n\nOutput folder:\n%s", destDir), "Overwrite material?", MB_YESNO|MB_ICONWARNING)
			if r != IDYES {
				return
			}
			// Ask + Yes uses the same Hammer-safe in-place replacement path.
		}
	}

	compileChecked, _, _ := procSendMessageW.Call(uintptr(hCompileCheck), BM_GETCHECK, 0, 0)
	job := createJob{
		Root: root, DestDir: destDir, Leaf: leaf, RelPrefix: relPrefix,
		Source: currentImage, SourceW: currentW, SourceH: currentH, Sets: sets,
		Compile:      compileChecked == BST_CHECKED,
		MaterialMode: selectedMaterialMode, AlphaThreshold: float64(alphaThreshold) / 100.0,
		RetryCompile: retryCompile, CompilerLock: compilerLockEnabled, OverwriteMode: overwriteMode, Replacing: replacingExisting && overwriteMode != 1,
	}
	createBusy = true
	if hCreateBtn != 0 {
		procEnableWindow.Call(uintptr(hCreateBtn), 0)
	}
	setText(hStatusLabel, "Starting background texture job...")
	go runCreateJob(job)
}

func countExistingAutonomousMaterials(root, addon, folder string) (int, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return 0, err
	}
	destDir := filepath.Join(root, "content", "csgo_addons", addon, "materials")
	seen := make(map[string]bool)
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(folder, entry.Name())
		if !isSupportedImageFile(path) {
			continue
		}
		base := sanitizeSegment(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if base == "" {
			continue
		}
		key := strings.ToLower(base)
		if seen[key] {
			continue
		}
		seen[key] = true
		pngPath := filepath.Join(destDir, base+"_color.png")
		vtexPath := filepath.Join(destDir, base+"_color.vtex")
		vmatPath := filepath.Join(destDir, base+".vmat")
		if existsAny(
			pngPath, vtexPath, vmatPath,
			compiledOutputForSource(root, vtexPath),
			compiledOutputForSource(root, vmatPath),
		) {
			count++
		}
	}
	return count, nil
}

func beginAutonomousProduction(owner syscall.Handle) {
	overwriteMode := syncOverwriteModeFromUI()
	customWorkers := syncCustomWorkersFromUI()
	if autoBusy || createBusy {
		msgBox(owner, "A texture job is already running. Wait for it to finish before starting Autonomous Production.", appTitle, MB_OK|MB_ICONINFORMATION)
		return
	}
	root := strings.TrimSpace(getText(hRootEdit))
	if !validCS2Root(root) {
		msgBox(owner, "Choose a valid CS2 Workshop Tools folder first.", appTitle, MB_OK|MB_ICONERROR)
		return
	}
	addon := sanitizeAddon(getSelectedAddon())
	if addon == "" {
		msgBox(owner, "Choose an addon first. Autonomous Production will place all generated materials into that addon.", appTitle, MB_OK|MB_ICONWARNING)
		return
	}
	addonContent := filepath.Join(root, "content", "csgo_addons", addon)
	if st, err := os.Stat(addonContent); err != nil || !st.IsDir() {
		msgBox(owner, "The selected addon folder does not exist. Refresh the addon list and choose it again.", appTitle, MB_OK|MB_ICONERROR)
		return
	}
	folder := browseFolder(owner, "Choose a folder containing texture images for Autonomous Production")
	if strings.TrimSpace(folder) == "" {
		return
	}

	// Ask mode in Autonomous Production should ask once for the whole batch, not
	// silently skip every existing material. This is especially important when
	// rerunning the same source folder with a different material mode.
	if overwriteMode == 0 {
		existingCount, err := countExistingAutonomousMaterials(root, addon, folder)
		if err != nil {
			msgBox(owner, "B.I.T. could not scan the selected folder for existing materials.\n\n"+err.Error(), appTitle, MB_OK|MB_ICONERROR)
			return
		}
		if existingCount > 0 {
			prompt := fmt.Sprintf(
				"B.I.T. found %d existing material(s) from this folder.\n\nCurrent material mode: %s\n\nYES  = Replace all existing materials using the current settings\nNO   = Keep existing materials and skip them\nCANCEL = Do not start the batch",
				existingCount, materialModeLabel(selectedMaterialMode),
			)
			r := msgBox(owner, prompt, "Existing materials found", MB_YESNOCANCEL|MB_ICONQUESTION)
			switch r {
			case IDYES:
				overwriteMode = 2 // Replace all for this batch only.
			case IDNO:
				overwriteMode = 1 // Keep/skip existing for this batch only.
			default:
				return
			}
		}
	}

	compileChecked, _, _ := procSendMessageW.Call(uintptr(hCompileCheck), BM_GETCHECK, 0, 0)
	autoCancelMu.Lock()
	autoCancelCh = make(chan struct{})
	cancelCh := autoCancelCh
	autoCancelMu.Unlock()
	job := autonomousJob{
		Root: root, Addon: addon, Folder: folder,
		Quality: qualitySelection(), Compile: compileChecked == BST_CHECKED,
		MaterialMode: selectedMaterialMode, AlphaThreshold: float64(alphaThreshold) / 100.0, Mode: selectedAutoMode, Cancel: cancelCh,
		RetryCompile: retryCompile, CompilerLock: compilerLockEnabled, OverwriteMode: overwriteMode, CustomWorkers: customWorkers,
	}
	autoBusy = true
	if hAutoBtn != 0 {
		procEnableWindow.Call(uintptr(hAutoBtn), 0)
	}
	if hStopBtn != 0 {
		procEnableWindow.Call(uintptr(hStopBtn), 1)
	}
	setText(hStatusLabel, "Autonomous Production: scanning the selected folder...")
	go runAutonomousProduction(job)
}

func requestStopAutonomous() {
	if !autoBusy {
		return
	}
	autoCancelMu.Lock()
	if autoCancelCh != nil {
		select {
		case <-autoCancelCh:
		default:
			close(autoCancelCh)
		}
	}
	autoCancelMu.Unlock()
	if hStopBtn != 0 {
		procEnableWindow.Call(uintptr(hStopBtn), 0)
	}
	setText(hStatusLabel, tr("stopping"))
}

func finishAutonomous(res createJobResult) {
	autoMu.Lock()
	autoResult = res
	autoMu.Unlock()
	procPostMessageW.Call(uintptr(hwndMain), WM_APP_AUTO_DONE, 0, 0)
}

func isSupportedImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tga":
		return true
	}
	return false
}

func centerCropSquare(src image.Image) image.Image {
	if src == nil {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	side := w
	if h < side {
		side = h
	}
	if side <= 0 || (w == side && h == side) {
		return src
	}
	x0 := b.Min.X + (w-side)/2
	y0 := b.Min.Y + (h-side)/2
	out := image.NewNRGBA(image.Rect(0, 0, side, side))
	draw.Draw(out, out.Bounds(), src, image.Pt(x0, y0), draw.Src)
	return out
}

func targetSizeForQuality(q, original int) int {
	switch q {
	case 1:
		return 4096
	case 2:
		return 2048
	case 3:
		return 1024
	case 4:
		return 512
	default:
		return original
	}
}

func autonomousModeLabel(mode int) string {
	switch mode {
	case 0:
		return tr("auto_slow")
	case 2:
		return tr("auto_fast")
	case 3:
		return tr("auto_extreme")
	case 4:
		return tr("auto_custom")
	default:
		return tr("auto_normal")
	}
}

// resolutionClass is the effective output-size class used to tune batch
// concurrency. 512 means low/small textures, 4096 means high-memory textures.
// For a fixed resize quality this is known immediately. For Original quality,
// runAutonomousProduction detects it from the source folder before workers start.
func autonomousWorkerCount(mode, resolutionClass, customWorkers int) int {
	cpu := runtime.NumCPU()
	if cpu < 1 {
		cpu = 1
	}

	// Custom mode gives advanced users explicit control. Unlike Fast/Extreme it
	// does not silently reduce the chosen worker count based on resolution.
	if mode == 4 {
		return clampCustomWorkers(customWorkers)
	}
	// Slow and Normal intentionally remain stable reference modes. Adaptive
	// concurrency is used by Fast/Extreme, where it provides the biggest gain.
	if mode == 0 || mode == 1 {
		return 1
	}

	if mode == 2 { // Experimental Fast
		n := cpu / 2
		if n < 4 {
			n = 4
		}
		capN := 8
		switch {
		case resolutionClass >= 4096:
			capN = 2
		case resolutionClass >= 2048:
			capN = 4
		case resolutionClass >= 1024:
			capN = 6
		default: // <= 512
			capN = 8
		}
		if n > capN {
			n = capN
		}
		return n
	}

	// Experimental Extreme: low-resolution batches can use substantially more
	// workers, while 2K/4K jobs are capped to avoid large RAM spikes.
	n := cpu
	if n < 8 {
		n = 8
	}
	capN := 16
	switch {
	case resolutionClass >= 4096:
		capN = 4
	case resolutionClass >= 2048:
		capN = 8
	case resolutionClass >= 1024:
		capN = 12
	default: // <= 512
		capN = 16
	}
	if n > capN {
		n = capN
	}
	return n
}

func autonomousCompilerSlots(mode, resolutionClass, workers int) int {
	// Resource Compiler concurrency also scales with texture size, but stays
	// lower than image-worker concurrency. Any shared-file collision is still
	// retried under the exclusive compiler lock when locking/retry are enabled.
	cpu := runtime.NumCPU()
	if cpu < 1 {
		cpu = 1
	}
	switch mode {
	case 4: // Custom: derive compiler concurrency conservatively from chosen workers.
		n := workers / 4
		if n < 1 {
			n = 1
		}
		if n > 6 {
			n = 6
		}
		return n
	case 2: // Fast
		if resolutionClass >= 4096 {
			return 1
		}
		if resolutionClass >= 2048 {
			return 2
		}
		n := 3
		if cpu < n {
			n = cpu
		}
		if n < 1 {
			n = 1
		}
		return n
	case 3: // Extreme
		var capN int
		switch {
		case resolutionClass >= 4096:
			capN = 2
		case resolutionClass >= 2048:
			capN = 3
		case resolutionClass >= 1024:
			capN = 4
		default:
			capN = 6
		}
		n := cpu / 2
		if n < 2 {
			n = 2
		}
		if n > capN {
			n = capN
		}
		return n
	default:
		return 1
	}
}

// fastImageDimensions reads only the image header where possible. It avoids
// decoding full images just to choose a safe worker count for Original quality.
func fastImageDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		cfg, err := png.DecodeConfig(f)
		if err != nil {
			return 0, 0, err
		}
		return cfg.Width, cfg.Height, nil
	case ".jpg", ".jpeg", ".jpe":
		cfg, err := jpeg.DecodeConfig(f)
		if err != nil {
			return 0, 0, err
		}
		return cfg.Width, cfg.Height, nil
	case ".gif":
		cfg, err := gif.DecodeConfig(f)
		if err != nil {
			return 0, 0, err
		}
		return cfg.Width, cfg.Height, nil
	case ".bmp":
		h := make([]byte, 26)
		if _, err := io.ReadFull(f, h); err != nil {
			return 0, 0, err
		}
		if string(h[:2]) != "BM" {
			return 0, 0, fmt.Errorf("invalid BMP")
		}
		w := int(int32(binary.LittleEndian.Uint32(h[18:22])))
		hh := int(int32(binary.LittleEndian.Uint32(h[22:26])))
		if hh < 0 {
			hh = -hh
		}
		if w < 0 {
			w = -w
		}
		return w, hh, nil
	case ".tga":
		h := make([]byte, 18)
		if _, err := io.ReadFull(f, h); err != nil {
			return 0, 0, err
		}
		return int(binary.LittleEndian.Uint16(h[12:14])), int(binary.LittleEndian.Uint16(h[14:16])), nil
	default:
		return 0, 0, fmt.Errorf("unsupported image format")
	}
}

func detectedOriginalResolutionClass(files []string) int {
	// Use the 75th percentile rather than the absolute maximum. This makes a
	// mostly-small texture pack fast without letting a single large outlier set
	// the concurrency for the entire batch.
	var dims []int
	for _, path := range files {
		w, h, err := fastImageDimensions(path)
		if err != nil || w <= 0 || h <= 0 {
			continue
		}
		d := w
		if h > d {
			d = h
		}
		dims = append(dims, d)
	}
	if len(dims) == 0 {
		return 1024
	} // conservative fallback
	sort.Ints(dims)
	idx := (len(dims) * 3) / 4
	if idx >= len(dims) {
		idx = len(dims) - 1
	}
	d := dims[idx]
	switch {
	case d <= 512:
		return 512
	case d <= 1024:
		return 1024
	case d <= 2048:
		return 2048
	default:
		return 4096
	}
}

func effectiveAutonomousResolutionClass(quality int, files []string) int {
	switch quality {
	case 1:
		return 4096
	case 2:
		return 2048
	case 3:
		return 1024
	case 4:
		return 512
	default:
		return detectedOriginalResolutionClass(files)
	}
}

func compactDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int64(d.Round(time.Second) / time.Second)
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func autonomousProgressLine(done, total int, started time.Time) string {
	elapsed := time.Since(started)
	if done <= 0 || elapsed < 150*time.Millisecond {
		return fmt.Sprintf("Processing %d / %d • calculating speed • %s elapsed • calculating remaining", done, total, compactDuration(elapsed))
	}
	rate := float64(done) / elapsed.Seconds()
	if rate <= 0 {
		return fmt.Sprintf("Processing %d / %d • calculating speed • %s elapsed • calculating remaining", done, total, compactDuration(elapsed))
	}
	remain := time.Duration(float64(total-done) / rate * float64(time.Second))
	return fmt.Sprintf("Processing %d / %d • %.1f textures/sec • %s elapsed • ~%s remaining", done, total, rate, compactDuration(elapsed), compactDuration(remain))
}

func processAutonomousItem(job autonomousJob, item autonomousItem, destDir string) autonomousItemResult {
	var log strings.Builder
	prefix := fmt.Sprintf("[%d] ", item.Index+1)
	if item.SkipReason != "" {
		fmt.Fprintf(&log, "%sSKIP %s - %s\r\n\r\n", prefix, item.Name, item.SkipReason)
		return autonomousItemResult{Index: item.Index, Skipped: true, Log: log.String()}
	}

	pngPath := filepath.Join(destDir, item.Base+"_color.png")
	vtexPath := filepath.Join(destDir, item.Base+"_color.vtex")
	vmatPath := filepath.Join(destDir, item.Base+".vmat")
	if existsAny(pngPath, vtexPath, vmatPath) && job.OverwriteMode != 2 {
		modeText := "Skip existing"
		if job.OverwriteMode == 0 {
			modeText = "Ask mode (no batch replacement choice was required)"
		}
		fmt.Fprintf(&log, "%sSKIP %s - output already exists [%s]\r\n\r\n", prefix, item.Name, modeText)
		return autonomousItemResult{Index: item.Index, Skipped: true, Log: log.String()}
	}
	replacingExisting := existsAny(pngPath, vtexPath, vmatPath, compiledOutputForSource(job.Root, vtexPath), compiledOutputForSource(job.Root, vmatPath)) && job.OverwriteMode == 2
	if replacingExisting {
		fmt.Fprintf(&log, "%sREPLACE existing material safely (compiled resource remains live until successful rebuild)\r\n", prefix)
	}

	var compiledBackups []compiledBackup
	var compiledBackupDir string
	if replacingExisting && job.Compile {
		var backupErr error
		compiledBackups, compiledBackupDir, backupErr = backupCompiledResources(job.Root, vtexPath, vmatPath)
		if backupErr != nil {
			fmt.Fprintf(&log, "%sFAIL %s - could not create Hammer-safe compiled backup: %v\r\n\r\n", prefix, item.Name, backupErr)
			return autonomousItemResult{Index: item.Index, Success: false, Log: log.String()}
		}
		defer cleanupCompiledBackup(compiledBackupDir)
	}
	restoreOnFailure := func() {
		if len(compiledBackups) == 0 {
			return
		}
		if err := restoreCompiledResources(compiledBackups); err != nil {
			fmt.Fprintf(&log, "ROLLBACK WARNING: could not fully restore previous compiled material: %v\r\n", err)
		} else {
			log.WriteString("ROLLBACK: previous compiled material restored for Hammer.\r\n")
		}
	}

	img, format, err := decodeSupportedImage(item.Path)
	if err != nil {
		fmt.Fprintf(&log, "%sFAIL %s - decode error: %v\r\n\r\n", prefix, item.Name, err)
		return autonomousItemResult{Index: item.Index, Success: false, Log: log.String()}
	}
	b := img.Bounds()
	origW, origH := b.Dx(), b.Dy()
	work := centerCropSquare(img)
	img = nil
	wb := work.Bounds()
	side := wb.Dx()
	target := targetSizeForQuality(job.Quality, side)
	if target != side {
		work = resizeFast(work, target, target)
	}

	if err := savePNG(pngPath, work); err != nil {
		fmt.Fprintf(&log, "%sFAIL %s - PNG save error: %v\r\n\r\n", prefix, item.Name, err)
		return autonomousItemResult{Index: item.Index, Success: false, Log: log.String()}
	}
	work = nil

	pngResource := "materials/" + filepath.Base(pngPath)
	vtexResource := "materials/" + strings.TrimSuffix(filepath.Base(vtexPath), ".vtex") + ".vtex_c"
	if err := os.WriteFile(vtexPath, []byte(makeVTEX(pngResource)), 0644); err != nil {
		fmt.Fprintf(&log, "%sFAIL %s - VTEX write error: %v\r\n\r\n", prefix, item.Name, err)
		return autonomousItemResult{Index: item.Index, Success: false, Log: log.String()}
	}
	if err := os.WriteFile(vmatPath, []byte(makeVMAT(job.MaterialMode, job.AlphaThreshold, vtexResource)), 0644); err != nil {
		fmt.Fprintf(&log, "%sFAIL %s - VMAT write error: %v\r\n\r\n", prefix, item.Name, err)
		return autonomousItemResult{Index: item.Index, Success: false, Log: log.String()}
	}

	fmt.Fprintf(&log, "%s%s -> %s.vmat | %s | %dx%d -> %dx%d", prefix, item.Name, item.Base, format, origW, origH, target, target)
	if origW != origH {
		log.WriteString(" | center-cropped to square")
	}
	log.WriteString("\r\n")

	if job.Compile {
		okTex, outTex := compileWithCS2Autonomous(job.Root, job.Mode, job.CompilerLock, job.RetryCompile, job.CompileGate, vtexPath)
		log.WriteString(outTex)
		if !okTex {
			restoreOnFailure()
			fmt.Fprintf(&log, "RESULT: FAILED compiling texture for %s\r\n\r\n", item.Name)
			return autonomousItemResult{Index: item.Index, Success: false, Log: log.String()}
		}
		okMat, outMat := compileWithCS2Autonomous(job.Root, job.Mode, job.CompilerLock, job.RetryCompile, job.CompileGate, vmatPath)
		log.WriteString(outMat)
		if !okMat {
			restoreOnFailure()
			fmt.Fprintf(&log, "RESULT: FAILED compiling VMAT for %s\r\n\r\n", item.Name)
			return autonomousItemResult{Index: item.Index, Success: false, Log: log.String()}
		}
	}
	fmt.Fprintf(&log, "RESULT: OK - %s.vmat\r\n\r\n", item.Base)
	return autonomousItemResult{Index: item.Index, Success: true, Log: log.String()}
}

func runAutonomousProduction(job autonomousJob) {
	defer func() {
		if r := recover(); r != nil {
			writeCrashLog("runAutonomousProduction", r)
			finishAutonomous(createJobResult{Status: "Autonomous Production stopped because of an internal error. See the crash log."})
		}
	}()

	entries, err := os.ReadDir(job.Folder)
	if err != nil {
		finishAutonomous(createJobResult{Status: "Autonomous Production could not read the selected folder: " + err.Error()})
		return
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(job.Folder, entry.Name())
		if isSupportedImageFile(path) {
			files = append(files, path)
		}
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i]) < strings.ToLower(files[j]) })
	if len(files) == 0 {
		finishAutonomous(createJobResult{Status: "Autonomous Production found no supported images in the selected folder."})
		return
	}

	destDir := filepath.Join(job.Root, "content", "csgo_addons", job.Addon, "materials")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		finishAutonomous(createJobResult{Status: "Autonomous Production could not create the addon materials folder: " + err.Error()})
		return
	}
	stamp := time.Now().Format("20060102_150405")
	logPath := filepath.Join(destDir, "autonomous_production_"+stamp+"_compile_log.txt")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		finishAutonomous(createJobResult{OutputDir: destDir, Status: "Autonomous Production could not create its log file: " + err.Error()})
		return
	}
	defer lf.Close()
	resolutionClass := effectiveAutonomousResolutionClass(job.Quality, files)
	workers := autonomousWorkerCount(job.Mode, resolutionClass, job.CustomWorkers)
	compileSlots := autonomousCompilerSlots(job.Mode, resolutionClass, workers)
	job.CompileGate = make(chan struct{}, compileSlots)
	fmt.Fprintf(lf, "B.I.T. Texture Tool V0.17.16 - Autonomous Production\r\nSource folder: %s\r\nAddon: %s\r\nQuality: %s\r\nMaterial mode: %s\r\nProcessing mode: %s\r\nAdaptive resolution class: %d px\r\nWorkers: %d\r\nCompiler slots: %d\r\nFiles found: %d\r\nRetry failed compile: %t\r\nCompiler locking: %t\r\nOverwrite mode: %s\r\n\r\n", job.Folder, job.Addon, qualityLabel(job.Quality), materialModeLabel(job.MaterialMode), autonomousModeLabel(job.Mode), resolutionClass, workers, compileSlots, len(files), job.RetryCompile, job.CompilerLock, overwriteModeLabel(job.OverwriteMode))
	postJobStatus(fmt.Sprintf("%s • %d px class • %d workers • %d compiler slot(s) • scanning...", autonomousModeLabel(job.Mode), resolutionClass, workers, compileSlots))

	if job.Compile {
		rc := filepath.Join(job.Root, "game", "bin", "win64", "resourcecompiler.exe")
		if _, err := os.Stat(rc); err != nil {
			fmt.Fprintf(lf, "ERROR: Resource Compiler not found: %s\r\n", rc)
			finishAutonomous(createJobResult{OutputDir: destDir, LogPath: logPath, Status: "Autonomous Production stopped: Resource Compiler was not found. Open compile log for details."})
			return
		}
	}

	// Pre-resolve output names so worker jobs can never collide with each other.
	items := make([]autonomousItem, 0, len(files))
	usedNames := make(map[string]bool)
	for i, path := range files {
		name := filepath.Base(path)
		base := sanitizeSegment(strings.TrimSuffix(name, filepath.Ext(name)))
		item := autonomousItem{Index: i, Path: path, Name: name, Base: base}
		if base == "" {
			item.SkipReason = "filename cannot be converted to a valid VMAT name"
		} else {
			key := strings.ToLower(base)
			if usedNames[key] {
				item.SkipReason = "another image in this folder maps to the same VMAT name " + base
			} else {
				usedNames[key] = true
			}
		}
		items = append(items, item)
	}

	started := time.Now()
	postJobStatus(autonomousProgressLine(0, len(items), started))
	jobsCh := make(chan autonomousItem)
	resultsCh := make(chan autonomousItemResult, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobsCh {
				select {
				case <-job.Cancel:
					return
				default:
				}
				res := processAutonomousItem(job, item, destDir)
				resultsCh <- res
				if job.Mode == 0 {
					// Experimental Slow deliberately yields CPU/disk time to other apps.
					time.Sleep(1100 * time.Millisecond)
					runtime.Gosched()
				}
			}
		}()
	}
	go func() {
		defer close(jobsCh)
		for _, item := range items {
			select {
			case <-job.Cancel:
				return
			case jobsCh <- item:
			}
		}
	}()
	go func() { wg.Wait(); close(resultsCh) }()

	done, success, skipped, failed := 0, 0, 0, 0
	for res := range resultsCh {
		done++
		if res.Success {
			success++
		} else if res.Skipped {
			skipped++
		} else {
			failed++
		}
		_, _ = lf.WriteString(res.Log)
		if done%10 == 0 {
			_ = lf.Sync()
		}
		postJobStatus(autonomousProgressLine(done, len(items), started))
	}
	elapsed := time.Since(started)
	rate := 0.0
	if elapsed.Seconds() > 0 {
		rate = float64(done) / elapsed.Seconds()
	}
	cancelled := false
	select {
	case <-job.Cancel:
		cancelled = true
	default:
	}
	if cancelled {
		fmt.Fprintf(lf, "----------------------------------------\r\nSTOPPED BY USER. Success: %d | Skipped: %d | Failed: %d | Processed: %d/%d | Average: %.2f textures/sec | Time: %s\r\n", success, skipped, failed, done, len(items), rate, compactDuration(elapsed))
	} else {
		fmt.Fprintf(lf, "----------------------------------------\r\nFinished. Success: %d | Skipped: %d | Failed: %d | Total: %d | Average: %.2f textures/sec | Time: %s\r\n", success, skipped, failed, len(items), rate, compactDuration(elapsed))
	}
	_ = lf.Sync()
	status := fmt.Sprintf("Completed %d / %d • %.1f textures/sec avg • %s total • %d success • %d skipped • %d failed", done, len(items), rate, compactDuration(elapsed), success, skipped, failed)
	if cancelled {
		status = fmt.Sprintf("Stopped %d / %d • %.1f textures/sec • %s elapsed • %d success • %d skipped • %d failed", done, len(items), rate, compactDuration(elapsed), success, skipped, failed)
	}
	finishAutonomous(createJobResult{OutputDir: destDir, LogPath: logPath, Status: status})
}

func postJobStatus(text string) {
	jobMu.Lock()
	jobStatus = text
	jobMu.Unlock()
	procPostMessageW.Call(uintptr(hwndMain), WM_APP_STATUS, 0, 0)
}

func finishJob(res createJobResult) {
	jobMu.Lock()
	jobResult = res
	jobMu.Unlock()
	procPostMessageW.Call(uintptr(hwndMain), WM_APP_DONE, 0, 0)
}

func runCreateJob(job createJob) {
	defer func() {
		if r := recover(); r != nil {
			writeCrashLog("runCreateJob", r)
			finishJob(createJobResult{
				Status: "Background resize failed.", DialogTitle: "Resize error",
				ErrText: fmt.Sprintf("The background resize job hit an internal error: %v\n\nA crash log was written next to the EXE.", r),
			})
		}
	}()
	var compilePaths []string
	var compiledBackups []compiledBackup
	var compiledBackupDir string
	if job.Replacing && job.Compile {
		var sourcePaths []string
		for _, set := range job.Sets {
			sourcePaths = append(sourcePaths, set.vtexPath, set.vmatPath)
		}
		var err error
		compiledBackups, compiledBackupDir, err = backupCompiledResources(job.Root, sourcePaths...)
		if err != nil {
			finishJob(createJobResult{Status: "Could not prepare safe replacement.", DialogTitle: "Overwrite failed", ErrText: "B.I.T. could not back up the currently compiled material before replacement:\n" + err.Error()})
			return
		}
		defer cleanupCompiledBackup(compiledBackupDir)
	}
	for idx, set := range job.Sets {
		postJobStatus(fmt.Sprintf("Resizing %s to %d x %d in background — %d of %d...", set.variant.Label, set.variant.Size, set.variant.Size, idx+1, len(job.Sets)))
		imgOut := job.Source
		if set.variant.Size != job.SourceW || set.variant.Size != job.SourceH {
			imgOut = resizeFast(job.Source, set.variant.Size, set.variant.Size)
		}
		postJobStatus(fmt.Sprintf("Saving %d x %d PNG...", set.variant.Size, set.variant.Size))
		if err := savePNG(set.pngPath, imgOut); err != nil {
			finishJob(createJobResult{Status: "Could not save texture.", DialogTitle: appTitle, ErrText: "Could not save texture:\n" + err.Error()})
			return
		}
		imgOut = nil
		runtime.GC()
		pngResource := job.RelPrefix + "/" + filepath.Base(set.pngPath)
		// VMAT texture parameters must reference the compiled texture resource (.vtex_c),
		// not the source .vtex descriptor. Resource Compiler writes the .vtex_c into
		// game\csgo_addons\<addon> at the matching resource path.
		vtexResource := job.RelPrefix + "/" + strings.TrimSuffix(filepath.Base(set.vtexPath), ".vtex") + ".vtex_c"
		if err := os.WriteFile(set.vtexPath, []byte(makeVTEX(pngResource)), 0644); err != nil {
			finishJob(createJobResult{Status: "Could not write VTEX.", DialogTitle: appTitle, ErrText: "Could not write VTEX:\n" + err.Error()})
			return
		}
		if err := os.WriteFile(set.vmatPath, []byte(makeVMAT(job.MaterialMode, job.AlphaThreshold, vtexResource)), 0644); err != nil {
			finishJob(createJobResult{Status: "Could not write VMAT.", DialogTitle: appTitle, ErrText: "Could not write VMAT:\n" + err.Error()})
			return
		}
		compilePaths = append(compilePaths, set.vtexPath, set.vmatPath)
	}

	if job.Compile {
		postJobStatus("Material created. Compiling with CS2 Workshop Tools in background...")
		ok, output := compileWithCS2Safe(job.Root, job.CompilerLock, job.RetryCompile, compilePaths...)
		logPath := filepath.Join(job.DestDir, job.Leaf+"_compile_log.txt")
		_ = os.WriteFile(logPath, []byte(output), 0644)
		if !ok {
			rollbackNote := ""
			if len(compiledBackups) > 0 {
				if err := restoreCompiledResources(compiledBackups); err != nil {
					rollbackNote = "\n\nWarning: B.I.T. could not fully restore the previous compiled material: " + err.Error()
				} else {
					rollbackNote = "\n\nThe previous compiled material was restored, so surfaces already using it in Hammer should stay valid."
				}
			}
			finishJob(createJobResult{
				OutputDir: job.DestDir, LogPath: logPath, Status: "Replacement compile failed; previous compiled material preserved when possible. See compile log.",
				DialogTitle: "CS2 compile error",
				ErrText:     "The PNG/VTEX/VMAT source files were updated, but CS2 Resource Compiler returned an error.\n\nLog:\n" + logPath + rollbackNote,
			})
			return
		}
		finishJob(createJobResult{OutputDir: job.DestDir, LogPath: logPath, Status: "Done. Material compiled successfully; Hammer-safe replacement completed.", DialogTitle: "Material created", Message: fmt.Sprintf("Created and compiled the material.\n\nOutput folder:\n%s\n\nIt should now appear in Hammer's Asset Browser.", job.DestDir)})
		return
	}
	finishJob(createJobResult{OutputDir: job.DestDir, Status: "Done. Material source files created; compilation was skipped.", DialogTitle: "Material created", Message: fmt.Sprintf("Created the material source files.\n\nOutput folder:\n%s\n\nOpen CS2 Workshop Tools to compile/use them.", job.DestDir)})
}

func overwriteModeLabel(mode int) string {
	switch mode {
	case 1:
		return tr("overwrite_skip")
	case 2:
		return tr("overwrite_replace")
	default:
		return tr("overwrite_ask")
	}
}

func runResourceCompilerOnce(rc, p string) ([]byte, error) {
	cmd := exec.Command(rc, "-i", p, "-f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}

func compileWithCS2Safe(root string, lockShared, retry bool, paths ...string) (bool, string) {
	rc := filepath.Join(root, "game", "bin", "win64", "resourcecompiler.exe")
	if _, err := os.Stat(rc); err != nil {
		return false, "resourcecompiler.exe was not found at:\r\n" + rc + "\r\n\r\nMake sure CS2 Workshop Tools are installed."
	}
	var all strings.Builder
	for _, p := range paths {
		// VMAT compilation may touch shared default normal/AO resources. Only
		// serialize that phase; unique color VTEX files can still compile in parallel.
		locked := lockShared && strings.EqualFold(filepath.Ext(p), ".vmat")
		if locked {
			sharedVMATCompileMu.Lock()
		}
		out, err := runResourceCompilerOnce(rc, p)
		all.WriteString("=== " + filepath.Base(p) + " ===\r\n")
		all.Write(out)
		if len(out) == 0 {
			all.WriteString("(no console output)\r\n")
		}
		if err != nil && retry {
			all.WriteString("\r\n--- Automatic retry 1/1 after compiler error ---\r\n")
			time.Sleep(180 * time.Millisecond)
			out2, err2 := runResourceCompilerOnce(rc, p)
			all.Write(out2)
			if len(out2) == 0 {
				all.WriteString("(no console output on retry)\r\n")
			}
			if err2 == nil {
				all.WriteString("\r\nRETRY RESULT: OK\r\n")
				err = nil
			} else {
				all.WriteString("\r\nRETRY ERROR: " + err2.Error() + "\r\n")
				err = err2
			}
		}
		if locked {
			sharedVMATCompileMu.Unlock()
		}
		if err != nil {
			all.WriteString("\r\nERROR: " + err.Error() + "\r\n")
			return false, all.String()
		}
		all.WriteString("\r\n")
	}
	return true, all.String()
}

func compileWithCS2Autonomous(root string, mode int, lockShared, retry bool, gate chan struct{}, p string) (bool, string) {
	rc := filepath.Join(root, "game", "bin", "win64", "resourcecompiler.exe")
	if _, err := os.Stat(rc); err != nil {
		return false, "resourcecompiler.exe was not found at:\r\n" + rc + "\r\n\r\nMake sure CS2 Workshop Tools are installed."
	}

	var all strings.Builder
	all.WriteString("=== " + filepath.Base(p) + " ===\r\n")
	isVMAT := strings.EqualFold(filepath.Ext(p), ".vmat")

	// Unique VTEX files are always safe to compile in parallel. VMAT files may
	// touch shared default AO/normal resources, so their concurrency depends on
	// the selected autonomous profile.
	if gate != nil && isVMAT && mode >= 2 {
		gate <- struct{}{}
		defer func() { <-gate }()
	}

	if !isVMAT || !lockShared {
		out, err := runResourceCompilerOnce(rc, p)
		all.Write(out)
		if len(out) == 0 {
			all.WriteString("(no console output)\r\n")
		}
		if err != nil && retry {
			all.WriteString("\r\n--- Automatic retry 1/1 after compiler error ---\r\n")
			time.Sleep(180 * time.Millisecond)
			out2, err2 := runResourceCompilerOnce(rc, p)
			all.Write(out2)
			if len(out2) == 0 {
				all.WriteString("(no console output on retry)\r\n")
			}
			if err2 == nil {
				all.WriteString("\r\nRETRY RESULT: OK\r\n")
				err = nil
			} else {
				err = err2
			}
		}
		if err != nil {
			all.WriteString("\r\nERROR: " + err.Error() + "\r\n")
			return false, all.String()
		}
		all.WriteString("\r\n")
		return true, all.String()
	}

	// Slow and Normal use a true exclusive compile lock for maximum stability.
	if mode <= 1 {
		sharedVMATCompileMu.Lock()
		out, err := runResourceCompilerOnce(rc, p)
		all.Write(out)
		if len(out) == 0 {
			all.WriteString("(no console output)\r\n")
		}
		if err != nil && retry {
			all.WriteString("\r\n--- Automatic retry 1/1 under compiler lock ---\r\n")
			time.Sleep(180 * time.Millisecond)
			out2, err2 := runResourceCompilerOnce(rc, p)
			all.Write(out2)
			if len(out2) == 0 {
				all.WriteString("(no console output on retry)\r\n")
			}
			if err2 == nil {
				all.WriteString("\r\nRETRY RESULT: OK\r\n")
				err = nil
			} else {
				err = err2
			}
		}
		sharedVMATCompileMu.Unlock()
		if err != nil {
			all.WriteString("\r\nERROR: " + err.Error() + "\r\n")
			return false, all.String()
		}
		all.WriteString("\r\n")
		return true, all.String()
	}

	// Fast/Extreme: optimistic parallel VMAT compilation. Multiple compilers may
	// run under a shared read lock. If one collides on a shared Source 2 resource,
	// it falls back to one exclusive retry after all parallel compilers drain.
	sharedVMATCompileMu.RLock()
	out, err := runResourceCompilerOnce(rc, p)
	sharedVMATCompileMu.RUnlock()
	all.Write(out)
	if len(out) == 0 {
		all.WriteString("(no console output)\r\n")
	}
	if err != nil && (retry || lockShared) {
		all.WriteString("\r\n--- Parallel compile failed; retrying once with exclusive compiler lock ---\r\n")
		time.Sleep(120 * time.Millisecond)
		sharedVMATCompileMu.Lock()
		out2, err2 := runResourceCompilerOnce(rc, p)
		sharedVMATCompileMu.Unlock()
		all.Write(out2)
		if len(out2) == 0 {
			all.WriteString("(no console output on locked retry)\r\n")
		}
		if err2 == nil {
			all.WriteString("\r\nLOCKED RETRY RESULT: OK\r\n")
			err = nil
		} else {
			err = err2
		}
	}
	if err != nil {
		all.WriteString("\r\nERROR: " + err.Error() + "\r\n")
		return false, all.String()
	}
	all.WriteString("\r\n")
	return true, all.String()
}

func makeVTEX(textureResource string) string {
	return fmt.Sprintf(`<!-- dmx encoding keyvalues2_noids 1 format vtex 1 -->
"CDmeVtex"
{
    "m_inputTextureArray" "element_array"
    [
        "CDmeInputTexture"
        {
            "m_name" "string" "InputTexture0"
            "m_fileName" "string" "%s"
            "m_colorSpace" "string" "srgb"
            "m_typeString" "string" "2D"
            "m_imageProcessorArray" "element_array"
            [
                "CDmeImageProcessor"
                {
                    "m_algorithm" "string" "None"
                    "m_stringArg" "string" ""
                    "m_vFloat4Arg" "vector4" "0 0 0 0"
                }
            ]
        }
    ]
    "m_outputTypeString" "string" "2D"
    "m_outputFormat" "string" "DXT5"
    "m_outputClearColor" "vector4" "0 0 0 0"
    "m_nOutputMinDimension" "int" "0"
    "m_nOutputMaxDimension" "int" "0"
    "m_textureOutputChannelArray" "element_array"
    [
        "CDmeTextureOutputChannel"
        {
            "m_inputTextureArray" "string_array" [ "InputTexture0" ]
            "m_srcChannels" "string" "rgba"
            "m_dstChannels" "string" "rgba"
            "m_mipAlgorithm" "CDmeImageProcessor"
            {
                "m_algorithm" "string" "Box"
                "m_stringArg" "string" ""
                "m_vFloat4Arg" "vector4" "0 0 0 0"
            }
            "m_outputColorSpace" "string" "srgb"
        }
    ]
    "m_vClamp" "vector3" "0 0 0"
    "m_bNoLod" "bool" "0"
}
`, textureResource)
}

func makeVMAT(materialMode int, alphaRef float64, compiledTextureResource string) string {
	const kv3 = `<!-- kv3 encoding:text:version{e21c7f3c-8a33-41c5-9977-a76d3a32aa0d} format:generic:version{7412167c-06e9-4698-aff2-e63eb59037e7} -->`
	shader := "csgo_lightmappedgeneric.vfx"
	extra := ""
	switch materialMode {
	case 1:
		// CS2 Material Editor exposes Alpha Test as hard-edged transparency and
		// an Alpha Test Reference value. These are emitted as shader features.
		shader = "csgo_complex.vfx"
		if alphaRef < 0.01 {
			alphaRef = 0.01
		}
		if alphaRef > 0.99 {
			alphaRef = 0.99
		}
		extra = fmt.Sprintf("    F_ALPHA_TEST = 1\n    g_flAlphaTestReference = %.2f\n", alphaRef)
	case 2:
		// Translucency preserves partial alpha values for see-through textures.
		shader = "csgo_complex.vfx"
		extra = "    F_TRANSLUCENT = 1\n"
	}
	return fmt.Sprintf(`%s
{
    shader = "%s"
    g_tColor = resource:"%s"
%s    g_flRoughness = 0.5
}
`, kv3, shader, compiledTextureResource, extra)
}

// resizeFast uses a direct NRGBA bilinear sampler. It is much faster than the
// old image.Image.At based loops and, because v8 runs it on a worker goroutine,
// the Windows UI stays responsive even for 4096 x 4096 output.
func resizeFast(src image.Image, dstW, dstH int) image.Image {
	if dstW <= 0 || dstH <= 0 {
		return src
	}
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == dstW && sh == dstH {
		return src
	}
	srcN := toNRGBA(src)
	out := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	xScale := float64(sw) / float64(dstW)
	yScale := float64(sh) / float64(dstH)
	for y := 0; y < dstH; y++ {
		fy := (float64(y)+0.5)*yScale - 0.5
		y0 := int(math.Floor(fy))
		ty := fy - float64(y0)
		y1 := y0 + 1
		y0 = clampInt(y0, 0, sh-1)
		y1 = clampInt(y1, 0, sh-1)
		row0 := y0 * srcN.Stride
		row1 := y1 * srcN.Stride
		drow := y * out.Stride
		for x := 0; x < dstW; x++ {
			fx := (float64(x)+0.5)*xScale - 0.5
			x0 := int(math.Floor(fx))
			tx := fx - float64(x0)
			x1 := x0 + 1
			x0 = clampInt(x0, 0, sw-1)
			x1 = clampInt(x1, 0, sw-1)
			i00 := row0 + x0*4
			i10 := row0 + x1*4
			i01 := row1 + x0*4
			i11 := row1 + x1*4
			di := drow + x*4
			for c := 0; c < 4; c++ {
				a := float64(srcN.Pix[i00+c])*(1-tx) + float64(srcN.Pix[i10+c])*tx
				b := float64(srcN.Pix[i01+c])*(1-tx) + float64(srcN.Pix[i11+c])*tx
				v := a*(1-ty) + b*ty
				if v < 0 {
					v = 0
				}
				if v > 255 {
					v = 255
				}
				out.Pix[di+c] = byte(v + 0.5)
			}
		}
	}
	return out
}

func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok && n.Bounds().Min.X == 0 && n.Bounds().Min.Y == 0 {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func rgba8(c color.Color) (byte, byte, byte, byte) {
	r, g, b, a := c.RGBA()
	return byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)
}
func lerp4(c00, c10, c01, c11 byte, tx, ty float64) byte {
	top := float64(c00)*(1-tx) + float64(c10)*tx
	bot := float64(c01)*(1-tx) + float64(c11)*tx
	v := top*(1-ty) + bot*ty
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v + 0.5)
}

func writeCrashLog(where string, recovered any) {
	defer func() { _ = recover() }()
	path := filepath.Join(appDataDir(), "BIT_Texture_Tool_V0.17_crash.log")
	text := fmt.Sprintf("B.I.T. Texture Tool V0.17 crash guard\r\nTime: %s\r\nLocation: %s\r\nError: %v\r\n\r\n%s", time.Now().Format(time.RFC3339), where, recovered, debug.Stack())
	_ = os.WriteFile(path, []byte(text), 0644)
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func beginDetectCS2() {
	detectMu.Lock()
	if detectBusy {
		detectMu.Unlock()
		return
	}
	detectBusy = true
	detectResult = ""
	gameKey := selectedGame
	detectResultGame = gameKey
	detectMu.Unlock()
	if hDetectBtn != 0 {
		procEnableWindow.Call(uintptr(hDetectBtn), 0)
	}
	setText(hStatusLabel, "Detecting "+gameProfileForKey(gameKey).Name+" in the background...")
	go func(key string) {
		root := detectGameRoot(key)
		detectMu.Lock()
		detectResult = root
		detectResultGame = key
		detectMu.Unlock()
		procPostMessageW.Call(uintptr(hwndMain), WM_APP_DETECT, 0, 0)
	}(gameKey)
}

func detectGameRoot(gameKey string) string {
	gp := gameProfileForKey(gameKey)
	libs := []string{}
	pf86 := os.Getenv("ProgramFiles(x86)")
	pf := os.Getenv("ProgramFiles")
	if pf86 != "" {
		libs = append(libs, filepath.Join(pf86, "Steam"))
	}
	if pf != "" {
		libs = append(libs, filepath.Join(pf, "Steam"))
	}
	libs = append(libs, `C:\Program Files (x86)\Steam`, `C:\Program Files\Steam`)

	seen := map[string]bool{}
	var libraryRoots []string
	for _, steam := range libs {
		steam = filepath.Clean(steam)
		if seen[strings.ToLower(steam)] {
			continue
		}
		seen[strings.ToLower(steam)] = true
		if _, err := os.Stat(steam); err != nil {
			continue
		}
		libraryRoots = append(libraryRoots, steam)
		vdf := filepath.Join(steam, "steamapps", "libraryfolders.vdf")
		if f, err := os.Open(vdf); err == nil {
			sc := bufio.NewScanner(f)
			re := regexp.MustCompile(`"path"\s+"([^"]+)"`)
			for sc.Scan() {
				m := re.FindStringSubmatch(sc.Text())
				if len(m) == 2 {
					p := strings.ReplaceAll(m[1], `\\`, `\`)
					libraryRoots = append(libraryRoots, p)
				}
			}
			f.Close()
		}
	}
	for _, lib := range libraryRoots {
		if strings.HasPrefix(filepath.Clean(lib), `\\`) {
			continue
		}
		root := filepath.Join(lib, "steamapps", "common", gp.SteamFolder)
		if validGameRoot(root, gameKey) {
			return root
		}
	}
	return ""
}

func detectCS2Root() string { return detectGameRoot("cs2") }

func validCS2Root(root string) bool { return validGameRoot(root, "cs2") }

func beginPopulateAddons(root string) {
	addonMu.Lock()
	if addonBusy {
		addonMu.Unlock()
		return
	}
	addonBusy = true
	gameKey := selectedGame
	addonMu.Unlock()
	setText(hStatusLabel, "Loading "+gameProfileForKey(gameKey).Name+" addon/custom folders in the background...")
	go func(r, key string) {
		defer func() {
			if rec := recover(); rec != nil {
				addonMu.Lock()
				addonResult = addonLoadResult{root: r, game: key, err: fmt.Errorf("addon scan error: %v", rec)}
				addonMu.Unlock()
				procPostMessageW.Call(uintptr(hwndMain), WM_APP_ADDONS, 0, 0)
			}
		}()
		if !validGameRoot(r, key) {
			addonMu.Lock()
			addonResult = addonLoadResult{root: r, game: key, err: fmt.Errorf("invalid %s installation folder", gameProfileForKey(key).Name)}
			addonMu.Unlock()
			procPostMessageW.Call(uintptr(hwndMain), WM_APP_ADDONS, 0, 0)
			return
		}
		dir := gameAddonDirectory(r, key)
		ents, err := os.ReadDir(dir)
		var names []string
		if err == nil {
			for _, e := range ents {
				if e.IsDir() {
					names = append(names, e.Name())
				}
			}
			sort.Strings(names)
		} else if os.IsNotExist(err) {
			err = nil
		}
		addonMu.Lock()
		addonResult = addonLoadResult{root: r, game: key, names: names, err: err}
		addonMu.Unlock()
		procPostMessageW.Call(uintptr(hwndMain), WM_APP_ADDONS, 0, 0)
	}(root, gameKey)
}

func getSelectedAddon() string {
	if hAddonCombo == 0 || len(addonNames) == 0 {
		return ""
	}
	r, _, _ := procSendMessageW.Call(uintptr(hAddonCombo), CB_GETCURSEL, 0, 0)
	i := int(r)
	if i < 0 || i >= len(addonNames) {
		return selectedAddon
	}
	selectedAddon = addonNames[i]
	return selectedAddon
}

func cleanMaterialName(s string) (string, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, `\`, "/"))
	lower := strings.ToLower(s)
	for _, ext := range []string{".vmat", ".vtex", ".png"} {
		if strings.HasSuffix(lower, ext) {
			s = s[:len(s)-len(ext)]
			break
		}
	}
	raw := strings.Split(s, "/")
	var parts []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "." || p == ".." {
			return "", fmt.Errorf("invalid VMAT path")
		}
		p = sanitizeSegment(p)
		if p == "" {
			return "", fmt.Errorf("enter a valid VMAT name")
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("enter a VMAT/material name")
	}
	return strings.Join(parts, "/"), nil
}

func sanitizeSegment(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		good := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if good {
			b.WriteRune(r)
			lastUnderscore = false
		} else if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func sanitizeAddon(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "..") || strings.ContainsAny(s, `\/:*?"<>|`) {
		return ""
	}
	return s
}

func isPowerOfTwo(v int) bool { return v > 0 && (v&(v-1)) == 0 }
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func existsAny(paths ...string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func uniqueSquarePath(src string) string {
	dir := filepath.Dir(src)
	ext := filepath.Ext(src)
	base := strings.TrimSuffix(filepath.Base(src), ext)
	p := filepath.Join(dir, base+"_square.png")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	for i := 2; ; i++ {
		p = filepath.Join(dir, fmt.Sprintf("%s_square_%d.png", base, i))
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

func uniqueConvertedPath(src string) string {
	dir := filepath.Dir(src)
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	p := filepath.Join(dir, base+"_converted.png")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	for i := 2; ; i++ {
		p = filepath.Join(dir, fmt.Sprintf("%s_converted_%d.png", base, i))
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

func clearJunkFolder() (int, error) {
	dir := junkDir()
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		// Only files directly inside the app-owned Junk directory are deleted.
		if err := os.Remove(p); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func openFile(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	procShellExecuteW.Call(0, uintptr(unsafe.Pointer(u16("open"))), uintptr(unsafe.Pointer(u16(path))), 0, 0, SW_SHOW)
}

func openFolder(path string) {
	procShellExecuteW.Call(0, uintptr(unsafe.Pointer(u16("open"))), uintptr(unsafe.Pointer(u16(path))), 0, 0, SW_SHOW)
}

func rgb(r, g, b byte) uintptr { return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16) }

// Keep image/color linked into the Windows build so transparent PNG decoding remains explicit.
var _ = color.RGBA{}
