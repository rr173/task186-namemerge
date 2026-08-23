package store

import (
	"database/sql"
	"errors"
	"time"

	"task186-namemerge/internal/model"
)

// SpecimenStore 提供模式标本的 CRUD 与名称绑定。
type SpecimenStore struct{ db *DB }

// NewSpecimenStore 构造模式存储。
func NewSpecimenStore(db *DB) *SpecimenStore { return &SpecimenStore{db: db} }

// Create 插入模式标本；指纹重复返回 ErrConflict。
func (s *SpecimenStore) Create(sp *model.Specimen) error {
	now := time.Now().UTC()
	sp.CreatedAt = now
	if sp.ID == "" {
		sp.ID = newID()
	}
	if sp.Fingerprint == "" {
		return model.ErrInvalidArgument
	}
	_, err := s.db.sql.Exec(
		`INSERT INTO specimens (id, collector, number, institution, fingerprint, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		sp.ID, sp.Collector, sp.Number, sp.Institution, sp.Fingerprint, sp.CreatedAt.Format(time.RFC3339),
	)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	return err
}

// Get 按 ID 读取模式标本。
func (s *SpecimenStore) Get(id string) (*model.Specimen, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, collector, number, institution, fingerprint, created_at FROM specimens WHERE id=?`, id)
	var sp model.Specimen
	var created string
	err := row.Scan(&sp.ID, &sp.Collector, &sp.Number, &sp.Institution, &sp.Fingerprint, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// ByFingerprint 按幂等指纹查模式标本。
func (s *SpecimenStore) ByFingerprint(fp string) (*model.Specimen, error) {
	row := s.db.sql.QueryRow(
		`SELECT id, collector, number, institution, fingerprint, created_at FROM specimens WHERE fingerprint=?`, fp)
	var sp model.Specimen
	var created string
	err := row.Scan(&sp.ID, &sp.Collector, &sp.Number, &sp.Institution, &sp.Fingerprint, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sp, nil
}

// List 返回全部模式标本。
func (s *SpecimenStore) List() ([]model.Specimen, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, collector, number, institution, fingerprint, created_at FROM specimens ORDER BY collector`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Specimen
	for rows.Next() {
		var sp model.Specimen
		var created string
		if err := rows.Scan(&sp.ID, &sp.Collector, &sp.Number, &sp.Institution, &sp.Fingerprint, &created); err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

// Link 把名称与发表证据、模式标本绑定（幂等：同名称+发表重复绑定
// 时仅当本次新提供模式标本才更新模式，保证"补齐模式"场景生效；
// 若本次未提供模式标本，则保留既有模式证据，避免空请求清空已绑定模式）。
func (s *SpecimenStore) Link(nameID, publicationID, specimenID string) (*model.NameLink, error) {
	// 名称与发表证据必须存在。
	_, err := s.db.sql.Exec(
		`INSERT INTO name_links (id, name_id, publication_id, specimen_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		newID(), nameID, publicationID, specimenID, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		if isUniqueViolation(err) {
			// 幂等：已绑定。仅当本次新提供模式标本时才补齐模式，
			// 否则保留既有模式证据（防止未带模式的重复请求清空已绑定模式）。
			if specimenID != "" {
				if _, uerr := s.db.sql.Exec(
					`UPDATE name_links SET specimen_id=? WHERE name_id=? AND publication_id=?`,
					specimenID, nameID, publicationID,
				); uerr != nil {
					return nil, uerr
				}
			}
			row := s.db.sql.QueryRow(
				`SELECT id, name_id, publication_id, specimen_id, created_at FROM name_links WHERE name_id=? AND publication_id=?`,
				nameID, publicationID)
			var l model.NameLink
			var created string
			if err := row.Scan(&l.ID, &l.NameID, &l.PublicationID, &l.SpecimenID, &created); err != nil {
				return nil, err
			}
			return &l, nil
		}
		return nil, err
	}
	row := s.db.sql.QueryRow(
		`SELECT id, name_id, publication_id, specimen_id, created_at FROM name_links WHERE name_id=? AND publication_id=?`,
		nameID, publicationID)
	var l model.NameLink
	var created string
	if err := row.Scan(&l.ID, &l.NameID, &l.PublicationID, &l.SpecimenID, &created); err != nil {
		return nil, err
	}
	return &l, nil
}

// LinksByName 查某名称的全部绑定。
func (s *SpecimenStore) LinksByName(nameID string) ([]model.NameLink, error) {
	rows, err := s.db.sql.Query(
		`SELECT id, name_id, publication_id, specimen_id, created_at FROM name_links WHERE name_id=?`, nameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.NameLink
	for rows.Next() {
		var l model.NameLink
		var created string
		if err := rows.Scan(&l.ID, &l.NameID, &l.PublicationID, &l.SpecimenID, &created); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// SpecimenByLink 由 name_links 解析出模式标本指纹集合。
func (s *SpecimenStore) SpecimenByLink(nameID string) ([]string, error) {
	rows, err := s.db.sql.Query(
		`SELECT sp.fingerprint FROM specimens sp JOIN name_links l ON l.specimen_id = sp.id WHERE l.name_id = ? AND l.specimen_id != ''`, nameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var fp string
		if err := rows.Scan(&fp); err != nil {
			return nil, err
		}
		out = append(out, fp)
	}
	return out, rows.Err()
}
