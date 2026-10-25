package eval

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

var (
	finalAnswerRegex = regexp.MustCompile(`(?i)FINAL_ANSWER:\s*(\d+)`)
	looseNumberRegex = regexp.MustCompile(`\b\d+\b`)
	svgTagRegex      = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`)
	htmlTagRegex     = regexp.MustCompile(`(?is)<!DOCTYPE html>.*?</html>|<html[^>]*>.*?</html>`)
)

type ProbeKind string

const (
	ProbeKindCandy       ProbeKind = "candy"
	ProbeKindPelicanSVG  ProbeKind = "pelican_svg"
	ProbeKindPelican3D   ProbeKind = "pelican_3d"
	ProbeKindCFShip3D    ProbeKind = "cf_ship_3d"
	ProbeKindQQSpeed3D   ProbeKind = "qq_speed_3d"
)

// ProbeDefinition holds metadata and prompt for each eval task
type ProbeDefinition struct {
	Kind        ProbeKind `json:"kind"`
	Name        string    `json:"name"`
	Category    string    `json:"category"` // "math", "svg", "3d"
	Description string    `json:"description"`
	Prompt      string    `json:"prompt"`
	Expected    any       `json:"expected,omitempty"`
}

// GetProbeDefinition returns the task prompt and expectations
func GetProbeDefinition(kind ProbeKind, randomize bool) ProbeDefinition {
	switch kind {
	case ProbeKindPelicanSVG:
		return ProbeDefinition{
			Kind:        ProbeKindPelicanSVG,
			Name:        "🚲 鹈鹕骑车 (2D SVG)",
			Category:    "svg",
			Description: "经典 Simon Willison 空间与矢量绘图基准，测试模型的空间拓扑关系与手写 SVG 能力",
			Prompt:      "Generate an SVG of a pelican riding a bicycle. Output only the valid, well-structured SVG code without any markdown fencing or extra explanation if possible.",
		}
	case ProbeKindPelican3D:
		return ProbeDefinition{
			Kind:        ProbeKindPelican3D,
			Name:        "🚲 鹈鹕骑车 (3D 游戏)",
			Category:    "3d",
			Description: "Claude Opus 5.5 级极限制工测试：单文件程序化生成 3D 互动页面 (Three.js)",
			Prompt: `生成一个鹈鹕骑自行车的 3D 页面，尽可能把你所有的能力全部都用上。
要求：
1. 包含完整的 HTML、CSS 和 JavaScript（单文件完整可运行 HTML，使用 Three.js CDN 资源）。
2. 不依赖任何外部图片、3D模型或音频文件，模型、材质、动画全部使用纯代码程序化构建。
3. 包含键盘交互控制（如前后移动、跳跃、视角切换等）与平滑动画渲染循环。
4. 代码必须完整无截断，可直接在浏览器中打开试玩。`,
		}
	case ProbeKindCFShip3D:
		return ProbeDefinition{
			Kind:        ProbeKindCFShip3D,
			Name:        "🔫 穿越火线·运输船 (3D FPS)",
			Category:    "3d",
			Description: "还原运输船甲板/集装箱/管道与射击物理对战 (Three.js 单文件 HTML)",
			Prompt: `尽可能真实的还原穿越火线中的运输船地图，我需要一个真实的枪战游戏，生成一个 3D 页面，尽可能发挥你的所有能力。
要求：
1. 单文件完整可运行 HTML（使用 Three.js CDN），不引用外部模型或贴图素材，全部通过纯代码程序化生成。
2. 还原长条形甲板、集装箱掩体、两端出生点。
3. 实现第一人称视角移动、准星、开火、弹道反馈与简单的敌方 AI 目标。
4. 代码必须完整无截断，可直接在浏览器中运行。`,
		}
	case ProbeKindQQSpeed3D:
		return ProbeDefinition{
			Kind:        ProbeKindQQSpeed3D,
			Name:        "🏎️ QQ 飞车 (3D 漂移)",
			Category:    "3d",
			Description: "还原经典赛道、赛车物理、Shift 漂移与氮气加速 (Three.js 单文件 HTML)",
			Prompt: `尽可能真实地还原 QQ 飞车中的游戏地图。我需要一个真实的 QQ 飞车游戏，包括游戏的各种键位以及漂移玩法，生成一个 3D 页面，尽可能发挥你的所有能力。
要求：
1. 单文件完整可运行 HTML（使用 Three.js CDN），纯代码程序化构建赛道、赛车与粒子特效。
2. 包含方向控制、Shift 漂移手感、漂移集气与氮气加速逻辑。
3. 代码必须完整无截断，可直接在浏览器中运行试玩。`,
		}
	case ProbeKindCandy:
		fallthrough
	default:
		prompt, ans := GenerateCandyPrompt(randomize)
		return ProbeDefinition{
			Kind:        ProbeKindCandy,
			Name:        "🍬 动态糖果最坏情况 (逻辑推理)",
			Category:    "math",
			Description: "最恶劣条件下的抽屉原理组合推理题，强依赖完整思维链 (Reasoning Tokens)",
			Prompt:      prompt,
			Expected:    ans,
		}
	}
}

// EvalAnalysisResult contains parsed assessment of the model output
type EvalAnalysisResult struct {
	Success         bool   `json:"success"`
	Score           int    `json:"score"` // 0 - 100
	ExtractedData   string `json:"extracted_data,omitempty"`
	Summary         string `json:"summary"`
	HTMLCode        string `json:"html_code,omitempty"`
	SVGCode         string `json:"svg_code,omitempty"`
}

// GradeProbeOutput evaluates model output string against the specific probe criteria
func GradeProbeOutput(kind ProbeKind, rawOutput string, expected any) EvalAnalysisResult {
	cleaned := strings.TrimSpace(rawOutput)

	switch kind {
	case ProbeKindCandy:
		expectedInt, ok := expected.(int)
		if !ok {
			expectedInt = 21
		}
		// 1. Try strict FINAL_ANSWER
		matches := finalAnswerRegex.FindStringSubmatch(cleaned)
		extracted := ""
		if len(matches) > 1 {
			extracted = matches[1]
		} else {
			// Fallback: search last line for numbers
			lines := strings.Split(cleaned, "\n")
			for i := len(lines) - 1; i >= 0; i-- {
				l := strings.TrimSpace(lines[i])
				if l == "" {
					continue
				}
				numMatches := looseNumberRegex.FindAllString(l, -1)
				if len(numMatches) > 0 {
					extracted = numMatches[len(numMatches)-1]
					break
				}
			}
		}

		isCorrect := extracted == fmt.Sprintf("%d", expectedInt)
		summary := fmt.Sprintf("期望答案: %d, 提取答案: %s", expectedInt, extracted)
		score := 0
		if isCorrect {
			score = 100
			summary = fmt.Sprintf("推导完全正确 (答案 %d)", expectedInt)
		}

		return EvalAnalysisResult{
			Success:       isCorrect,
			Score:         score,
			ExtractedData: extracted,
			Summary:       summary,
		}

	case ProbeKindPelicanSVG:
		// Extract <svg>...</svg>
		svgMatch := svgTagRegex.FindString(cleaned)
		if svgMatch == "" {
			// Try cleaning code fences
			cleanFence := strings.TrimPrefix(cleaned, "```xml")
			cleanFence = strings.TrimPrefix(cleanFence, "```svg")
			cleanFence = strings.TrimPrefix(cleanFence, "```html")
			cleanFence = strings.TrimPrefix(cleanFence, "```")
			cleanFence = strings.TrimSuffix(cleanFence, "```")
			cleanFence = strings.TrimSpace(cleanFence)
			if strings.HasPrefix(cleanFence, "<svg") && strings.HasSuffix(cleanFence, "</svg>") {
				svgMatch = cleanFence
			}
		}

		if svgMatch == "" {
			return EvalAnalysisResult{
				Success: false,
				Score:   0,
				Summary: "未提取到有效的 <svg> 根标签",
			}
		}

		// XML Well-formed test
		var parsed struct{}
		err := xml.Unmarshal([]byte(svgMatch), &parsed)
		if err != nil {
			return EvalAnalysisResult{
				Success:  false,
				Score:    30,
				SVGCode:  svgMatch,
				Summary:  fmt.Sprintf("SVG 语法不合规 / 未正常闭合: %v", err),
			}
		}

		// Feature inspection
		hasCircle := strings.Contains(svgMatch, "<circle")
		hasPath := strings.Contains(svgMatch, "<path")
		score := 70
		if hasCircle && hasPath {
			score = 100
		}

		return EvalAnalysisResult{
			Success:  true,
			Score:    score,
			SVGCode:  svgMatch,
			Summary:  fmt.Sprintf("合法 SVG (大小: %d 字符, 包含基本几何元素)", len(svgMatch)),
		}

	case ProbeKindPelican3D, ProbeKindCFShip3D, ProbeKindQQSpeed3D:
		// Extract HTML
		htmlMatch := htmlTagRegex.FindString(cleaned)
		if htmlMatch == "" {
			cleanFence := strings.TrimPrefix(cleaned, "```html")
			cleanFence = strings.TrimPrefix(cleanFence, "```")
			cleanFence = strings.TrimSuffix(cleanFence, "```")
			cleanFence = strings.TrimSpace(cleanFence)
			if strings.Contains(cleanFence, "<html") || strings.Contains(cleanFence, "<!DOCTYPE") {
				htmlMatch = cleanFence
			}
		}

		if htmlMatch == "" {
			return EvalAnalysisResult{
				Success: false,
				Score:   10,
				Summary: "未能提取到完整的 HTML 单文件网页",
			}
		}

		hasThreeJS := strings.Contains(htmlMatch, "three")
		hasRenderLoop := strings.Contains(htmlMatch, "requestAnimationFrame")
		hasScript := strings.Contains(htmlMatch, "<script")

		score := 40
		if hasScript {
			score += 20
		}
		if hasThreeJS {
			score += 20
		}
		if hasRenderLoop {
			score += 20
		}

		return EvalAnalysisResult{
			Success:  score >= 80,
			Score:    score,
			HTMLCode: htmlMatch,
			Summary:  fmt.Sprintf("3D 单文件工程页面 (代码量: %d 字符, Three.js 状态: %v, 渲染循环: %v)", len(htmlMatch), hasThreeJS, hasRenderLoop),
		}
	}

	return EvalAnalysisResult{
		Success: false,
		Score:   0,
		Summary: "未知评测类型",
	}
}
