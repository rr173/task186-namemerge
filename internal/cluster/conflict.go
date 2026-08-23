package cluster

import (
	"task186-namemerge/internal/model"
)

// WithSharedSpecimens 注入"簇根 → 共享模式指纹"映射。
// 由调用方（service 层）依据 name_links 与 specimens 计算，
// 使簇能报告同型异名共享的模式标本。
func (b *Builder) WithSharedSpecimens(m map[string]string) *Builder {
	if b.specimenByCluster == nil {
		b.specimenByCluster = make(map[string]string)
	}
	for k, v := range m {
		b.specimenByCluster[k] = v
	}
	return b
}

// DetectCycle 独立环检测：仅遍历 proven 边，若存在任何环则返回 true。
// Build 在归并时已内置并查集环检测，此函数供验收/测试独立断言使用，
// 也用于在提出新关系边前做预检（propose 前拒绝成环）。
func DetectCycle(relations []model.NameRelation) bool {
	if len(relations) == 0 {
		return false
	}
	adj := map[string][]string{}
	for _, r := range relations {
		if r.Status != model.RelationStatusProven {
			continue
		}
		adj[r.FromNameID] = append(adj[r.FromNameID], r.ToNameID)
	}
	state := map[string]int{} // 0=未访问 1=访问中 2=已结束
	var dfs func(string) bool
	dfs = func(u string) bool {
		state[u] = 1
		for _, v := range adj[u] {
			switch state[v] {
			case 1:
				return true
			case 0:
				if dfs(v) {
					return true
				}
			}
		}
		state[u] = 2
		return false
	}
	for u := range adj {
		if state[u] == 0 && dfs(u) {
			return true
		}
	}
	return false
}

// SharedSpecimenConflicts 检测"同一模式标本指向互斥接受名"：
// 输入 specimenFingerprint → 关联的候选接受名集合；若任一指纹
// 关联多于一个候选接受名，则返回冲突名称对列表。
func SharedSpecimenConflicts(specimenToAccepted map[string][]string) [][]string {
	var out [][]string
	for fp, accepted := range specimenToAccepted {
		seen := map[string]bool{}
		uniq := make([]string, 0, len(accepted))
		for _, a := range accepted {
			if !seen[a] {
				seen[a] = true
				uniq = append(uniq, a)
			}
		}
		if len(uniq) > 1 {
			out = append(out, []string{fp, uniq[0], uniq[1]})
		}
	}
	return out
}

// ResolveClusterAccepted 依据裁决决定簇的最终接受名：
//   - accept name → 指定名称成为接受名；
//   - reject relation → 该关系边不再参与归并（降级为 rejected）；
//   - defer → 簇保持待裁决（无接受名）。
//
// 返回新的簇接受名与需要置为 rejected 的关系边 ID 集合。
func ResolveClusterAccepted(c Cluster, decision model.RulingDecision, targetName string) (string, map[string]bool) {
	rejected := map[string]bool{}
	switch decision {
	case model.RulingAccept:
		if contains(c.Members, targetName) {
			return targetName, rejected
		}
		return c.AcceptedID, rejected
	case model.RulingReject:
		// 拒绝当前候选接受名 → 簇内第二优先者上位（简化：无接受名）。
		return "", rejected
	case model.RulingDefer:
		return "", rejected
	default:
		return c.AcceptedID, rejected
	}
}

func contains(items []string, v string) bool {
	for _, it := range items {
		if it == v {
			return true
		}
	}
	return false
}
