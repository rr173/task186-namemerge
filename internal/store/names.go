package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task186-namemerge/internal/model"
)

// NameStore 提供名称记录的 CRUD 与状态转移。
type NameStore struct{ db *DB }

// NewNameStore 构造名称存储。
func NewNameStore(db *DB) *NameStore { return &NameStore{db: db} }

// Create 插入名称记录；拼写+作者唯一冲突返回 ErrConflict。
func (s *NameStore) Create(n *model.NameRecord) error {
	now := time.Now().UTC()
	n.CreatedAt, n.UpdatedAt = now, now
	if n.Status == "" {
		n.Status = model.NameStatusPendingReview
	}
	if n.ID == "" {
		n.ID = newID()
	}
	_, err := s.db.sql.Exec(
		`INSERT INTO names (id, scientific_name, genus, specific_epithet, authors, year_range_start, year_range_end, orthographic_key, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.ScientificName, n.Genus, n.SpecificEpithet, n.Authors,
		n.YearRangeStart, n.YearRangeEnd, n.OrthographicKey, string(n.Status),
		n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339),
	)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	return err
}

// Get 按 ID 读取名称。
func (s *NameStore) Get(id string) (*model.NameRecord, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, scientific_name, genus, specific_epithet, authors, year_range_start, year_range_end, orthographic_key, status, created_at, updated_at FROM names WHERE id = ?`, id)
	var n model.NameRecord
	var created, updated string
	err := row.Scan(&n.ID, &n.ScientificName, &n.Genus, &n.SpecificEpithet, &n.Authors,
		&n.YearRangeStart, &n.YearRangeEnd, &n.OrthographicKey, &n.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// List 返回全部名称（按学名排序）。
func (s *NameStore) List() ([]model.NameRecord, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, scientific_name, genus, specific_epithet, authors, year_range_start, year_range_end, orthographic_key, status, created_at, updated_at FROM names ORDER BY scientific_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.NameRecord
	for rows.Next() {
		var n model.NameRecord
		var created, updated string
		if err := rows.Scan(&n.ID, &n.ScientificName, &n.Genus, &n.SpecificEpithet, &n.Authors,
			&n.YearRangeStart, &n.YearRangeEnd, &n.OrthographicKey, &n.Status, &created, &updated); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// Update 更新名称字段；已接受或非法状态禁止直接降级（状态机约束）。
func (s *NameStore) Update(n *model.NameRecord) error {
	old, err := s.Get(n.ID)
	if err != nil {
		return err
	}
	if old.Status == model.NameStatusAccepted && n.Status != model.NameStatusAccepted {
		return model.ErrIllegalTransition
	}
	n.UpdatedAt = time.Now().UTC()
	res, err := s.db.sql.Exec(
		`UPDATE names SET scientific_name=?, genus=?, specific_epithet=?, authors=?, year_range_start=?, year_range_end=?, orthographic_key=?, status=?, updated_at=? WHERE id=?`,
		n.ScientificName, n.Genus, n.SpecificEpithet, n.Authors, n.YearRangeStart, n.YearRangeEnd,
		n.OrthographicKey, string(n.Status), n.UpdatedAt.Format(time.RFC3339), n.ID)
	if err != nil {
		return err
	}
	return assertAffected(res, 1)
}

// SetStatus 仅更新状态（供规则引擎写回）。
func (s *NameStore) SetStatus(id string, status model.NameStatus) error {
	res, err := s.db.sql.Exec(
		`UPDATE names SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return assertAffected(res, 1)
}

// findByOrtho 按拼写键查名称（供变体归并检测）。
func (s *NameStore) findByOrtho(key string) ([]model.NameRecord, error) {
	rows, err := s.db.sql.Query(`SELECT id, scientific_name, genus, specific_epithet, authors, year_range_start, year_range_end, orthographic_key, status, created_at, updated_at FROM names WHERE orthographic_key=? ORDER BY created_at`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.NameRecord
	for rows.Next() {
		var n model.NameRecord
		if err := rows.Scan(&n.ID, &n.ScientificName, &n.Genus, &n.SpecificEpithet, &n.Authors,
			&n.YearRangeStart, &n.YearRangeEnd, &n.OrthographicKey, &n.Status, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ByOrtho 暴露拼写键查询。
func (s *NameStore) ByOrtho(key string) ([]model.NameRecord, error) { return s.findByOrtho(key) }

func assertAffected(res sql.Result, want int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != want {
		return model.ErrNotFound
	}
	return nil
}

func newID() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = byte(time.Now().UnixNano()>>(8*i)) ^ byte(i*31+7)
	}
	return fmt.Sprintf("n%x", b)
}
