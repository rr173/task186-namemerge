package store

import (
	"database/sql"
	"errors"
	"time"

	"task186-namemerge/internal/model"
)

// RelationStore 提供名称关系边的 CRUD 与状态转移。
type RelationStore struct{ db *DB }

// NewRelationStore 构造关系存储。
func NewRelationStore(db *DB) *RelationStore { return &RelationStore{db: db} }

// Create 插入关系边；同名对（任意方向）重复返回 ErrConflict。
func (s *RelationStore) Create(r *model.NameRelation) error {
	// 对称唯一：a→b 与 b→a 视为同一对名称关系。
	if _, err := s.Pair(r.FromNameID, r.ToNameID); err == nil {
		return model.ErrConflict
	} else if err != model.ErrNotFound {
		return err
	}
	now := time.Now().UTC()
	r.CreatedAt, r.UpdatedAt = now, now
	if r.Status == "" {
		r.Status = model.RelationStatusProposed
	}
	if r.ID == "" {
		r.ID = newID()
	}
	_, err := s.db.sql.Exec(
		`INSERT INTO relations (id, from_name_id, to_name_id, kind, basis, status, view_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.FromNameID, r.ToNameID, string(r.Kind), r.Basis, string(r.Status), r.ViewID,
		r.CreatedAt.Format(time.RFC3339), r.UpdatedAt.Format(time.RFC3339),
	)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	return err
}

// Get 按 ID 读取关系。
func (s *RelationStore) Get(id string) (*model.NameRelation, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, from_name_id, to_name_id, kind, basis, status, view_id, created_at, updated_at FROM relations WHERE id=?`, id)
	var r model.NameRelation
	var created, updated string
	err := row.Scan(&r.ID, &r.FromNameID, &r.ToNameID, &r.Kind, &r.Basis, &r.Status, &r.ViewID, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// List 返回全部关系边。
func (s *RelationStore) List() ([]model.NameRelation, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, from_name_id, to_name_id, kind, basis, status, view_id, created_at, updated_at FROM relations ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.NameRelation
	for rows.Next() {
		var r model.NameRelation
		var created, updated string
		if err := rows.Scan(&r.ID, &r.FromNameID, &r.ToNameID, &r.Kind, &r.Basis, &r.Status, &r.ViewID, &created, &updated); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetStatus 更新关系状态（proposed → proven / conflicting / rejected）。
func (s *RelationStore) SetStatus(id string, status model.RelationStatus) error {
	res, err := s.db.sql.Exec(
		`UPDATE relations SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return assertAffected(res, 1)
}

// Pair 查两个名称之间已存在的关系边（任一方向）。
// 学名关系是无向对：a→b 与 b→a 视为同一对名称关系，
// 因此两个方向都要匹配，否则反向重复提交会被漏判为非重复。
func (s *RelationStore) Pair(a, b string) (*model.NameRelation, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, from_name_id, to_name_id, kind, basis, status, view_id, created_at, updated_at
		 FROM relations
		 WHERE (from_name_id=? AND to_name_id=?) OR (from_name_id=? AND to_name_id=?)
		 LIMIT 1`,
		a, b, b, a)
	var r model.NameRelation
	var created, updated string
	err := row.Scan(&r.ID, &r.FromNameID, &r.ToNameID, &r.Kind, &r.Basis, &r.Status, &r.ViewID, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ByView 查指定观点下的关系边。
func (s *RelationStore) ByView(viewID string) ([]model.NameRelation, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, from_name_id, to_name_id, kind, basis, status, view_id, created_at, updated_at FROM relations WHERE view_id=? ORDER BY created_at`, viewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.NameRelation
	for rows.Next() {
		var r model.NameRelation
		var created, updated string
		if err := rows.Scan(&r.ID, &r.FromNameID, &r.ToNameID, &r.Kind, &r.Basis, &r.Status, &r.ViewID, &created, &updated); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
