//go:build windows

// Package wintun embeds the official amd64 Wintun driver DLL
// (https://www.wintun.net, dual GPLv2/MIT licensed -- see
// WINTUN_LICENSE.txt in this directory) so Kite ships as a single
// portable executable: xray-core's TUN inbound needs wintun.dll sitting
// next to the running binary on Windows, and this lets Kite write it out
// itself on first use instead of requiring a separate installer/download.
package wintun

import _ "embed"

//go:embed wintun_amd64.dll
var DLL []byte
