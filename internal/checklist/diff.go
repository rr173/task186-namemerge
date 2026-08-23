// Package checklist 负责已发布清单快照的查询与差异对比：
// 两个观点发布的清单之间，哪些名称的角色发生了变化、
// 哪些名称新增/移出，供分类学家审阅修订影响。
package checklist

import (
	"sort"

	"task186-namemerge/internal/model"
)

// Diff 是两个清单快照的差异摘要。
type Diff struct {
	FromChecklistID string   `json:"from_checklist_id"`
	ToChecklistID   string   `json:"to_checklist_id"`
	RoleChanges     []Change `json:"role_changes"`      // 角色变化（accepted↔synonym↔deferred）
	Added           []string `json:"added"`             // 新增名称（仅在 to）
	Removed         []string `json:"removed"`           // 移出名称（仅在 from）
	AcceptedAdded   []string `json:"accepted_added"`    // 新成为接受名的名称
	AcceptedLost    []string `json:"accepted_lost"`     // 失去接受名地位的名称
	FingerprintSame bool     `json:"fingerprint_same"`  // 两清单指纹是否相同
}

// Change 是一个名称在两个清单中的角色变化。
type Change struct {
	NameID        string `json:"name_id"`
	ScientificName string `json:"scientific_name"`
	FromRole      string `json:"from_role"`
	ToRole        string `json:"to_role"`
}

// Compare 对比两份清单条目。
func Compare(from model.Checklist, fromItems []model.ChecklistItem, to model.Checklist, toItems []model.ChecklistItem) Diff {
	d := Diff{
		FromChecklistID: from.ViewID,
		ToChecklistID:   to.ViewID,
		FingerprintSame: from.Fingerprint == to.Fingerprint,
	}
	fromByID := indexItems(fromItems)
	toByID := indexItems(toItems)

	var added, removed []string
	for id := range toByID {
		if _, ok := fromByID[id]; !ok {
			added = append(added, id)
		}
	}
	for id := range fromByID {
		if _, ok := toByID[id]; !ok {
			removed = append(removed, id)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	d.Added = added
	d.Removed = removed

	var roleChanges []Change
	for id, fromItem := range fromByID {
		toItem, ok := toByID[id]
		if !ok {
			continue
		}
		if fromItem.Role != toItem.Role {
			roleChanges = append(roleChanges, Change{
				NameID: id, ScientificName: toItem.ScientificName,
				FromRole: fromItem.Role, ToRole: toItem.Role,
			})
		}
		if fromItem.Role == "accepted" && toItem.Role != "accepted" {
			d.AcceptedLost = append(d.AcceptedLost, id)
		}
		if fromItem.Role != "accepted" && toItem.Role == "accepted" {
			d.AcceptedAdded = append(d.AcceptedAdded, id)
		}
	}
	sort.Slice(roleChanges, func(i, j int) bool { return roleChanges[i].NameID < roleChanges[j].NameID })
	sort.Strings(d.AcceptedAdded)
	sort.Strings(d.AcceptedLost)
	d.RoleChanges = roleChanges
	return d
}

func indexItems(items []model.ChecklistItem) map[string]model.ChecklistItem {
	m := make(map[string]model.ChecklistItem, len(items))
	for _, it := range items {
		m[it.NameID] = it
	}
	return m
}
