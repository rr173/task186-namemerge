// Package model 定义植物学名异名归并证据服务的核心实体与状态机。
//
// 领域模型围绕《国际植物命名法规》(ICN) 的四个核心对象：
// 名称记录 (NameRecord)、发表证据 (Publication)、模式标本 (Specimen)、
// 名称关系 (NameRelation)，以及组织它们的两层容器：
// 分类观点 (View) 与清单快照 (Checklist)。
package model

import "time"

// NameStatus 是名称记录的状态机取值。
//
// 流转：pending_review → legitimate / illegitimate / synonym_candidate → accepted
//   - pending_review      待核验：已登记但尚未完成法规判定；
//   - legitimate          合法：发表有效、日期可排序、具备模式关联或满足豁免；
//   - illegitimate        非合法：后同名、无有效发表或模式指向已被占据的接受名；
//   - synonym_candidate   异名候选：与另一名称共享模式（同型）或经组合变更归并（异型）；
//   - accepted            已接受：在某个分类观点中被采纳为当前接受名。
type NameStatus string

const (
	NameStatusPendingReview    NameStatus = "pending_review"
	NameStatusLegitimate       NameStatus = "legitimate"
	NameStatusIllegitimate     NameStatus = "illegitimate"
	NameStatusSynonymCandidate NameStatus = "synonym_candidate"
	NameStatusAccepted         NameStatus = "accepted"
)

// PublicationStatus 是发表证据的状态机取值。
//
// 流转：pending_check → valid / date_conflict / missing_type
//   - pending_check  待核对：证据已录入但尚未校验；
//   - valid          有效：作者引证完整、发表日期区间非空且可排序、模式关联成立；
//   - date_conflict  日期冲突：两个名称的发表日期区间重叠或缺失，无法判定先后；
//   - missing_type   缺少模式：名称已发表但未指定可核验的模式标本。
type PublicationStatus string

const (
	PublicationStatusPendingCheck PublicationStatus = "pending_check"
	PublicationStatusValid        PublicationStatus = "valid"
	PublicationStatusDateConflict PublicationStatus = "date_conflict"
	PublicationStatusMissingType  PublicationStatus = "missing_type"
)

// RelationStatus 是名称关系的状态机取值。
//
// 流转：proposed → proven / conflicting / rejected
//   - proposed    提议：两个名称之间已登记候选关系边；
//   - proven      已证明：证据链（共享模式 / 组合变更 / 优先权裁决）闭合；
//   - conflicting 存在冲突：同一模式指向互斥接受名，或日期无法排序；
//   - rejected    已驳回：经裁决判定不构成异名关系。
type RelationStatus string

const (
	RelationStatusProposed    RelationStatus = "proposed"
	RelationStatusProven      RelationStatus = "proven"
	RelationStatusConflicting RelationStatus = "conflicting"
	RelationStatusRejected    RelationStatus = "rejected"
)

// RelationKind 描述名称关系的证据类别。
type RelationKind string

const (
	// RelationKindHomotypic 同型异名：共享同一模式标本。
	RelationKindHomotypic RelationKind = "homotypic"
	// RelationKindHeterotypic 异型异名：模式不同但按分类观点归并。
	RelationKindHeterotypic RelationKind = "heterotypic"
	// RelationKindCombination 组合变更：同一基名在不同属下的新组合。
	RelationKindCombination RelationKind = "combination"
)

// ViewStatus 是分类观点的状态机取值。
//
// 流转：draft → publishable / pending_ruling → published → superseded
//   - draft          草案：观点正在编辑；
//   - publishable    可发布：无未决冲突；
//   - pending_ruling 待裁决：存在冲突关系等待分类学家裁定；
//   - published      已发布：清单快照冻结对外可查；
//   - superseded     已替代：观点修订产生新清单，旧观点只读。
type ViewStatus string

const (
	ViewStatusDraft        ViewStatus = "draft"
	ViewStatusPublishable  ViewStatus = "publishable"
	ViewStatusPendingRuling ViewStatus = "pending_ruling"
	ViewStatusPublished    ViewStatus = "published"
	ViewStatusSuperseded   ViewStatus = "superseded"
)

// RulingDecision 是裁决结论。
type RulingDecision string

const (
	RulingAccept   RulingDecision = "accept"
	RulingReject   RulingDecision = "reject"
	RulingDefer    RulingDecision = "defer"
)

