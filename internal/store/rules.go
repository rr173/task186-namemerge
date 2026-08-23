package store

import (
	"database/sql"
	"errors"
	"time"

	"task186-namemerge/internal/model"
)

// RuleStore 提供法规规则版本的存取。
type RuleStore struct{ db *DB }

// NewRuleStore 构造规则存储。
func NewRuleStore(db *DB) *RuleStore { return &RuleStore{db: db} }

// Create 插入规则版本；版本号重复返回 ErrConflict。
func (s *RuleStore) Create(r *model.RuleVersion) error {
	if r.ID == "" {
		r.ID = newID()
	}
	r.CreatedAt = time.Now().UTC()
	_, err := s.db.sql.Exec(
		`INSERT INTO rule_versions (id, version, priority_rule, legitimacy_req, homonym_rule, orthography, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Version, boolToInt(r.PriorityRule), boolToInt(r.LegitimacyReq),
		boolToInt(r.HomonymRule), boolToInt(r.Orthography), r.CreatedAt.Format(time.RFC3339),
	)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	return err
}

// Get 按 ID 读取规则版本。
func (s *RuleStore) Get(id string) (*model.RuleVersion, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, version, priority_rule, legitimacy_req, homonym_rule, orthography, created_at FROM rule_versions WHERE id=?`, id)
	var r model.RuleVersion
	var created string
	err := row.Scan(&r.ID, &r.Version, &r.PriorityRule, &r.LegitimacyReq, &r.HomonymRule, &r.Orthography, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Current 返回最近创建的规则版本（无则返回 nil, nil）。
func (s *RuleStore) Current() (*model.RuleVersion, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, version, priority_rule, legitimacy_req, homonym_rule, orthography, created_at FROM rule_versions ORDER BY created_at DESC LIMIT 1`)
	var r model.RuleVersion
	var created string
	err := row.Scan(&r.ID, &r.Version, &r.PriorityRule, &r.LegitimacyReq, &r.HomonymRule, &r.Orthography, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// List 返回全部规则版本。
func (s *RuleStore) List() ([]model.RuleVersion, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, version, priority_rule, legitimacy_req, homonym_rule, orthography, created_at FROM rule_versions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.RuleVersion
	for rows.Next() {
		var r model.RuleVersion
		var created string
		if err := rows.Scan(&r.ID, &r.Version, &r.PriorityRule, &r.LegitimacyReq, &r.HomonymRule, &r.Orthography, &created); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
