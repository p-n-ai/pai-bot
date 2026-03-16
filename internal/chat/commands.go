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
}
