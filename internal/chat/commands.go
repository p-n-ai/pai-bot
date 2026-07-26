// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

// BotCommand defines a bot command with its slash name and description.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// registeredCommands is the single source of truth for all bot commands
// shown in chat channel command menus.
// Add new commands here — they auto-sync to Telegram on bot startup.
var registeredCommands = []BotCommand{
	{Command: "help", Description: "Senarai semua arahan"},
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

var devCommands = []BotCommand{
	{Command: "dev_reset", Description: "[DEV] Full reset: mastery, XP, streaks, goals"},
	{Command: "dev_boost", Description: "[DEV] Boost current topic mastery (default 85%)"},
	{Command: "dev_close_group", Description: "[DEV] Toggle group open/closed"},
}

// AllCommands returns a caller-owned snapshot of the configured bot commands.
func AllCommands(devMode bool) []BotCommand {
	count := len(registeredCommands)
	if devMode {
		count += len(devCommands)
	}
	all := make([]BotCommand, 0, count)
	all = append(all, registeredCommands...)
	if devMode {
		all = append(all, devCommands...)
	}
	return all
}
