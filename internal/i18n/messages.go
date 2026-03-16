package i18n

import (
	"fmt"
	"strings"
)

type Key string

const (
	DefaultLocale = "ms"

	MsgTechnicalIssue        Key = "technical_issue"
	MsgImageProcessingFailed Key = "image_processing_failed"
	MsgHistoryCleared        Key = "history_cleared"
	MsgUnknownCommand        Key = "unknown_command"
	MsgMultilingualDisabled  Key = "multilingual_disabled"
	MsgLanguagePrompt        Key = "language_prompt"
	MsgLanguageInvalidFormat Key = "language_invalid_format"
	MsgDefaultStudentName    Key = "default_student_name"
	MsgStartOnboardingForm   Key = "start_onboarding_form"
	MsgStartOnboardingLang   Key = "start_onboarding_lang"
	MsgLanguageUnclear       Key = "language_unclear"
	MsgOnboardingFormUnclear Key = "onboarding_form_unclear"
	MsgOnboardingFormPrompt  Key = "onboarding_form_prompt"
	MsgOnboardingCompleted   Key = "onboarding_completed"
	MsgLanguageChanged       Key = "language_changed"
	MsgRatingThanks          Key = "rating_thanks"
	MsgProfileReset          Key = "profile_reset"
	MsgLearnUsage            Key = "learn_usage"
	MsgLearnTopicNotFound    Key = "learn_topic_not_found"
	MsgLearnTopicSet         Key = "learn_topic_set"
)

