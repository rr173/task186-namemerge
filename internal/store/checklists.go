package store

import (
	"database/sql"
	"errors"
	"time"

	"task186-namemerge/internal/model"
)

// ChecklistStore 提供清单快照的存取；已发布清单不可变。
type ChecklistStore struct{ db *DB }

// NewChecklistStore 构造清单存储。
func NewChecklistStore(db *DB) *ChecklistStore { return &ChecklistStore{db: db} }

// Save 保存清单快照及其条目（冻结）。
func (s *ChecklistStore) Save(c *model.Checklist, items []model.ChecklistItem) error {
	if c.ID == "" {
		c.ID = newID()
	}
	if c.Status == "" {
		c.Status = "frozen"
	}
	c.CreatedAt = time.Now().UTC()
	tx, err := s.db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO checklists (id, view_id, rule_version, fingerprint, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.ViewID, c.RuleVersion, c.Fingerprint, c.Status, c.CreatedAt.Format(time.RFC3339),
	); err != nil {
		return err
	}
	for _, it := range items {
		if _, err := tx.Exec(
			`INSERT INTO checklist_items (checklist_id, name_id, scientific_name, role, accepted_name_id) VALUES (?, ?, ?, ?, ?)`,
			c.ID, it.NameID, it.ScientificName, it.Role, it.AcceptedNameID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Get 按 ID 读取清单头。
func (s *ChecklistStore) Get(id string) (*model.Checklist, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, view_id, rule_version, fingerprint, status, created_at FROM checklists WHERE id=?`, id)
	var c model.Checklist
	var created string
	err := row.Scan(&c.ID, &c.ViewID, &c.RuleVersion, &c.Fingerprint, &c.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ByView 返回某观点最新清单。
func (s *ChecklistStore) ByView(viewID string) (*model.Checklist, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, view_id, rule_version, fingerprint, status, created_at FROM checklists WHERE view_id=? ORDER BY created_at DESC LIMIT 1`, viewID)
	var c model.Checklist
	var created string
	err := row.Scan(&c.ID, &c.ViewID, &c.RuleVersion, &c.Fingerprint, &c.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Items 返回清单条目。
func (s *ChecklistStore) Items(checklistID string) ([]model.ChecklistItem, error) {
	rows, err := s.db.sql.Query(
		`SELECT checklist_id, name_id, scientific_name, role, accepted_name_id FROM checklist_items WHERE checklist_id=? ORDER BY scientific_name`, checklistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ChecklistItem
	for rows.Next() {
		var it model.ChecklistItem
		if err := rows.Scan(&it.ChecklistID, &it.NameID, &it.ScientificName, &it.Role, &it.AcceptedNameID); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// List 返回全部清单头。
func (s *ChecklistStore) List() ([]model.Checklist, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, view_id, rule_version, fingerprint, status, created_at FROM checklists ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Checklist
	for rows.Next() {
		var c model.Checklist
		var created string
		if err := rows.Scan(&c.ID, &c.ViewID, &c.RuleVersion, &c.Fingerprint, &c.Status, &created); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Delete 删除清单及其条目（仅用于测试清理，生产清单冻结不可删）。
func (s *ChecklistStore) Delete(id string) error {
	tx, err := s.db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM checklist_items WHERE checklist_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM checklists WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
