// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

// BotCommand defines a bot command with its slash name and description.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// RegisteredCommands is the single source of truth for all bot commands
// shown in Telegram's command menu (and any future chat channel).
// Add new commands here — they auto-sync to Telegram on bot startup.
var RegisteredCommands = []BotCommand{
	{Command: "start", Description: "Mulakan sesi pembelajaran"},
	{Command: "clear", Description: "Reset perbualan semasa"},
	{Command: "language", Description: "Tukar bahasa (English/BM/中文)"},
	{Command: "progress", Description: "Lihat kemajuan pembelajaran"},
	{Command: "goal", Description: "Tetapkan matlamat pembelajaran"},
	{Command: "learn", Description: "Pilih topik untuk belajar"},
	{Command: "create_group", Description: "Buat kumpulan belajar baru"},
	{Command: "join", Description: "Sertai kumpulan dengan kod"},
	{Command: "leaderboard", Description: "Papan pendahulu mingguan kumpulan"},
	{Command: "challenge", Description: "Cabaran kuiz dengan rakan atau AI"},
}

// DevCommands are only shown when dev mode is enabled.
var DevCommands = []BotCommand{
	{Command: "dev_reset", Description: "[DEV] Full reset: mastery, XP, streaks, goals"},
	{Command: "dev_boost", Description: "[DEV] Boost current topic mastery (default 85%)"},
	{Command: "dev_close_group", Description: "[DEV] Toggle group open/closed"},
}

// AllCommands returns RegisteredCommands + DevCommands when devMode is true.
func AllCommands(devMode bool) []BotCommand {
	if !devMode {
		return RegisteredCommands
	}
	all := make([]BotCommand, 0, len(RegisteredCommands)+len(DevCommands))
	all = append(all, RegisteredCommands...)
	all = append(all, DevCommands...)
	return all
}
