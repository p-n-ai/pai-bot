package adminapi

import "testing"

func TestNormalizeOnboardingSubmitGeneratesSlug(t *testing.T) {
	req, err := normalizeOnboardingSubmit(SubmitOnboardingRequest{
		Curriculum: OnboardingCurriculum{
			SyllabusID: "kssm-algebra",
			Label:      "KSSM Algebra",
		},
		FirstClass: OnboardingFirstClass{
			Name: "Steady Otter Harbor",
		},
		BotSetup: OnboardingBotSetup{
			Preset: "guided-practice",
		},
	})
	if err != nil {
		t.Fatalf("normalizeOnboardingSubmit() error = %v", err)
	}
	if req.FirstClass.Slug != "steady-otter-harbor" {
		t.Fatalf("first_class.slug = %q, want steady-otter-harbor", req.FirstClass.Slug)
	}
}

func TestBuildOnboardingJoinLink(t *testing.T) {
	got := buildOnboardingJoinLink("http://127.0.0.1:3000/", "steady-otter-harbor")
	want := "http://127.0.0.1:3000/join/steady-otter-harbor"
	if got != want {
		t.Fatalf("buildOnboardingJoinLink() = %q, want %q", got, want)
	}
}
