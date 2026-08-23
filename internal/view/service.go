// Package view 实现分类观点 (View) 的求值、裁决与发布闭环：
// 在指定规则版本下把名称簇收敛为"接受名 + 异名"清单，
// 冲突进入待裁决；裁决通过后冻结为不可变清单快照。
package view

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"task186-namemerge/internal/cluster"
	"task186-namemerge/internal/model"
)

// Evaluation 是观点求值的结果：簇、冲突、逐名称角色。
type Evaluation struct {
	ViewID   string                 `json:"view_id"`
	Clusters []cluster.Cluster      `json:"clusters"`
	Conflicts []Conflict            `json:"conflicts"`
	Roles    map[string]string      `json:"roles"` // nameID → accepted / synonym / deferred
	EvaluatedAt time.Time           `json:"evaluated_at"`
}

// Conflict 描述一个待裁决冲突。
type Conflict struct {
	Kind       string   `json:"kind"`        // date_unsortable / specimen_conflict / homonym
	MemberIDs  []string `json:"member_ids"`
	Basis      string   `json:"basis"`
}

// Evaluator 汇集求值所需的领域数据。
type Evaluator struct {
	Names      []model.NameRecord
	Publications map[string]model.Publication // nameID → publication
	Relations  []model.NameRelation
	SpecimenToNames map[string][]string // specimen fp → 关联的名称 ID 列表
	HasType    map[string]bool             // nameID → 是否已绑定模式标本
	Rules      model.RuleVersion
}

// Evaluate 执行求值：
//  1. 构造簇（含环检测与共享模式注入）；
//  2. 为每个名称分配角色（簇接受名 → accepted，其余 → synonym，
//     冲突/缺模式 → deferred）；
//  3. 检测同模式互斥冲突（同一模式标本指向多个接受名）。
func (e *Evaluator) Evaluate(viewID string) (*Evaluation, error) {
	b := cluster.NewBuilder(e.Names, e.Publications, e.Relations)
	b.WithSharedSpecimens(e.specimenToRoot())
	clusters, _, err := b.Build()
	if err != nil {
		return nil, err
	}

	ev := &Evaluation{
		ViewID:     viewID,
		Clusters:   clusters,
		Conflicts:  make([]Conflict, 0),
		Roles:      make(map[string]string, len(e.Names)),
		EvaluatedAt: time.Now().UTC(),
	}
	// 日期不可排序冲突。
	for _, c := range clusters {
		if c.AcceptedID == "" {
			ev.Conflicts = append(ev.Conflicts, Conflict{
				Kind: "date_unsortable", MemberIDs: c.Members, Basis: "priority dates unsortable",
			})
		}
	}
	// 角色分配。
	for _, c := range clusters {
		if c.AcceptedID == "" {
			for _, m := range c.Members {
				ev.Roles[m] = "deferred"
			}
			continue
		}
		for _, m := range c.Members {
			// 缺少模式标本 → 证据不足，待裁决（deferred + missing_type 冲突）。
			if !e.HasType[m] {
				ev.Roles[m] = "deferred"
				ev.Conflicts = append(ev.Conflicts, Conflict{
					Kind: "missing_type", MemberIDs: []string{m},
					Basis: "type specimen missing",
				})
				continue
			}
			if m == c.AcceptedID {
				ev.Roles[m] = "accepted"
			} else {
				ev.Roles[m] = "synonym"
			}
		}
	}
	// 未进入任何簇的名称 → deferred（证据不足）。
	inCluster := map[string]bool{}
	for _, c := range clusters {
		for _, m := range c.Members {
			inCluster[m] = true
		}
	}
	for _, n := range e.Names {
		if !inCluster[n.ID] {
			ev.Roles[n.ID] = "deferred"
		}
	}
	// 同一模式标本指向多个"接受名" → specimen_conflict。
	// 注意：同型异名簇共享模式时仅一个接受名，不冲突。
	for fp, names := range e.SpecimenToNames {
		accepted := map[string]bool{}
		for _, nid := range names {
			if ev.Roles[nid] == "accepted" {
				accepted[nid] = true
			}
		}
		if len(accepted) > 1 {
			ids := make([]string, 0, len(accepted))
			for nid := range accepted {
				ids = append(ids, nid)
			}
			ev.Conflicts = append(ev.Conflicts, Conflict{
				Kind: "specimen_conflict", MemberIDs: ids,
				Basis: "same specimen " + fp + " points to mutually exclusive accepted names",
			})
		}
	}
	return ev, nil
}

// specimenToRoot 把模式指纹映射到簇根，供 Builder 注入共享模式。
func (e *Evaluator) specimenToRoot() map[string]string {
	// 简化：取每个名称所属的簇根即名称自身（单例簇时）。
	out := map[string]string{}
	for _, n := range e.Names {
		out[n.ID] = n.ID
	}
	return out
}

// ViewService 是观点领域的编排入口（store 由调用方注入）。
type ViewService struct {
	RuleVersion func() (model.RuleVersion, error)
}

// NewViewService 构造观点服务。
func NewViewService(ruleVersionFn func() (model.RuleVersion, error)) *ViewService {
	return &ViewService{RuleVersion: ruleVersionFn}
}

// ChecklistSnapshot 把求值结果冻结为清单快照（不可变）。
func ChecklistSnapshot(viewID, ruleVersion string, ev *Evaluation, nameByID map[string]model.NameRecord) (model.Checklist, []model.ChecklistItem) {
	items := make([]model.ChecklistItem, 0, len(ev.Roles))
	for nameID, role := range ev.Roles {
		n := nameByID[nameID]
		item := model.ChecklistItem{
			ChecklistID:   viewID + "-snapshot",
			NameID:        nameID,
			ScientificName: n.ScientificName,
			Role:          role,
		}
		if role == "synonym" {
			item.AcceptedNameID = acceptedFor(nameID, ev)
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ScientificName < items[j].ScientificName })
	fp := fingerprint(viewID, ruleVersion, items)
	return model.Checklist{
		ViewID:      viewID,
		RuleVersion: ruleVersion,
		Fingerprint: fp,
		Status:      "frozen",
		CreatedAt:   time.Now().UTC(),
	}, items
}

// acceptedFor 找到包含 nameID 的簇的接受名。
func acceptedFor(nameID string, ev *Evaluation) string {
	for _, c := range ev.Clusters {
		for _, m := range c.Members {
			if m == nameID {
				return c.AcceptedID
			}
		}
	}
	return ""
}

// fingerprint 计算清单快照指纹（序列化条目哈希）。
func fingerprint(viewID, ruleVersion string, items []model.ChecklistItem) string {
	raw, _ := json.Marshal(map[string]any{
		"view": viewID, "rules": ruleVersion, "items": items,
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
