package eval

import (
	"strings"
	"testing"
)

func TestSolveCandyDefault(t *testing.T) {
	d := CandyDistribution{
		AppleCircle:      7,
		PeachCircle:      9,
		WatermelonCircle: 8,
		AppleStar:        7,
		PeachStar:        6,
		WatermelonStar:   4,
	}
	if ans := SolveCandy(d); ans != 21 {
		t.Fatalf("expected 21, got %d", ans)
	}
}

// The solver must agree with a brute-force simulation of the worst case.
func TestSolveCandyMatchesBruteForce(t *testing.T) {
	cases := []CandyDistribution{
		{7, 9, 8, 7, 6, 4},
		{5, 5, 4, 5, 5, 3},
		{10, 10, 8, 10, 10, 7},
		{6, 8, 5, 9, 7, 6},
	}
	for _, d := range cases {
		got := SolveCandy(d)
		want := bruteForceCandy(d)
		if got != want {
			t.Errorf("distribution %+v: solver=%d bruteForce=%d", d, got, want)
		}
	}
}

// bruteForceCandy enumerates every (circles, stars) pair and checks whether an
// adversarial draw can defeat the requirement. The smallest winning pair +1
// is the answer.
func bruteForceCandy(d CandyDistribution) int {
	totalCircle := d.AppleCircle + d.PeachCircle + d.WatermelonCircle
	totalStar := d.AppleStar + d.PeachStar + d.WatermelonStar
	best := 1 << 30
	for c := 1; c <= totalCircle; c++ {
		for s := 1; s <= totalStar; s++ {
			if canAdversaryDefeat(d, c, s) {
				continue
			}
			if c+s < best {
				best = c + s
			}
		}
	}
	if best == 1<<30 {
		return 0
	}
	return best
}

// canAdversaryDefeat returns true when the adversary can hand out exactly c
// circles and s stars while never producing a valid apple/peach cross-shape pair.
func canAdversaryDefeat(d CandyDistribution, c, s int) bool {
	for ac := 0; ac <= d.AppleCircle; ac++ {
		for pc := 0; pc <= d.PeachCircle; pc++ {
			for wc := 0; wc <= d.WatermelonCircle; wc++ {
				if ac+pc+wc != c {
					continue
				}
				for as := 0; as <= d.AppleStar; as++ {
					for ps := 0; ps <= d.PeachStar; ps++ {
						for ws := 0; ws <= d.WatermelonStar; ws++ {
							if as+ps+ws != s {
								continue
							}
							ok := (ac > 0 && ps > 0) || (pc > 0 && as > 0)
							if !ok {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

func TestGenerateCandyPrompt(t *testing.T) {
	prompt, ans := GenerateCandyPrompt(false)
	if ans != 21 {
		t.Fatalf("canonical answer should be 21, got %d", ans)
	}
	if !strings.Contains(prompt, "FINAL_ANSWER:") {
		t.Fatal("prompt must instruct the FINAL_ANSWER output contract")
	}
}

func TestGradeCandy(t *testing.T) {
	cases := []struct {
		name     string
		output   string
		expected int
		ok       bool
	}{
		{"strict correct", "some reasoning\nFINAL_ANSWER: 21", 21, true},
		{"strict wrong", "reasoning\nFINAL_ANSWER: 20", 21, false},
		{"loose correct", "推导很久\n最后答案是 21", 21, true},
		{"loose wrong", "推导很久\n最后答案是 22", 21, false},
		{"randomized", "FINAL_ANSWER: 23", 23, true},
	}
	for _, tc := range cases {
		got := GradeProbeOutput(ProbeKindCandy, tc.output, tc.expected)
		if got.Success != tc.ok {
			t.Errorf("%s: expected success=%v got %v (summary: %s)", tc.name, tc.ok, got.Success, got.Summary)
		}
	}
}

func TestGradePelicanSVG(t *testing.T) {
	valid := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="10" cy="10" r="5"/><path d="M0 0 L10 10"/></svg>`
	res := GradeProbeOutput(ProbeKindPelicanSVG, valid, nil)
	if !res.Success || res.Score != 100 {
		t.Fatalf("expected a valid SVG to score 100, got success=%v score=%d summary=%s", res.Success, res.Score, res.Summary)
	}

	fenced := "Here you go:\n```svg\n" + valid + "\n```"
	if res := GradeProbeOutput(ProbeKindPelicanSVG, fenced, nil); !res.Success {
		t.Fatalf("expected fenced SVG to be recovered, got %s", res.Summary)
	}

	if res := GradeProbeOutput(ProbeKindPelicanSVG, "I cannot draw that.", nil); res.Success {
		t.Fatal("a refusal must not be graded as success")
	}
}

func TestGrade3DProbe(t *testing.T) {
	html := `<!DOCTYPE html><html><head></head><body><script src="https://cdn.jsdelivr.net/npm/three/build/three.min.js"></script><script>function loop(){requestAnimationFrame(loop)}loop();</script></body></html>`
	res := GradeProbeOutput(ProbeKindPelican3D, html, nil)
	if !res.Success || res.Score < 80 {
		t.Fatalf("expected a complete 3D page to pass, got success=%v score=%d", res.Success, res.Score)
	}

	if res := GradeProbeOutput(ProbeKindPelican3D, "Here is some pseudocode: render()", nil); res.Success {
		t.Fatal("pseudocode must not pass the 3D probe")
	}
}

func TestProbeDefinitions(t *testing.T) {
	for _, kind := range []ProbeKind{
		ProbeKindCandy,
		ProbeKindPelicanSVG,
		ProbeKindPelican3D,
		ProbeKindCFShip3D,
		ProbeKindQQSpeed3D,
	} {
		def := GetProbeDefinition(kind, false)
		if def.Kind != kind {
			t.Errorf("kind mismatch: asked %s got %s", kind, def.Kind)
		}
		if strings.TrimSpace(def.Prompt) == "" {
			t.Errorf("%s: empty prompt", kind)
		}
		if def.Name == "" || def.Category == "" {
			t.Errorf("%s: missing name/category", kind)
		}
	}
}
