// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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

	MsgGroupCreateUsage  Key = "group_create_usage"
	MsgGroupCreated      Key = "group_created"
	MsgGroupJoinUsage    Key = "group_join_usage"
	MsgGroupJoined       Key = "group_joined"
	MsgGroupNotFound     Key = "group_not_found"
	MsgGroupUserNotFound Key = "group_user_not_found"
	MsgGroupNoGroups     Key = "group_no_groups"
	MsgLeaderboardEmpty  Key = "leaderboard_empty"
	MsgGroupClosed       Key = "group_closed"

	MsgChallengeComplete      Key = "challenge_complete"
	MsgChallengeReviewOffer   Key = "challenge_review_offer"
	MsgChallengeReviewDone    Key = "challenge_review_done"
	MsgChallengeReviewSkip    Key = "challenge_review_skip"
	MsgChallengeFinishFirst   Key = "challenge_finish_first"
	MsgChallengeCorrect       Key = "challenge_correct"
	MsgChallengeIncorrect     Key = "challenge_incorrect"
	MsgChallengeReviewRetry   Key = "challenge_review_retry"
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
		MsgGroupCreateUsage:  "Guna: /create_group <nama>\nContoh: /create_group Kumpulan Algebra",
		MsgGroupCreated:      "Kumpulan *%s* telah dibuat!\n\nKod jemputan: *%s*\nKongsi kod ini supaya rakan boleh sertai dengan /join %s",
		MsgGroupJoinUsage:    "Guna: /join <kod>\nContoh: /join ABC123",
		MsgGroupJoined:       "Anda telah menyertai *%s*! 🎉",
		MsgGroupNotFound:     "Kumpulan dengan kod %s tidak dijumpai.",
		MsgGroupUserNotFound: "Sila mulakan dulu dengan /start.",
		MsgGroupNoGroups:     "Anda belum menyertai sebarang kumpulan.\nGuna /join <kod> untuk sertai, atau /create_group <nama> untuk buat baru.",
		MsgLeaderboardEmpty:  "Belum ada data papan pendahulu untuk *%s*.\nTeruskan belajar dan semak semula minggu depan!",
		MsgGroupClosed:       "*%s* tidak lagi menerima ahli baru.",
		MsgChallengeComplete:    "🏁 Cabaran selesai!\n\n📊 Skor: %d/%d (%d%%)",
		MsgChallengeReviewOffer: "Anda salah %d soalan. Mahu ulang kaji?\n\nBalas *review* untuk mula, atau apa sahaja untuk teruskan.",
		MsgChallengeReviewDone:  "🎉 Ulang kaji selesai!\nAnda dapat %d/%d betul.\n⭐ +50 XP",
		MsgChallengeReviewSkip:  "Baik, kita teruskan. Anda boleh ulang kaji kemudian.",
		MsgChallengeFinishFirst: "Selesaikan cabaran semasa anda dulu, kemudian cuba lagi.",
		MsgChallengeCorrect:     "✅ Betul!",
		MsgChallengeIncorrect:   "❌ Salah\nJawapan: %s",
		MsgChallengeReviewRetry: "Belum tepat. Cuba lagi.",
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
		MsgGroupCreateUsage:  "Usage: /create_group <name>\nExample: /create_group Algebra Squad",
		MsgGroupCreated:      "Group *%s* created!\n\nJoin code: *%s*\nShare this code so friends can join with /join %s",
		MsgGroupJoinUsage:    "Usage: /join <code>\nExample: /join ABC123",
		MsgGroupJoined:       "You've joined *%s*! 🎉",
		MsgGroupNotFound:     "No group found with code %s.",
		MsgGroupUserNotFound: "Please start first with /start.",
		MsgGroupNoGroups:     "You haven't joined any groups yet.\nUse /join <code> to join, or /create_group <name> to create one.",
		MsgLeaderboardEmpty:  "No leaderboard data yet for *%s*.\nKeep studying and check back next week!",
		MsgGroupClosed:       "*%s* is no longer accepting new members.",
		MsgChallengeComplete:    "🏁 Challenge complete!\n\n📊 Score: %d/%d (%d%%)",
		MsgChallengeReviewOffer: "You missed %d question(s). Want to review them?\n\nReply *review* to start, or anything else to continue.",
		MsgChallengeReviewDone:  "🎉 Review complete!\nYou got %d/%d correct.\n⭐ +50 XP",
		MsgChallengeReviewSkip:  "Okay, moving on. You can always review later.",
		MsgChallengeFinishFirst: "Finish your current challenge first, then try again.",
		MsgChallengeCorrect:     "✅ Correct!",
		MsgChallengeIncorrect:   "❌ Incorrect\nAnswer: %s",
		MsgChallengeReviewRetry: "Not quite. Try again.",
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
		MsgGroupCreateUsage:  "用法：/create_group <名称>\n例如：/create_group 代数小组",
		MsgGroupCreated:      "小组 *%s* 已创建！\n\n加入代码：*%s*\n分享此代码，好友可以用 /join %s 加入",
		MsgGroupJoinUsage:    "用法：/join <代码>\n例如：/join ABC123",
		MsgGroupJoined:       "你已加入 *%s*！🎉",
		MsgGroupNotFound:     "未找到代码为 %s 的小组。",
		MsgGroupUserNotFound: "请先使用 /start 开始。",
		MsgGroupNoGroups:     "你还没有加入任何小组。\n使用 /join <代码> 加入，或 /create_group <名称> 创建一个。",
		MsgLeaderboardEmpty:  "*%s* 暂无排行榜数据。\n继续学习，下周再来查看！",
		MsgGroupClosed:       "*%s* 不再接受新成员。",
		MsgChallengeComplete:    "🏁 挑战完成！\n\n📊 分数：%d/%d (%d%%)",
		MsgChallengeReviewOffer: "你答错了 %d 道题。要复习吗？\n\n回复 *review* 开始，或其他内容继续。",
		MsgChallengeReviewDone:  "🎉 复习完成！\n你答对了 %d/%d 道题。\n⭐ +50 XP",
		MsgChallengeReviewSkip:  "好的，我们继续。你随时可以回来复习。",
		MsgChallengeFinishFirst: "请先完成当前挑战，然后再试。",
		MsgChallengeCorrect:     "✅ 正确！",
		MsgChallengeIncorrect:   "❌ 不正确\n答案：%s",
		MsgChallengeReviewRetry: "还不对。再试一次。",
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
