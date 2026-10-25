package eval

import (
	"fmt"
	"math/rand"
	"time"
)

// CandyDistribution defines counts for circles and stars
type CandyDistribution struct {
	AppleCircle      int
	PeachCircle      int
	WatermelonCircle int
	AppleStar        int
	PeachStar        int
	WatermelonStar   int
}

// SolveCandy calculates the exact minimum number of candies needed to guarantee
// having either (Circle Apple + Star Peach) or (Circle Peach + Star Apple),
// taking advantage of the fact that shapes can be distinguished by hand.
//
// Let C be number of circle candies drawn, S be number of star candies drawn.
// Circle pools: AC, PC, WC (TotalCircle = AC + PC + WC)
// Star pools: AS, PS, WS (TotalStar = AS + PS + WS)
//
// A failure (bad outcome) occurs if:
// (No AC || No PS) && (No PC || No AS)
// Which expands to 4 failure scenarios:
// 1) No AC && No PC: adversary only gives WC -> possible if C <= WC
// 2) No PS && No AS: adversary only gives WS -> possible if S <= WS
// 3) No AC && No AS: adversary gives only (PC+WC) in circles and (PS+WS) in stars
//    -> possible if C <= PC + WC AND S <= PS + WS
// 4) No PC && No PS: adversary gives only (AC+WC) in circles and (AS+WS) in stars
//    -> possible if C <= AC + WC AND S <= AS + WS
//
// To guarantee success, (C, S) must prevent ALL 4 failure scenarios:
//   Cond 1: C >= WC + 1
//   Cond 2: S >= WS + 1
//   Cond 3: C >= PC + WC + 1 OR S >= PS + WS + 1
//   Cond 4: C >= AC + WC + 1 OR S >= AS + WS + 1
//
// With 1 <= C <= TotalCircle, 1 <= S <= TotalStar.
// We search for min(C + S) satisfying all 4 conditions.
func SolveCandy(d CandyDistribution) int {
	totalCircle := d.AppleCircle + d.PeachCircle + d.WatermelonCircle
	totalStar := d.AppleStar + d.PeachStar + d.WatermelonStar

	minC := d.WatermelonCircle + 1
	minS := d.WatermelonStar + 1

	c3 := d.PeachCircle + d.WatermelonCircle + 1
	s3 := d.PeachStar + d.WatermelonStar + 1

	c4 := d.AppleCircle + d.WatermelonCircle + 1
	s4 := d.AppleStar + d.WatermelonStar + 1

	bestTotal := 999999

	for c := minC; c <= totalCircle; c++ {
		for s := minS; s <= totalStar; s++ {
			cond3 := (c >= c3 || s >= s3)
			cond4 := (c >= c4 || s >= s4)
			if cond3 && cond4 {
				if c+s < bestTotal {
					bestTotal = c + s
				}
			}
		}
	}

	if bestTotal == 999999 {
		return 0
	}
	return bestTotal
}

// GenerateCandyPrompt returns prompt and expected answer.
// If randomize is false, returns the canonical 21 puzzle.
func GenerateCandyPrompt(randomize bool) (string, int) {
	d := CandyDistribution{
		AppleCircle:      7,
		PeachCircle:      9,
		WatermelonCircle: 8,
		AppleStar:        7,
		PeachStar:        6,
		WatermelonStar:   4,
	}

	if randomize {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		d.AppleCircle = r.Intn(6) + 5      // 5-10
		d.PeachCircle = r.Intn(6) + 5      // 5-10
		d.WatermelonCircle = r.Intn(5) + 4 // 4-8
		d.AppleStar = r.Intn(6) + 5        // 5-10
		d.PeachStar = r.Intn(6) + 5        // 5-10
		d.WatermelonStar = r.Intn(5) + 3   // 3-7
	}

	ans := SolveCandy(d)
	prompt := fmt.Sprintf(`不使用任何外部工具回答以下问题：
在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。现已知不同口味的糖和不同形状的数量统计如下表。参赛者需要在活动前决定摸出的糖果数目，那么，最少取出多少个糖果才能保证手中同时拥有不同形状的苹果味和桃子味的糖？（同时手中有圆形苹果味匹配五角星桃子味糖果，或者有圆形桃子味匹配五角星苹果味糖果都满足要求）

| 口味 | 圆形 | 五角星形 |
| :--- | :--- | :--- |
| 苹果味 | %d | %d |
| 桃子味 | %d | %d |
| 西瓜味 | %d | %d |

请一步步推导。在回答的最后一行必须严格输出最终整数答案，格式为：
FINAL_ANSWER: [数字]`, d.AppleCircle, d.AppleStar, d.PeachCircle, d.PeachStar, d.WatermelonCircle, d.WatermelonStar)

	return prompt, ans
}
