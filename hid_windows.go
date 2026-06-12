//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Implementação HID em Go puro para Windows: enumera os dispositivos HID via
// SetupAPI, encontra o cooler por VID/PID com hid.dll e escreve com WriteFile.
// Não usa cgo — nenhum compilador C/MinGW é necessário para gerar o .exe.

var (
	modHID                     = windows.NewLazyDLL("hid.dll")
	procHidD_GetHidGuid        = modHID.NewProc("HidD_GetHidGuid")
	procHidD_GetAttributes     = modHID.NewProc("HidD_GetAttributes")
	procHidD_GetPreparsedData  = modHID.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData = modHID.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps           = modHID.NewProc("HidP_GetCaps")

	modSetupAPI                         = windows.NewLazyDLL("setupapi.dll")
	procSetupDiGetClassDevs             = modSetupAPI.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces     = modSetupAPI.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetail = modSetupAPI.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList    = modSetupAPI.NewProc("SetupDiDestroyDeviceInfoList")
)

const (
	digcfPresent         = 0x02
	digcfDeviceInterface = 0x10
	invalidHandleValue   = ^uintptr(0)
)

type spDeviceInterfaceData struct {
	cbSize             uint32
	interfaceClassGUID windows.GUID
	flags              uint32
	reserved           uintptr
}

// O DevicePath real é um WCHAR[] logo após cbSize; lemos a partir do offset.
type spDeviceInterfaceDetailData struct {
	cbSize     uint32
	devicePath [1]uint16
}

type hiddAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

// HIDP_CAPS — só usamos OutputReportByteLength, mas o struct precisa do tamanho
// completo para o HidP_GetCaps preencher sem corromper a pilha.
type hidpCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

type windowsDevice struct {
	handle windows.Handle
	outLen int // OutputReportByteLength exigido pelo WriteFile
}

func openDevice(vid, pid uint16) (Device, error) {
	var guid windows.GUID
	procHidD_GetHidGuid.Call(uintptr(unsafe.Pointer(&guid)))

	devInfo, _, err := procSetupDiGetClassDevs.Call(
		uintptr(unsafe.Pointer(&guid)), 0, 0, uintptr(digcfPresent|digcfDeviceInterface))
	if devInfo == invalidHandleValue {
		return nil, fmt.Errorf("SetupDiGetClassDevs falhou: %v", err)
	}
	defer procSetupDiDestroyDeviceInfoList.Call(devInfo)

	var iface spDeviceInterfaceData
	iface.cbSize = uint32(unsafe.Sizeof(iface))

	for i := uint32(0); ; i++ {
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			devInfo, 0, uintptr(unsafe.Pointer(&guid)), uintptr(i),
			uintptr(unsafe.Pointer(&iface)))
		if r == 0 {
			break // ERROR_NO_MORE_ITEMS: fim da enumeração
		}

		// Primeira chamada: descobre o tamanho necessário do buffer de detalhe.
		var reqSize uint32
		procSetupDiGetDeviceInterfaceDetail.Call(
			devInfo, uintptr(unsafe.Pointer(&iface)), 0, 0,
			uintptr(unsafe.Pointer(&reqSize)), 0)
		if reqSize == 0 {
			continue
		}

		buf := make([]byte, reqSize)
		detail := (*spDeviceInterfaceDetailData)(unsafe.Pointer(&buf[0]))
		// cbSize do header: 8 em 64-bit, 6 em 32-bit (regra do Win32).
		if unsafe.Sizeof(uintptr(0)) == 8 {
			detail.cbSize = 8
		} else {
			detail.cbSize = 6
		}
		r, _, _ = procSetupDiGetDeviceInterfaceDetail.Call(
			devInfo, uintptr(unsafe.Pointer(&iface)),
			uintptr(unsafe.Pointer(detail)), uintptr(reqSize), 0, 0)
		if r == 0 {
			continue
		}

		pathPtr := (*uint16)(unsafe.Pointer(
			uintptr(unsafe.Pointer(detail)) + unsafe.Offsetof(detail.devicePath)))
		path := windows.UTF16PtrToString(pathPtr)

		h, err := windows.CreateFile(
			windows.StringToUTF16Ptr(path),
			windows.GENERIC_WRITE,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
			nil, windows.OPEN_EXISTING, 0, 0)
		if err != nil {
			continue // alguns devices não permitem abrir; ignora
		}

		var attrs hiddAttributes
		attrs.Size = uint32(unsafe.Sizeof(attrs))
		ok, _, _ := procHidD_GetAttributes.Call(uintptr(h), uintptr(unsafe.Pointer(&attrs)))
		if ok == 0 || attrs.VendorID != vid || attrs.ProductID != pid {
			windows.CloseHandle(h)
			continue
		}

		// Encontrado. Descobre o tamanho de report exigido pelo WriteFile.
		outLen := len([]byte{0x00, 0x00})
		var pp uintptr
		if ok, _, _ := procHidD_GetPreparsedData.Call(uintptr(h), uintptr(unsafe.Pointer(&pp))); ok != 0 {
			var caps hidpCaps
			procHidP_GetCaps.Call(pp, uintptr(unsafe.Pointer(&caps)))
			if caps.OutputReportByteLength > 0 {
				outLen = int(caps.OutputReportByteLength)
			}
			procHidD_FreePreparsedData.Call(pp)
		}

		return &windowsDevice{handle: h, outLen: outLen}, nil
	}

	return nil, fmt.Errorf("dispositivo HID %04x:%04x não encontrado (o cooler está conectado?)", vid, pid)
}

func (d *windowsDevice) WriteTemp(temp byte) error {
	// Report HID: byte 0 = report ID (0x00), byte 1 = temperatura.
	report := []byte{0x00, temp}
	// O Windows exige que o buffer tenha exatamente OutputReportByteLength.
	if d.outLen > len(report) {
		padded := make([]byte, d.outLen)
		copy(padded, report)
		report = padded
	}
	var written uint32
	return windows.WriteFile(d.handle, report, &written, nil)
}

func (d *windowsDevice) Close() error {
	return windows.CloseHandle(d.handle)
}
