package store

import (
	"database/sql"
	"errors"
	"time"

	"task186-namemerge/internal/model"
)

// PublicationStore 提供发表证据的 CRUD 与状态转移。
type PublicationStore struct{ db *DB }

// NewPublicationStore 构造发表存储。
func NewPublicationStore(db *DB) *PublicationStore { return &PublicationStore{db: db} }

// Create 插入发表证据；指纹重复（同一文献）返回 ErrConflict（幂等拒绝）。
func (s *PublicationStore) Create(p *model.Publication) error {
	now := time.Now().UTC()
	p.CreatedAt, p.UpdatedAt = now, now
	if p.Status == "" {
		p.Status = model.PublicationStatusPendingCheck
	}
	if p.ID == "" {
		p.ID = newID()
	}
	if p.Fingerprint == "" {
		return model.ErrInvalidArgument
	}
	_, err := s.db.sql.Exec(
		`INSERT INTO publications (id, title, authors, journal, year_range_start, year_range_end, fingerprint, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Title, p.Authors, p.Journal, p.YearRangeStart, p.YearRangeEnd,
		p.Fingerprint, string(p.Status), p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
	)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	return err
}

// Get 按 ID 读取发表证据。
func (s *PublicationStore) Get(id string) (*model.Publication, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, title, authors, journal, year_range_start, year_range_end, fingerprint, status, created_at, updated_at FROM publications WHERE id=?`, id)
	var p model.Publication
	var created, updated string
	err := row.Scan(&p.ID, &p.Title, &p.Authors, &p.Journal, &p.YearRangeStart, &p.YearRangeEnd,
		&p.Fingerprint, &p.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ByFingerprint 按幂等指纹查发表证据。
func (s *PublicationStore) ByFingerprint(fp string) (*model.Publication, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, title, authors, journal, year_range_start, year_range_end, fingerprint, status, created_at, updated_at FROM publications WHERE fingerprint=?`, fp)
	var p model.Publication
	var created, updated string
	err := row.Scan(&p.ID, &p.Title, &p.Authors, &p.Journal, &p.YearRangeStart, &p.YearRangeEnd,
		&p.Fingerprint, &p.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List 返回全部发表证据（按标题排序）。
func (s *PublicationStore) List() ([]model.Publication, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, title, authors, journal, year_range_start, year_range_end, fingerprint, status, created_at, updated_at FROM publications ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Publication
	for rows.Next() {
		var p model.Publication
		var created, updated string
		if err := rows.Scan(&p.ID, &p.Title, &p.Authors, &p.Journal, &p.YearRangeStart, &p.YearRangeEnd,
			&p.Fingerprint, &p.Status, &created, &updated); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetStatus 更新状态（供校验引擎写回）。
func (s *PublicationStore) SetStatus(id string, status model.PublicationStatus) error {
	res, err := s.db.sql.Exec(
		`UPDATE publications SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return assertAffected(res, 1)
}

// ByName 查某名称关联的发表证据（经 name_links）。
func (s *PublicationStore) ByName(nameID string) (*model.Publication, error) {
	row := s.db.sql.QueryRow(
		`SELECT p.id, p.title, p.authors, p.journal, p.year_range_start, p.year_range_end, p.fingerprint, p.status, p.created_at, p.updated_at
		 FROM publications p JOIN name_links l ON l.publication_id = p.id WHERE l.name_id = ? ORDER BY l.created_at DESC LIMIT 1`, nameID)
	var p model.Publication
	var created, updated string
	err := row.Scan(&p.ID, &p.Title, &p.Authors, &p.Journal, &p.YearRangeStart, &p.YearRangeEnd,
		&p.Fingerprint, &p.Status, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