var catalog = map[string]map[Key]string{
	"ms": {
		MsgTechnicalIssue:        "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar.",
		MsgImageProcessingFailed: "Saya terima gambar anda, tapi gagal memproses fail gambar itu. Cuba hantar semula gambar yang lebih jelas.",
		MsgHistoryCleared:        "Sejarah perbualan telah dikosongkan. Hantar soalan baru untuk mula semula.",
		MsgUnknownCommand:        "Arahan tidak diketahui: %s\nGuna /start untuk bermula, /clear untuk reset perbualan, atau /language untuk tukar bahasa.",
		MsgMultilingualDisabled:  "Ciri multi-bahasa dimatikan oleh konfigurasi pelayan.",
		MsgLanguagePrompt:        "Bahasa pilihan anda?\nChoose your language:\n- English\n- Bahasa Melayu\n- 中文",
		MsgLanguageInvalidFormat: "Format tidak sah. Guna /language en, /language ms, atau /language zh.",
		MsgDefaultStudentName:    "pelajar",
		MsgStartOnboardingForm: `Hai %s!

Saya P&AI Bot — tutor matematik peribadi anda!

Saya boleh membantu anda dengan KSSM Matematik:
- Tingkatan 1
- Tingkatan 2
- Tingkatan 3

Tingkatan berapa anda sekarang?
Balas dengan: 1, 2, atau 3.`,
		MsgStartOnboardingLang: `Hai %s!

Saya P&AI Bot — tutor matematik peribadi anda.

Bahasa pilihan anda untuk sesi ini?
- English
- Bahasa Melayu
- 中文

Anda boleh jawab bebas (contoh: English / BM / Chinese).`,
		MsgLanguageUnclear:       "Saya belum pasti bahasa pilihan anda. Boleh jawab: English, Bahasa Melayu, atau 中文.",
		MsgOnboardingFormUnclear: "Saya belum pasti tingkatan anda. Boleh jawab bebas (contoh: saya tingkatan 2 / form two), atau balas terus 1, 2, atau 3.",
		MsgOnboardingFormPrompt: `Baik. Saya boleh bantu untuk:
- Tingkatan 1
- Tingkatan 2
- Tingkatan 3

Tingkatan berapa anda sekarang?`,
		MsgOnboardingCompleted: "Bagus, anda Tingkatan %d. Sekarang hantar topik atau soalan matematik yang anda mahu belajar.",
		MsgLanguageChanged:     "Bahasa telah ditukar ke Bahasa Melayu.",
		MsgRatingThanks:        "Terima kasih atas rating anda. Jom kita sambung.",
		MsgProfileReset:        "Profil pembelajaran anda telah direset. Mari tetapkan semula.",
		MsgLearnUsage:          "Guna: /learn <topik>\nContoh: /learn persamaan linear",
		MsgLearnTopicNotFound:  "Topik tidak dijumpai: %s\nGuna /learn <topik> dengan nama topik yang betul.",
		MsgLearnTopicSet:       "Topik ditetapkan: %s\nMari kita mula belajar!",
	},
	"en": {
		MsgTechnicalIssue:        "Sorry, I'm facing a technical issue right now. Please try again shortly.",
		MsgImageProcessingFailed: "I received your image, but couldn't process it. Please resend a clearer image.",
		MsgHistoryCleared:        "Conversation history has been cleared. Send a new question to start again.",
		MsgUnknownCommand:        "Unknown command: %s\nUse /start to begin, /clear to reset, or /language to change language.",
		MsgMultilingualDisabled:  "Multilingual mode is disabled by server configuration.",
		MsgLanguagePrompt:        "Choose your language:\n- English\n- Bahasa Melayu\n- 中文",
		MsgLanguageInvalidFormat: "Invalid format. Use /language en, /language ms, or /language zh.",
		MsgDefaultStudentName:    "student",
		MsgStartOnboardingForm: `Hi %s!

I'm P&AI Bot — your personal math tutor!

I can help you with KSSM Mathematics:
- Form 1
- Form 2
- Form 3

What form are you in now?
Reply with: 1, 2, or 3.`,
		MsgStartOnboardingLang: `Hi %s!

I'm P&AI Bot — your personal math tutor.

Choose your preferred session language:
- English
- Bahasa Melayu
- 中文

You can answer freely (example: English / BM / Chinese).`,
		MsgLanguageUnclear:       "I couldn't determine your preferred language. Please reply with: English, Bahasa Melayu, or 中文.",
		MsgOnboardingFormUnclear: "I couldn't determine your form. You can reply freely (for example: form 2 / tingkatan 2), or just 1, 2, or 3.",
		MsgOnboardingFormPrompt: `Great. I can help with:
- Form 1
- Form 2
- Form 3

Which form are you in now?`,
		MsgOnboardingCompleted: "Great, you are Form %d. Send any math topic or question you want to learn now.",
		MsgLanguageChanged:     "Language updated to English.",
		MsgRatingThanks:        "Thanks for your rating. Let's continue.",
		MsgProfileReset:        "Your learner profile has been reset. Let's set it up again.",
		MsgLearnUsage:          "Usage: /learn <topic>\nExample: /learn linear equations",
		MsgLearnTopicNotFound:  "Topic not found: %s\nUse /learn <topic> with a valid topic name.",
		MsgLearnTopicSet:       "Topic set: %s\nLet's start learning!",
	},
	"zh": {
		MsgTechnicalIssue:        "抱歉，我目前遇到技术问题。请稍后再试。",
		MsgImageProcessingFailed: "我收到了你的图片，但暂时无法处理。请重新发送更清晰的图片。",
		MsgHistoryCleared:        "对话记录已清除。发送新问题即可重新开始。",
		MsgUnknownCommand:        "未知指令：%s\n使用 /start 开始，/clear 重置，或 /language 切换语言。",
		MsgMultilingualDisabled:  "多语言模式已被服务器配置禁用。",
		MsgLanguagePrompt:        "请选择你的语言：\n- English\n- Bahasa Melayu\n- 中文",
		MsgLanguageInvalidFormat: "格式无效。请使用 /language en、/language ms 或 /language zh。",
		MsgDefaultStudentName:    "学生",
		MsgStartOnboardingForm: `你好 %s！

我是 P&AI Bot —— 你的数学私人导师！

我可以帮助你学习 KSSM 数学：
- Form 1
- Form 2
- Form 3

你现在是几年级？
请回复：1、2 或 3。`,
		MsgStartOnboardingLang: `你好 %s！

我是 P&AI Bot —— 你的数学私人导师。

请选择本次学习语言：
- English
- Bahasa Melayu
- 中文

你可以自由输入（例如：English / BM / Chinese）。`,
		MsgLanguageUnclear:       "我还不能确定你的语言偏好。请回复：English、Bahasa Melayu 或 中文。",
		MsgOnboardingFormUnclear: "我还不能确定你的年级。你可以自由回答（例如：Form 2 / Tingkatan 2），或直接回复 1、2、3。",
		MsgOnboardingFormPrompt: `好的。我可以帮助你学习：
- Form 1
- Form 2
- Form 3

你现在是几年级（中学）？`,
		MsgOnboardingCompleted: "好的，你现在是 Form %d。现在发你想学的数学题目或主题。",
		MsgLanguageChanged:     "语言已切换为中文。",
		MsgRatingThanks:        "谢谢你的评分。我们继续。",
		MsgProfileReset:        "你的学习档案已重置。我们重新设置一次。",
		MsgLearnUsage:          "用法：/learn <主题>\n例如：/learn 线性方程",
		MsgLearnTopicNotFound:  "未找到主题：%s\n请使用 /learn <主题> 并输入正确的主题名称。",
		MsgLearnTopicSet:       "主题已设置：%s\n我们开始学习吧！",
	},
}

func NormalizeLocale(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	switch {
	case strings.HasPrefix(l, "en"):
		return "en"
	case strings.HasPrefix(l, "zh"):
		return "zh"
	case strings.HasPrefix(l, "ms"), strings.HasPrefix(l, "bm"), strings.HasPrefix(l, "id"):
		return "ms"
	default:
		return ""
	}
}

func S(locale string, key Key, args ...any) string {
	loc := NormalizeLocale(locale)
	if loc == "" {
		loc = DefaultLocale
	}
	msg := lookup(loc, key)
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

func lookup(locale string, key Key) string {
	if dict, ok := catalog[locale]; ok {
		if msg, ok := dict[key]; ok {
			return msg
		}
	}
	if msg, ok := catalog[DefaultLocale][key]; ok {
		return msg
	}
	return string(key)
}
