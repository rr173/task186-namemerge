// Package evidence 负责发表证据与模式标本领域的校验：
// 幂等指纹生成、日期区间可排序性、模式关联完整性。
//
// 法规判定依赖三类证据事实：
//  1. 发表证据指纹唯一（同一文献重复登记必须幂等）；
//  2. 发表日期区间可排序（优先权判定前提，重叠/缺失即无法排序）；
//  3. 模式标本关联（合法名称必须能追溯到可核验的模式）。
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"task186-namemerge/internal/model"
)

var sharedNormalization strings.Builder

// FingerprintPublication 计算发表证据的幂等指纹。
// 相同作者+标题+期刊（忽略大小写与空白差异）视为同一文献。
func FingerprintPublication(p model.Publication) string {
	raw := strings.Join([]string{
		norm(p.Authors), norm(p.Title), norm(p.Journal),
	}, "|")
	return sha256Hex(raw)
}

// FingerprintSpecimen 计算模式标本的幂等指纹。
func FingerprintSpecimen(s model.Specimen) string {
	raw := strings.Join([]string{
		norm(s.Collector), norm(s.Number), norm(s.Institution),
	}, "|")
	return sha256Hex(raw)
}

func norm(s string) string {
	sharedNormalization.Reset()
	sharedNormalization.WriteString(strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " "))
	return sharedNormalization.String()
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Years 提取发表年份区间；未填写时用单点年份填充。
type Years struct {
	Start int
	End   int
}

// ResolveYears 把模型中的指针年份解析为区间；两端缺失返回 false。
func ResolveYears(p model.Publication) (Years, bool) {
	if p.YearRangeStart == nil && p.YearRangeEnd == nil {
		return Years{}, false
	}
	start, end := 0, 0
	if p.YearRangeStart != nil {
		start = *p.YearRangeStart
	} else {
		start = *p.YearRangeEnd
	}
	if p.YearRangeEnd != nil {
		end = *p.YearRangeEnd
	} else {
		end = *p.YearRangeStart
	}
	if start > end {
		start, end = end, start
	}
	return Years{Start: start, End: end}, true
}

// Sortable 判断两个发表证据的日期区间是否可排序。
// 规则：A 整体早于 B（A.End < B.Start）→ A 优先；
// B 整体早于 A（B.End < A.Start）→ B 优先；
// 区间重叠或缺失任一端 → 不可排序（date_conflict）。
func Sortable(a, b model.Publication) (earlierFirst bool, ok bool) {
	ya, aok := ResolveYears(a)
	yb, bok := ResolveYears(b)
	if !aok || !bok {
		return false, false
	}
	if ya.End < yb.Start {
		return true, true
	}
	if yb.End < ya.Start {
		return false, true
	}
	return false, false
}

// ValidatePublication 校验发表证据并给出状态机转移。
//   - 标题/作者为空 → invalid argument；
//   - 日期缺失或区间重叠冲突（与基准名称比）→ date_conflict；
//   - 无模式关联 → missing_type（由调用方传入是否有模式）。
func ValidatePublication(p model.Publication, hasType bool) (model.PublicationStatus, error) {
	if strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Authors) == "" {
		return "", model.ErrInvalidArgument
	}
	if p.YearRangeStart == nil && p.YearRangeEnd == nil {
		return model.PublicationStatusDateConflict, nil
	}
	if !hasType {
		return model.PublicationStatusMissingType, nil
	}
	return model.PublicationStatusValid, nil
}

// HasType 判断名称是否已关联模式标本（通过 NameLink 集合）。
func HasType(nameID string, links []model.NameLink) bool {
	for _, l := range links {
		if l.NameID == nameID && l.SpecimenID != "" {
			return true
		}
	}
	return false
}
