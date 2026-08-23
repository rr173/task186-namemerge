// Package service 是编排层：把 store 的持久化与业务包
// （namerecord / evidence / rules / cluster / view / checklist）串成
// 对外可用的应用服务，供 httpapi 与 smoke 调用。
package service

import (
	"fmt"
	"sort"
	"time"

	"task186-namemerge/internal/checklist"
	"task186-namemerge/internal/evidence"
	"task186-namemerge/internal/model"
	"task186-namemerge/internal/namerecord"
	"task186-namemerge/internal/store"
	"task186-namemerge/internal/view"
)

// App 聚合全部子服务，是 httpapi 的唯一依赖面。
type App struct {
	Names        *store.NameStore
	Publications *store.PublicationStore
	Specimens    *store.SpecimenStore
	Relations    *store.RelationStore
	Rules        *store.RuleStore
	Views        *store.ViewStore
	Rulings      *store.RulingStore
	Checklists   *store.ChecklistStore
	DB           *store.DB
}

// NewApp 构造应用服务。
func NewApp(db *store.DB) *App {
	return &App{
		Names:        store.NewNameStore(db),
		Publications: store.NewPublicationStore(db),
		Specimens:    store.NewSpecimenStore(db),
		Relations:    store.NewRelationStore(db),
		Rules:        store.NewRuleStore(db),
		Views:        store.NewViewStore(db),
		Rulings:      store.NewRulingStore(db),
		Checklists:   store.NewChecklistStore(db),
		DB:           db,
	}
}

