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
	MsgTopicUnlocked         Key = "topic_unlocked"

	MsgMilestoneTopicMastered Key = "milestone_topic_mastered"
	MsgMilestoneXP            Key = "milestone_xp"
	MsgMilestoneSubjectDone   Key = "milestone_subject_done"
	MsgMilestoneStreakRecord   Key = "milestone_streak_record"

	MsgGroupCreated          Key = "group_created"
	MsgGroupJoined           Key = "group_joined"
	MsgGroupLeft             Key = "group_left"
	MsgGroupList             Key = "group_list"
	MsgGroupListEmpty        Key = "group_list_empty"
	MsgGroupNotFound         Key = "group_not_found"
	MsgGroupAlreadyMember    Key = "group_already_member"
	MsgGroupOwnerCannotLeave Key = "group_owner_cannot_leave"
	MsgGroupArchived         Key = "group_archived"
	MsgGroupCreateDenied     Key = "group_create_denied"
	MsgGroupSyllabusMismatch Key = "group_syllabus_mismatch"
	MsgGroupUsage            Key = "group_usage"
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
		MsgTopicUnlocked:          "Tahniah! Anda telah membuka topik baru:\n- %s\n\nGuna /learn untuk mula belajar topik ini.",
		MsgMilestoneTopicMastered: "🏆 *Tahniah!*\n\nAnda telah menguasai topik *%s*!\n⭐ +%d XP\n\nTeruskan ke topik seterusnya! 💪",
		MsgMilestoneXP:            "🌟 *Pencapaian XP!*\n\nAnda telah mencapai *%d XP*!\n\nKerja keras anda membuahkan hasil! 🎉",
		MsgMilestoneSubjectDone:   "🎓 *LUAR BIASA!*\n\nAnda telah menguasai semua topik dalam *%s*!\n\nAnda seorang juara matematik! 🏅",
		MsgMilestoneStreakRecord:   "🔥 *Rekod Baru!*\n\nStreak terpanjang anda: *%d hari*!\n\nDedikasi yang mengagumkan! 🏅",
		MsgGroupCreated:          "Kumpulan berjaya dicipta!\n\nNama: %s\nKod: *%s*\n\nKongsi kod ini dengan pelajar anda.",
		MsgGroupJoined:           "Berjaya menyertai *%s*! Anda kini mempunyai %d rakan sekelas.",
		MsgGroupLeft:             "Anda telah keluar dari *%s*.",
		MsgGroupList:             "Kumpulan anda:\n%s",
		MsgGroupListEmpty:        "Anda belum menyertai sebarang kumpulan. Guna /join <kod> untuk menyertai.",
		MsgGroupNotFound:         "Kod kumpulan tidak dijumpai. Sila semak dan cuba lagi.",
		MsgGroupAlreadyMember:    "Anda sudah menjadi ahli kumpulan ini.",
		MsgGroupOwnerCannotLeave: "Pemilik tidak boleh keluar dari kumpulan. Arkibkan kumpulan ini dahulu.",
		MsgGroupArchived:         "Kumpulan ini telah diarkibkan.",
		MsgGroupCreateDenied:     "Hanya guru dan pentadbir boleh mencipta kumpulan.",
		MsgGroupSyllabusMismatch: "Kumpulan ini untuk pelajar %s. Profil anda ditetapkan kepada %s.",
		MsgGroupUsage:            "Guna:\n/group create <nama> — Cipta kumpulan\n/group join <kod> atau /join <kod> — Sertai kumpulan\n/group leave — Keluar dari kumpulan\n/group list — Senarai kumpulan anda",
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
		MsgTopicUnlocked:          "Congratulations! You've unlocked new topics:\n- %s\n\nUse /learn to start studying them.",
		MsgMilestoneTopicMastered: "🏆 *Congratulations!*\n\nYou've mastered *%s*!\n⭐ +%d XP\n\nKeep going to the next topic! 💪",
		MsgMilestoneXP:            "🌟 *XP Milestone!*\n\nYou've reached *%d XP*!\n\nYour hard work is paying off! 🎉",
		MsgMilestoneSubjectDone:   "🎓 *AMAZING!*\n\nYou've mastered all topics in *%s*!\n\nYou're a math champion! 🏅",
		MsgMilestoneStreakRecord:   "🔥 *New Record!*\n\nYour longest streak: *%d days*!\n\nIncredible dedication! 🏅",
		MsgGroupCreated:          "Group created!\n\nName: %s\nCode: *%s*\n\nShare this code with your students.",
		MsgGroupJoined:           "Joined *%s*! You now have %d classmates.",
		MsgGroupLeft:             "You've left *%s*.",
		MsgGroupList:             "Your groups:\n%s",
		MsgGroupListEmpty:        "You haven't joined any groups yet. Use /join <code> to join one.",
		MsgGroupNotFound:         "Group code not found. Please check and try again.",
		MsgGroupAlreadyMember:    "You're already a member of this group.",
		MsgGroupOwnerCannotLeave: "Owner cannot leave the group. Archive it instead.",
		MsgGroupArchived:         "This group has been archived.",
		MsgGroupCreateDenied:     "Only teachers and admins can create groups.",
		MsgGroupSyllabusMismatch: "This group is for %s students. Your profile is set to %s.",
		MsgGroupUsage:            "Usage:\n/group create <name> — Create a group\n/group join <code> or /join <code> — Join a group\n/group leave — Leave a group\n/group list — List your groups",
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
		MsgTopicUnlocked:          "恭喜！你已解锁新主题：\n- %s\n\n使用 /learn 开始学习。",
		MsgMilestoneTopicMastered: "🏆 *恭喜！*\n\n你已经掌握了 *%s*！\n⭐ +%d XP\n\n继续下一个主题吧！💪",
		MsgMilestoneXP:            "🌟 *XP 里程碑！*\n\n你已达到 *%d XP*！\n\n你的努力正在得到回报！🎉",
		MsgMilestoneSubjectDone:   "🎓 *太厉害了！*\n\n你已掌握 *%s* 的所有主题！\n\n你是数学冠军！🏅",
		MsgMilestoneStreakRecord:   "🔥 *新纪录！*\n\n你最长的连续学习记录：*%d 天*！\n\n令人敬佩的毅力！🏅",
		MsgGroupCreated:          "群组创建成功！\n\n名称：%s\n代码：*%s*\n\n将此代码分享给你的学生。",
		MsgGroupJoined:           "已加入 *%s*！你现在有 %d 位同学。",
		MsgGroupLeft:             "你已退出 *%s*。",
		MsgGroupList:             "你的群组：\n%s",
		MsgGroupListEmpty:        "你还没有加入任何群组。使用 /join <代码> 加入。",
		MsgGroupNotFound:         "找不到群组代码。请检查后重试。",
		MsgGroupAlreadyMember:    "你已经是这个群组的成员了。",
		MsgGroupOwnerCannotLeave: "群主不能退出群组。请先归档群组。",
		MsgGroupArchived:         "这个群组已被归档。",
		MsgGroupCreateDenied:     "只有老师和管理员可以创建群组。",
		MsgGroupSyllabusMismatch: "这个群组是为 %s 的学生设立的。你的资料设置为 %s。",
		MsgGroupUsage:            "用法：\n/group create <名称> — 创建群组\n/group join <代码> 或 /join <代码> — 加入群组\n/group leave — 退出群组\n/group list — 列出你的群组",
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
