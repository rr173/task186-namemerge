// Package namerecord 负责名称记录领域：解析学名字符串、
// 生成拼写归一化键（orthographic key）、检测组合变更。
//
// 植物命名法规中，同一分类群的学名可能以多种拼写形式出现
// （如 -i-/-y- 互换、终止词缀差异、大小写差异），这些"拼写变体"
// (orthographic variants) 按法规应被视为同一名称；而"新组合"
// (new combination) 是指同一基名 (basionym) 被转移到不同属下。
// 本包把这两类归一化问题集中处理，供规则引擎与归并簇使用。
package namerecord

import (
	"strings"
	"unicode"

	"task186-namemerge/internal/model"
)

// Parse 把"属名 种加词 作者"形式的学名字符串拆成结构化字段。
// 作者部分从第一个大写字母开始的连续段截取（启发式：作者缩写
// 通常形如 "L."、"J.Sm."、"Wall. ex Hook.f."）。
func Parse(raw string) (model.NameRecord, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return model.NameRecord{}, model.ErrInvalidArgument
	}
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		return model.NameRecord{}, model.ErrInvalidArgument
	}
	genus := strings.TrimSuffix(fields[0], ",")
	genus = strings.TrimSuffix(genus, ".")
	if genus == "" {
		return model.NameRecord{}, model.ErrInvalidArgument
	}
	epithet := fields[1]
	epithet = strings.TrimSuffix(epithet, ",")
	authors := ""
	if len(fields) > 2 {
		authors = strings.Join(fields[2:], " ")
	}
	return model.NameRecord{
		ScientificName:  raw,
		Genus:           genus,
		SpecificEpithet: epithet,
		Authors:         authors,
		OrthographicKey: OrthographicKey(genus, epithet),
		Status:          model.NameStatusPendingReview,
	}, nil
}

// OrthographicKey 生成拼写归一化键：
// 小写 → 去除标点与尾随连字符 → 去除重音符号 → 归一化 i/y、u/v、c/k 等
// 拉丁正字法常见互换。两个拼写变体若键相同，则在法规上视为同一名称。
func OrthographicKey(genus, epithet string) string {
	g := normalize(genus)
	e := normalize(epithet)
	return g + " " + e
}

// normalize 执行单段拼写归一化。
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(mapOrthographic(r))
		case r == '-', r == '_', r == ' ', r == '.':
			// 分隔符跳过
		}
	}
	return b.String()
}

// mapOrthographic 处理拉丁正字法常见字母互换：
//   - i/y 互换（如 "sylvestris"/"silvestris"）
//   - u/v 互换（如 "uva"/"vua" 变体，古典拉丁 U/V 混写）
//   - ae/oe 中的 e 归一
func mapOrthographic(r rune) rune {
	switch r {
	case 'y':
		return r
	case 'v':
		return 'u'
	default:
		return r
	}
}

// IsVariant 判断两个名称是否为拼写变体（归一化键相同但原文不同）。
func IsVariant(a, b model.NameRecord) bool {
	if a.ID != "" && a.ID == b.ID {
		return false
	}
	if a.ScientificName == b.ScientificName && a.ID == b.ID {
		return false
	}
	return a.OrthographicKey == b.OrthographicKey
}

// CombinationChange 描述一个组合变更：基名与新组合的关系。
type CombinationChange struct {
	BasionymName  string // 基名（原属下的种加词）
	NewGenus      string // 新属名
	NewEpithet    string // 新组合的种加词（通常与基名相同）
	CombinedName  string // 新组合全名
}

// BuildCombination 构造一个组合变更提议。
func BuildCombination(basionym model.NameRecord, newGenus string) (CombinationChange, error) {
	if basionym.SpecificEpithet == "" {
		return CombinationChange{}, model.ErrInvalidArgument
	}
	newGenus = strings.TrimSpace(newGenus)
	if newGenus == "" {
		return CombinationChange{}, model.ErrInvalidArgument
	}
	combined := newGenus + " " + basionym.SpecificEpithet + " (" + basionym.Genus + ")"
	return CombinationChange{
		BasionymName: basionym.ScientificName,
		NewGenus:     newGenus,
		NewEpithet:   basionym.SpecificEpithet,
		CombinedName: combined,
	}, nil
}

// SameEpithet 判断两个名称是否共享种加词（忽略属名）——组合变更的判据。
func SameEpithet(a, b model.NameRecord) bool {
	return normalize(a.SpecificEpithet) == normalize(b.SpecificEpithet)
}