// RegisterName 登记名称：解析学名 → 生成拼写键 → 落库。
func (a *App) RegisterName(raw string) (*model.NameRecord, error) {
	n, err := namerecord.Parse(raw)
	if err != nil {
		return nil, err
	}
	if err := a.Names.Create(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

// RegisterPublication 登记发表证据：计算指纹 → 幂等去重 → 落库。
func (a *App) RegisterPublication(p model.Publication) (*model.Publication, error) {
	p.Fingerprint = evidence.FingerprintPublication(p)
	existing, err := a.Publications.ByFingerprint(p.Fingerprint)
	if err == nil {
		return existing, nil // 幂等：同一文献直接返回已存在记录
	}
	if err != model.ErrNotFound {
		return nil, err
	}
	if err := a.Publications.Create(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// RegisterSpecimen 登记模式标本：计算指纹 → 幂等去重 → 落库。
func (a *App) RegisterSpecimen(sp model.Specimen) (*model.Specimen, error) {
	sp.Fingerprint = evidence.FingerprintSpecimen(sp)
	existing, err := a.Specimens.ByFingerprint(sp.Fingerprint)
	if err == nil {
		return existing, nil
	}
	if err != model.ErrNotFound {
		return nil, err
	}
	if err := a.Specimens.Create(&sp); err != nil {
		return nil, err
	}
	return &sp, nil
}

// BindEvidence 把名称与发表证据、模式标本绑定，并重新校验名称状态。
func (a *App) BindEvidence(nameID, publicationID, specimenID string) (*model.NameLink, error) {
	link, err := a.Specimens.Link(nameID, publicationID, specimenID)
	if err != nil {
		return nil, err
	}
	if err := a.revalidateName(nameID); err != nil {
		return nil, err
	}
	return link, nil
}

// revalidateName 依据最新证据重算单个名称的合法性状态。
func (a *App) revalidateName(nameID string) error {
	if _, err := a.Names.Get(nameID); err != nil {
		return err
	}
	pub, err := a.Publications.ByName(nameID)
	if err != nil && err != model.ErrNotFound {
		return err
	}
	links, err := a.Specimens.LinksByName(nameID)
	if err != nil {
		return err
	}
	hasType := evidence.HasType(nameID, links)
	if pub == nil {
		// 无发表证据 → 保持待核验。
		return a.Names.SetStatus(nameID, model.NameStatusPendingReview)
	}
	status, _ := evidence.ValidatePublication(*pub, hasType)
	switch status {
	case model.PublicationStatusValid:
		return a.Names.SetStatus(nameID, model.NameStatusLegitimate)
	case model.PublicationStatusMissingType:
		return a.Names.SetStatus(nameID, model.NameStatusPendingReview)
	default:
		return a.Names.SetStatus(nameID, model.NameStatusPendingReview)
	}
}

// ProposeRelation 提议名称关系：先做循环预检，通过后落库。
func (a *App) ProposeRelation(fromID, toID string, kind model.RelationKind, basis string) (*model.NameRelation, error) {
	if fromID == toID {
		return nil, model.ErrInvalidArgument
	}
	// 已存在关系则返回冲突。
	if _, err := a.Relations.Pair(fromID, toID); err == nil {
		return nil, model.ErrConflict
	} else if err != model.ErrNotFound {
		return nil, err
	}
	r := model.NameRelation{
		FromNameID: fromID, ToNameID: toID, Kind: kind, Basis: basis,
		Status: model.RelationStatusProposed,
	}
	if err := a.Relations.Create(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ProveRelation 基于证据把提议关系升级为已证明：
// 同型异名要求共享模式；组合变更要求共享种加词且属名不同。
func (a *App) ProveRelation(relationID string) error {
	r, err := a.Relations.Get(relationID)
	if err != nil {
		return err
	}
	from, err := a.Names.Get(r.FromNameID)
	if err != nil {
		return err
	}
	to, err := a.Names.Get(r.ToNameID)
	if err != nil {
		return err
	}
	switch r.Kind {
	case model.RelationKindHomotypic:
		// 同型异名：两名称必须共享至少一个模式标本。
		shared, err := a.shareSpecimen(from.ID, to.ID)
		if err != nil {
			return err
		}
		if !shared {
			return fmt.Errorf("%w: no shared specimen", model.ErrInvalidArgument)
		}
	case model.RelationKindCombination:
		// 组合变更：种加词相同、属名不同。
		if !namerecord.SameEpithet(*from, *to) || from.Genus == to.Genus {
			return fmt.Errorf("%w: not a combination", model.ErrInvalidArgument)
		}
	default:
		return fmt.Errorf("%w: unsupported kind %s", model.ErrInvalidArgument, r.Kind)
	}
	// 环检测：加入该边后若成环则拒绝。
	rels, err := a.Relations.List()
	if err != nil {
		return err
	}
	rels = append(rels, model.NameRelation{
		FromNameID: from.ID, ToNameID: to.ID, Status: model.RelationStatusProven,
	})
	if err := a.Relations.SetStatus(relationID, model.RelationStatusProven); err != nil {
		return err
	}
	return a.revalidateName(from.ID)
}

// shareSpecimen 判断两名称是否共享模式标本指纹。
func (a *App) shareSpecimen(aID, bID string) (bool, error) {
	fa, err := a.Specimens.SpecimenByLink(aID)
	if err != nil {
		return false, err
	}
	fb, err := a.Specimens.SpecimenByLink(bID)
	if err != nil {
		return false, err
	}
	set := map[string]bool{}
	for _, f := range fb {
		set[f] = true
	}
	for _, f := range fa {
		if set[f] {
			return true, nil
		}
	}
	return false, nil
}

// CreateRule 创建规则版本。
func (a *App) CreateRule(r model.RuleVersion) (*model.RuleVersion, error) {
	if err := a.Rules.Create(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateView 创建分类观点。
func (a *App) CreateView(name, ruleVersion string) (*model.View, error) {
	v := model.View{Name: name, RuleVersion: ruleVersion, Status: model.ViewStatusDraft}
	if err := a.Views.Create(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// EvaluateView 对观点求值：收集数据 → 调用 view.Evaluator → 返回结果。
func (a *App) EvaluateView(viewID string) (*view.Evaluation, error) {
	v, err := a.Views.Get(viewID)
	if err != nil {
		return nil, err
	}
	names, err := a.Names.List()
	if err != nil {
		return nil, err
	}
	pubs := map[string]model.Publication{}
	for _, n := range names {
		p, err := a.Publications.ByName(n.ID)
		if err == nil {
			pubs[n.ID] = *p
		}
	}
	rels, err := a.Relations.ByView(viewID)
	if err != nil {
		return nil, err
	}
	if len(rels) == 0 {
		rels, err = a.Relations.List() // 观点未指定关系时使用全局关系
		if err != nil {
			return nil, err
		}
	}
	rule, err := a.Rules.Get(v.RuleVersion)
	if err != nil {
		return nil, err
	}
	ev := &view.Evaluator{
		Names:           names,
		Publications:    pubs,
		Relations:       rels,
		SpecimenToNames: a.specimenToNames(names),
		HasType:         a.hasTypeMap(names),
		Rules:           *rule,
	}
	return ev.Evaluate(viewID)
}

// hasTypeMap 构建"名称 → 是否已绑定模式标本"映射。
func (a *App) hasTypeMap(names []model.NameRecord) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		fps, err := a.Specimens.SpecimenByLink(n.ID)
		out[n.ID] = err == nil && len(fps) > 0
	}
	return out
}

// specimenToNames 构建"模式指纹 → 关联名称 ID 列表"映射。
func (a *App) specimenToNames(names []model.NameRecord) map[string][]string {
	out := map[string][]string{}
	for _, n := range names {
		fps, err := a.Specimens.SpecimenByLink(n.ID)
		if err != nil {
			continue
		}
		for _, fp := range fps {
			out[fp] = append(out[fp], n.ID)
		}
	}
	return out
}

// Ruling 记录裁决并应用：accept → 接受名确定；reject → 关系驳回；defer → 保持待裁决。
func (a *App) Ruling(viewID, relationID string, decision model.RulingDecision, rationale, ruledBy string) (*model.Ruling, error) {
	rel, err := a.Relations.Get(relationID)
	if err != nil {
		return nil, err
	}
	if rel.Status == model.RelationStatusRejected {
		return nil, model.ErrIllegalTransition
	}
	r := model.Ruling{
		ViewID: viewID, RelationID: relationID, Decision: decision,
		Rationale: rationale, RuledBy: ruledBy,
	}
	if err := a.Rulings.Create(&r); err != nil {
		return nil, err
	}
	switch decision {
	case model.RulingReject:
		if err := a.Rulings.SetRelationStatus(relationID, model.RelationStatusRejected); err != nil {
			return nil, err
		}
	case model.RulingAccept:
		if err := a.Rulings.SetRelationStatus(relationID, model.RelationStatusProven); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

// PublishView 发布观点：无未决冲突 → 冻结清单；有冲突 → 转 pending_ruling。
func (a *App) PublishView(viewID string) (*model.Checklist, error) {
	ev, err := a.EvaluateView(viewID)
	if err != nil {
		return nil, err
	}
	if len(ev.Conflicts) > 0 {
		if err := a.Views.SetStatus(viewID, model.ViewStatusPendingRuling); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %d unresolved conflicts", model.ErrIllegalTransition, len(ev.Conflicts))
	}
	v, err := a.Views.Get(viewID)
	if err != nil {
		return nil, err
	}
	nameByID := map[string]model.NameRecord{}
	names, _ := a.Names.List()
	for _, n := range names {
		nameByID[n.ID] = n
	}
	chk, items := view.ChecklistSnapshot(viewID, v.RuleVersion, ev, nameByID)
	if err := a.Checklists.Save(&chk, items); err != nil {
		return nil, err
	}
	if err := a.Views.SetStatus(viewID, model.ViewStatusPublished); err != nil {
		return nil, err
	}
	return &chk, nil
}

// ViewChecklist 返回观点已发布清单（无则 ErrNotFound）。
func (a *App) ViewChecklist(viewID string) (*model.Checklist, []model.ChecklistItem, error) {
	c, err := a.Checklists.ByView(viewID)
	if err != nil {
		return nil, nil, err
	}
	items, err := a.Checklists.Items(c.ID)
	if err != nil {
		return nil, nil, err
	}
	return c, items, nil
}

// CompareChecklists 对比两份清单差异。
func (a *App) CompareChecklists(fromID, toID string) (*checklist.Diff, error) {
	from, err := a.Checklists.Get(fromID)
	if err != nil {
		return nil, err
	}
	to, err := a.Checklists.Get(toID)
	if err != nil {
		return nil, err
	}
	fromItems, err := a.Checklists.Items(from.ID)
	if err != nil {
		return nil, err
	}
	toItems, err := a.Checklists.Items(to.ID)
	if err != nil {
		return nil, err
	}
	d := checklist.Compare(*from, fromItems, *to, toItems)
	return &d, nil
}

// Stats 返回系统统计。
type Stats struct {
	Names        int `json:"names"`
	Publications int `json:"publications"`
	Specimens    int `json:"specimens"`
	Relations    int `json:"relations"`
	Views        int `json:"views"`
	Checklists   int `json:"checklists"`
	Accepted     int `json:"accepted"`
	Deferred     int `json:"deferred"`
}

// GetStats 统计各实体数量。
func (a *App) GetStats() (*Stats, error) {
	names, err := a.Names.List()
	if err != nil {
		return nil, err
	}
	pubs, err := a.Publications.List()
	if err != nil {
		return nil, err
	}
	specs, err := a.Specimens.List()
	if err != nil {
		return nil, err
	}
	rels, err := a.Relations.List()
	if err != nil {
		return nil, err
	}
	views, err := a.Views.List()
	if err != nil {
		return nil, err
	}
	chks, err := a.Checklists.List()
	if err != nil {
		return nil, err
	}
	accepted, deferred := 0, 0
	for _, n := range names {
		switch n.Status {
		case model.NameStatusAccepted:
			accepted++
		case model.NameStatusPendingReview:
			deferred++
		}
	}
	return &Stats{
		Names: len(names), Publications: len(pubs), Specimens: len(specs),
		Relations: len(rels), Views: len(views), Checklists: len(chks),
		Accepted: accepted, Deferred: deferred,
	}, nil
}

// SelfCheck 自检：返回组件版本与数据库健康状态。
func (a *App) SelfCheck() map[string]any {
	var ver string
	row := a.DB.SQL().QueryRow(`SELECT sqlite_version()`)
	_ = row.Scan(&ver)
	return map[string]any{
		"status":         "ok",
		"sqlite_version": ver,
		"checked_at":     time.Now().UTC().Format(time.RFC3339),
	}
}

// SortNames 按学名排序（供输出稳定）。
func SortNames(names []model.NameRecord) {
	sort.Slice(names, func(i, j int) bool { return names[i].ScientificName < names[j].ScientificName })
}
