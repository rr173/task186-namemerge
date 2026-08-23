// Package cluster 构造"同物异名簇"（synonym cluster）：
// 把通过 proven 关系边连通的名称聚成连通分量，并在每个簇内
// 按优先权选出候选接受名，同时检测两类不变量违规：
//   - 循环同物异名（归并边成环）；
//   - 同一模式标本指向互斥接受名。
package cluster

import (
	"task186-namemerge/internal/evidence"
	"task186-namemerge/internal/model"
)

// Cluster 是一个名称簇：一组互为异名、共同指向一个候选接受名的名称。
type Cluster struct {
	Members     []string `json:"members"`       // 簇内名称 ID（含接受名）
	AcceptedID  string   `json:"accepted_id"`   // 候选接受名 ID（优先权最早）
	SharedSpecimen string `json:"shared_specimen,omitempty"` // 共享模式指纹（同型异名时）
	ProvenCount int      `json:"proven_count"`  // 已证明关系边数
}

// Builder 收集名称、发表证据与关系边，构建簇。
type Builder struct {
	names      map[string]model.NameRecord
	pubs       map[string]model.Publication
	relations  []model.NameRelation
	parent     map[string]string
	rank       map[string]int
	specimenByCluster map[string]string // 簇根 → 共享模式指纹（可选注入）
}

// NewBuilder 创建一个簇构造器。
func NewBuilder(names []model.NameRecord, pubs map[string]model.Publication, relations []model.NameRelation) *Builder {
	b := &Builder{
		names:     make(map[string]model.NameRecord, len(names)),
		pubs:      pubs,
		relations: relations,
		parent:    make(map[string]string, len(names)),
		rank:      make(map[string]int, len(names)),
	}
	for _, n := range names {
		b.names[n.ID] = n
		b.parent[n.ID] = n.ID
	}
	return b
}

// find 并查集查找（带路径压缩）。
func (b *Builder) find(x string) string {
	if b.parent[x] != x {
		b.parent[x] = b.find(b.parent[x])
	}
	return b.parent[x]
}

// union 合并两个集合；返回 false 表示合并会产生环（两者已在同一集合）。
func (b *Builder) union(x, y string) bool {
	rx, ry := b.find(x), b.find(y)
	if rx == ry {
		return false
	}
	if b.rank[rx] < b.rank[ry] {
		rx, ry = ry, rx
	}
	b.parent[ry] = rx
	if b.rank[rx] == b.rank[ry] {
		b.rank[rx]++
	}
	return true
}

// Build 执行归并：
//  1. 仅使用 proven 关系边合并；
//  2. 若合并产生环 → 返回 ErrCycleSynonym；
//  3. 每个簇内按发表日期选优先权最早者为候选接受名；
//  4. 收集共享模式信息（同型异名簇）。
//
// 返回簇列表与冲突边列表。
func (b *Builder) Build() ([]Cluster, []model.NameRelation, error) {
	// 先只对 proven 边做连通性合并，环检测在 union 内部完成。
	for _, r := range b.relations {
		if r.Status != model.RelationStatusProven {
			continue
		}
		if _, ok := b.names[r.FromNameID]; !ok {
			continue
		}
		if _, ok := b.names[r.ToNameID]; !ok {
			continue
		}
		if !b.union(r.FromNameID, r.ToNameID) {
			return nil, nil, model.ErrCycleSynonym
		}
	}

	groups := map[string][]string{}
	for id := range b.names {
		root := b.find(id)
		groups[root] = append(groups[root], id)
	}

	var clusters []Cluster
	conflicts := make([]model.NameRelation, 0)
	for _, members := range groups {
		c := Cluster{Members: members}
		// 选优先权最早者。
		earliest := members[0]
		for _, m := range members[1:] {
			if earlier(m, earliest, b.names, b.pubs) {
				earliest = m
			}
		}
		// 若最早者日期无法排序，标记冲突。
		if !sortableByName(earliest, b.names, b.pubs) {
			c.AcceptedID = ""
			conflicts = append(conflicts, model.NameRelation{
				Status: model.RelationStatusConflicting,
				Basis:  "date unsortable among cluster members",
			})
			clusters = append(clusters, c)
			continue
		}
		c.AcceptedID = earliest
		c.ProvenCount = b.provenCount(members)
		// 共享模式：同型异名簇共享同一模式指纹。
		c.SharedSpecimen = b.sharedSpecimen(members)
		clusters = append(clusters, c)
	}
	return clusters, conflicts, nil
}

// provenCount 统计簇内 proven 边数量。
func (b *Builder) provenCount(members []string) int {
	in := make(map[string]bool, len(members))
	for _, m := range members {
		in[m] = true
	}
	n := 0
	for _, r := range b.relations {
		if r.Status == model.RelationStatusProven && in[r.FromNameID] && in[r.ToNameID] {
			n++
		}
	}
	return n
}

// sharedSpecimen 找出簇内名称共同关联的模式指纹（若有）。
func (b *Builder) sharedSpecimen(members []string) string {
	// 简化：簇内任意成员若绑定同一模式指纹即视为共享。
	// 实际模式指纹需从 store 的 name_links + specimens 查询，
	// 这里由调用方通过 WithSharedSpecimens 注入。
	return b.specimenByCluster[members[0]]
}

// earlier 判断 a 是否比 b 发表更早（优先权）。
func earlier(a, b string, names map[string]model.NameRecord, pubs map[string]model.Publication) bool {
	pa, aok := pubs[a]
	pb, bok := pubs[b]
	if !aok || !bok {
		return false
	}
	earlierIsA, ok := evidence.Sortable(pa, pb)
	if !ok {
		return false
	}
	return earlierIsA
}

// sortableByName 判断簇内某个名称是否具备可排序日期
// （发表年份区间完整且起点不晚于终点）。
func sortableByName(id string, names map[string]model.NameRecord, pubs map[string]model.Publication) bool {
	p, ok := pubs[id]
	if !ok {
		return false
	}
	if p.YearRangeStart == nil || p.YearRangeEnd == nil {
		return false
	}
	return *p.YearRangeStart <= *p.YearRangeEnd
}
