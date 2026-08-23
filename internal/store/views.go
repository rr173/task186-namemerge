package store

import (
	"database/sql"
	"errors"
	"time"

	"task186-namemerge/internal/model"
)

// ViewStore 提供分类观点的 CRUD 与状态转移。
type ViewStore struct{ db *DB }

// NewViewStore 构造观点存储。
func NewViewStore(db *DB) *ViewStore { return &ViewStore{db: db} }

// Create 插入观点；默认 draft 状态。
func (s *ViewStore) Create(v *model.View) error {
	now := time.Now().UTC()
	v.CreatedAt, v.UpdatedAt = now, now
	if v.Status == "" {
		v.Status = model.ViewStatusDraft
	}
	if v.ID == "" {
		v.ID = newID()
	}
	_, err := s.db.sql.Exec(
		`INSERT INTO views (id, name, rule_version, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		v.ID, v.Name, v.RuleVersion, string(v.Status), v.CreatedAt.Format(time.RFC3339), v.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// Get 按 ID 读取观点。
func (s *ViewStore) Get(id string) (*model.View, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, name, rule_version, status, created_at, updated_at FROM views WHERE id=?`, id)
	var v model.View
	var created, updated string
	err := row.Scan(&v.ID, &v.Name, &v.RuleVersion, &v.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// List 返回全部观点（按创建时间倒序）。
func (s *ViewStore) List() ([]model.View, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, name, rule_version, status, created_at, updated_at FROM views ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.View
	for rows.Next() {
		var v model.View
		var created, updated string
		if err := rows.Scan(&v.ID, &v.Name, &v.RuleVersion, &v.Status, &created, &updated); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// SetStatus 转移观点状态，遵循状态机约束：
//   - draft → publishable / pending_ruling / published
//   - published → superseded（仅允许标记替代，禁止直接编辑）
//   - 其他非法转移返回 ErrIllegalTransition。
func (s *ViewStore) SetStatus(id string, status model.ViewStatus) error {
	cur, err := s.Get(id)
	if err != nil {
		return err
	}
	if !validTransition(cur.Status, status) {
		return model.ErrIllegalTransition
	}
	res, err := s.db.sql.Exec(
		`UPDATE views SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return assertAffected(res, 1)
}

func validTransition(from, to model.ViewStatus) bool {
	switch from {
	case model.ViewStatusDraft:
		return to == model.ViewStatusPublishable ||
			to == model.ViewStatusPendingRuling ||
			to == model.ViewStatusPublished
	case model.ViewStatusPublishable:
		return to == model.ViewStatusPublished || to == model.ViewStatusPendingRuling
	case model.ViewStatusPendingRuling:
		return to == model.ViewStatusPublished || to == model.ViewStatusPublishable
	case model.ViewStatusPublished:
		return to == model.ViewStatusSuperseded
	default:
		return false
	}
}

// RulingStore 提供裁决记录存取。
type RulingStore struct{ db *DB }

// NewRulingStore 构造裁决存储。
func NewRulingStore(db *DB) *RulingStore { return &RulingStore{db: db} }

// Create 插入裁决；同观点+同关系重复裁决返回 ErrConflict。
func (s *RulingStore) Create(r *model.Ruling) error {
	if r.ID == "" {
		r.ID = newID()
	}
	r.CreatedAt = time.Now().UTC()
	_, err := s.db.sql.Exec(
		`INSERT INTO rulings (id, view_id, relation_id, decision, rationale, ruled_by, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ViewID, r.RelationID, string(r.Decision), r.Rationale, r.RuledBy, r.CreatedAt.Format(time.RFC3339),
	)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	return err
}

// ByView 返回某观点下全部裁决。
func (s *RulingStore) ByView(viewID string) ([]model.Ruling, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, view_id, relation_id, decision, rationale, ruled_by, created_at FROM rulings WHERE view_id=? ORDER BY created_at`, viewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Ruling
	for rows.Next() {
		var r model.Ruling
		var created string
		if err := rows.Scan(&r.ID, &r.ViewID, &r.RelationID, &r.Decision, &r.Rationale, &r.RuledBy, &created); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetRelationStatus 按裁决把关系边置为 proven / rejected。
func (s *RulingStore) SetRelationStatus(relationID string, status model.RelationStatus) error {
	res, err := s.db.sql.Exec(
		`UPDATE relations SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now().UTC().Format(time.RFC3339), relationID)
	if err != nil {
		return err
	}
	return assertAffected(res, 1)
}
