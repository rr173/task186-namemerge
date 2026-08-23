// Package rules 实现命名法规判定规则引擎：
// 优先权 (priority)、合法性 (legitimacy)、同名冲突 (homonym)、
// 拼写变体归并 (orthography)。规则引擎是无状态纯函数集合，
// 输入名称、发表证据、关系边，输出名称状态与冲突标记。
package rules

import (
	"task186-namemerge/internal/evidence"
	"task186-namemerge/internal/model"
)

// Judgment 是一次法规判定的输出：某个名称在其候选簇中的角色。
type Judgment struct {
	NameID         string
	Status         model.NameStatus
	PriorityFirst  bool   // 是否优先权最早
	DateSortable   bool   // 日期是否可排序
	HasValidType   bool   // 是否具备合法模式
	Conflicting    bool   // 是否存在冲突（同模式互斥 / 日期不可排序）
	RejectedByName string // 若被后同名规则驳回，记录压制它的名称
}

// Evaluate 对单个名称应用规则集，产出判定。
// links 用于判断模式关联；conflictingSpecimen 表示同一模式是否
// 被多个互斥候选共享（由簇构造层计算后传入）。
func Evaluate(n model.NameRecord, pub model.Publication, hasType bool, conflictingSpecimen bool) Judgment {
	j := Judgment{NameID: n.ID}
	// 合法性：发表证据必须有效且具备模式。
	status, err := evidence.ValidatePublication(pub, hasType)
	j.HasValidType = hasType
	if err != nil {
		j.Status = model.NameStatusPendingReview
		return j
	}
	switch status {
	case model.PublicationStatusDateConflict:
		j.DateSortable = false
		j.Conflicting = true
		j.Status = model.NameStatusPendingReview
		return j
	case model.PublicationStatusMissingType:
		j.Status = model.NameStatusPendingReview // 缺模式 → 待裁决，暂不判定
		return j
	}
	j.DateSortable = true
	if conflictingSpecimen {
		j.Conflicting = true
		j.Status = model.NameStatusPendingReview
		return j
	}
	j.PriorityFirst = true // 默认；由簇层按日期比较覆盖
	j.Status = model.NameStatusLegitimate
	return j
}

// ComparePriority 比较两个名称的发表日期，决定优先权。
// 返回 earlier 是否来自 a；无法排序返回 ok=false。
func ComparePriority(a, b model.Publication) (earlierIsA bool, ok bool) {
	return evidence.Sortable(a, b)
}

// IsHomonym 判断 b 是否为 a 的后同名 (later homonym)：
// 拼写相同（归一化键相等）但作者不同，且 b 发表更晚。
func IsHomonym(a, b model.NameRecord, pubA, pubB model.Publication) bool {
	if a.OrthographicKey != b.OrthographicKey {
		return false
	}
	if normAuthor(a.Authors) == normAuthor(b.Authors) {
		return false
	}
	earlierIsA, ok := evidence.Sortable(pubA, pubB)
	if !ok {
		return false
	}
	return earlierIsA // a 更早 → b 是后同名
}

func normAuthor(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			out = append(out, r)
		}
	}
	return string(out)
}

// ApplyLegitimacy 后处理：被后同名规则驳回的名称置为 illegitimate。
func ApplyLegitimacy(j *Judgment, laterHomonym bool) {
	if laterHomonym {
		j.Status = model.NameStatusIllegitimate
		j.Conflicting = true
	}
}