// NameRecord 是一条植物学名记录。
type NameRecord struct {
	ID              string     `json:"id"`
	ScientificName  string     `json:"scientific_name"`  // 完整学名（属 + 种加词 + 命名人缩写）
	Genus           string     `json:"genus"`            // 属名
	SpecificEpithet string     `json:"specific_epithet"` // 种加词
	Authors         string     `json:"authors"`          // 作者引证（如 "L."、"J.Sm."）
	YearRangeStart  *int       `json:"year_range_start,omitempty"` // 发表年份区间起点
	YearRangeEnd    *int       `json:"year_range_end,omitempty"`   // 发表年份区间终点
	OrthographicKey string     `json:"orthographic_key"`           // 拼写归一化键（忽略变体差异）
	Status          NameStatus `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Publication 是一条发表证据记录。
type Publication struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Authors         string            `json:"authors"`
	Journal         string            `json:"journal"`
	YearRangeStart  *int              `json:"year_range_start,omitempty"`
	YearRangeEnd    *int              `json:"year_range_end,omitempty"`
	Fingerprint     string            `json:"fingerprint"` // 幂等指纹：sha256(title|authors|journal)
	Status          PublicationStatus `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// Specimen 是一条模式标本记录。
type Specimen struct {
	ID          string    `json:"id"`
	Collector   string    `json:"collector"`    // 采集人
	Number      string    `json:"number"`       // 采集号
	Institution string    `json:"institution"`  // 保存机构（标本馆代码，如 K、P、BM）
	Fingerprint string    `json:"fingerprint"`  // 幂等指纹：sha256(collector|number|institution)
	CreatedAt   time.Time `json:"created_at"`
}

// NameLink 关联一个名称与其发表证据、模式标本。
type NameLink struct {
	ID            string    `json:"id"`
	NameID        string    `json:"name_id"`
	PublicationID string    `json:"publication_id"`
	SpecimenID    string    `json:"specimen_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// NameRelation 是两条名称之间的候选/已证明关系边。
type NameRelation struct {
	ID          string         `json:"id"`
	FromNameID  string         `json:"from_name_id"`
	ToNameID    string         `json:"to_name_id"`
	Kind        RelationKind   `json:"kind"`
	Basis       string         `json:"basis"` // 证据依据说明
	Status      RelationStatus `json:"status"`
	ViewID      string         `json:"view_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// RuleVersion 是一份法规规则版本（判定所依据的规则集）。
type RuleVersion struct {
	ID            string    `json:"id"`
	Version       string    `json:"version"`  // 如 "ICN-2026"
	PriorityRule  bool      `json:"priority_rule"`
	LegitimacyReq bool      `json:"legitimacy_req"`
	HomonymRule   bool      `json:"homonym_rule"`
	Orthography   bool      `json:"orthography"`
	CreatedAt     time.Time `json:"created_at"`
}

// View 是一个分类观点：在某一规则版本下对名称簇的取舍。
type View struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	RuleVersion string    `json:"rule_version"`
	Status      ViewStatus `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Ruling 是一条针对冲突关系的裁决记录。
type Ruling struct {
	ID        string         `json:"id"`
	ViewID    string         `json:"view_id"`
	RelationID string        `json:"relation_id"`
	Decision  RulingDecision `json:"decision"`
	Rationale string         `json:"rationale"`
	RuledBy   string         `json:"ruled_by"`
	CreatedAt time.Time      `json:"created_at"`
}

// Checklist 是已发布清单的快照。
type Checklist struct {
	ID            string    `json:"id"`
	ViewID        string    `json:"view_id"`
	RuleVersion   string    `json:"rule_version"`
	Fingerprint   string    `json:"fingerprint"` // 快照指纹：sha256(序列化条目)
	Status        string    `json:"status"`      // frozen
	CreatedAt     time.Time `json:"created_at"`
}

// ChecklistItem 是清单快照中的一行：一个名称在观点中的角色。
type ChecklistItem struct {
	ChecklistID  string `json:"checklist_id"`
	NameID       string `json:"name_id"`
	ScientificName string `json:"scientific_name"`
	Role         string `json:"role"` // accepted / synonym / deferred
	AcceptedNameID string `json:"accepted_name_id,omitempty"`
}
