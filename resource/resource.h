#pragma once

// Artwork carried in the binary, the way the Go build used //go:embed.
// RCDATA rather than RT_BITMAP: these stay PNG bytes and get decoded through
// WIC at startup, so the alpha channel survives.
#define IDR_LOGO_LIGHT   101
#define IDR_LOGO_DARK    102

#define IDR_FLAG_GB      110
#define IDR_FLAG_RU      111
#define IDR_FLAG_CZ      112
#define IDR_FLAG_BR      113
#define IDR_FLAG_FR      114
#define IDR_FLAG_DE      115
#define IDR_FLAG_ES      116
#define IDR_FLAG_PL      117
#define IDR_FLAG_TR      118
