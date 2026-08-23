// Package smoke 实现 --smoke-test 自检：
// 真实创建文档/实体、调用核心逻辑、关闭并重新打开数据库
// 验证持久化与重启恢复，最后以 0 退出码结束。
package smoke

import (
	"fmt"
	"os"
	"path/filepath"

	"task186-namemerge/internal/model"
	"task186-namemerge/internal/service"
	"task186-namemerge/internal/store"
)

// Run 执行冒烟测试，任何断言失败返回错误。
func Run(dbPath string) error {
	if dbPath == "" || dbPath == ":memory:" {
		dbPath = filepath.Join(os.TempDir(), "namemerge-smoke.db")
	}
	// 清理旧库确保幂等。
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	app := service.NewApp(db)

	// 1. 登记名称与证据。
	nameA, err := app.RegisterName("Quercus robur L.")
	if err != nil {
		return fmt.Errorf("register A: %w", err)
	}
	nameB, err := app.RegisterName("Quercus pedunculata Ehrh.")
	if err != nil {
		return fmt.Errorf("register B: %w", err)
	}
	nameC, err := app.RegisterName("Fagus sylvatica L.")
	if err != nil {
		return fmt.Errorf("register C: %w", err)
	}

	yA, yB := 1753, 1790
	pubA, err := app.RegisterPublication(model.Publication{
		Title: "Species Plantarum", Authors: "Linnaeus, C.",
		Journal: "Sp. Pl.", YearRangeStart: &yA, YearRangeEnd: &yA,
	})
	if err != nil {
		return fmt.Errorf("pub A: %w", err)
	}
	pubB, err := app.RegisterPublication(model.Publication{
		Title: "Beiträge zur Naturkunde", Authors: "Ehrhart, F.",
		Journal: "Beitr. Naturk.", YearRangeStart: &yB, YearRangeEnd: &yB,
	})
	if err != nil {
		return fmt.Errorf("pub B: %w", err)
	}
	pubC, err := app.RegisterPublication(model.Publication{
		Title: "Species Plantarum", Authors: "Linnaeus, C.",
		Journal: "Sp. Pl.", YearRangeStart: &yA, YearRangeEnd: &yA,
	})
	if err != nil {
		return fmt.Errorf("pub C: %w", err)
	}

	// 2. 模式标本：A 与 B 共享同一模式 → 同型异名。
	sp, err := app.RegisterSpecimen(model.Specimen{
		Collector: "Smith", Number: "1234", Institution: "K",
	})
	if err != nil {
		return fmt.Errorf("specimen: %w", err)
	}
	if _, err := app.BindEvidence(nameA.ID, pubA.ID, sp.ID); err != nil {
		return fmt.Errorf("bind A: %w", err)
	}
	if _, err := app.BindEvidence(nameB.ID, pubB.ID, sp.ID); err != nil {
		return fmt.Errorf("bind B: %w", err)
	}
	if _, err := app.BindEvidence(nameC.ID, pubC.ID, ""); err != nil {
		return fmt.Errorf("bind C: %w", err)
	}

	// 3. 提议并证明 A↔B 同型异名关系。
	rel, err := app.ProposeRelation(nameA.ID, nameB.ID, model.RelationKindHomotypic, "shared type specimen K-1234")
	if err != nil {
		return fmt.Errorf("propose: %w", err)
	}
	if err := app.ProveRelation(rel.ID); err != nil {
		return fmt.Errorf("prove: %w", err)
	}

	// 4. 创建规则版本与观点并求值：A（1753 早）应为接受名，B 为异名，C 缺模式待裁决。
	rule, err := app.CreateRule(model.RuleVersion{
		Version: "ICN-2026", PriorityRule: true, LegitimacyReq: true,
		HomonymRule: true, Orthography: true,
	})
	if err != nil {
		return fmt.Errorf("rule: %w", err)
	}
	view1, err := app.CreateView("World Checklist v1", rule.ID)
	if err != nil {
		return fmt.Errorf("view: %w", err)
	}
	ev, err := app.EvaluateView(view1.ID)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}
	if got := ev.Roles[nameA.ID]; got != "accepted" {
		return fmt.Errorf("A role = %q, want accepted", got)
	}
	if got := ev.Roles[nameB.ID]; got != "synonym" {
		return fmt.Errorf("B role = %q, want synonym", got)
	}
	if got := ev.Roles[nameC.ID]; got != "deferred" {
		return fmt.Errorf("C role = %q, want deferred (missing type)", got)
	}

	// 5. 发布清单应因 C 缺模式（missing_type）进入 pending_ruling 而被拒绝。
	if _, err := app.PublishView(view1.ID); err == nil {
		return fmt.Errorf("publish should fail while C lacks type specimen")
	}

	// 6. 补齐 C 的模式标本 → 重新求值 → 发布成功。
	spC, err := app.RegisterSpecimen(model.Specimen{
		Collector: "Smith", Number: "5678", Institution: "P",
	})
	if err != nil {
		return fmt.Errorf("specimen C: %w", err)
	}
	if _, err := app.BindEvidence(nameC.ID, pubC.ID, spC.ID); err != nil {
		return fmt.Errorf("bind C type: %w", err)
	}
	ev2, err := app.EvaluateView(view1.ID)
	if err != nil {
		return fmt.Errorf("re-evaluate: %w", err)
	}
	if got := ev2.Roles[nameC.ID]; got != "accepted" {
		return fmt.Errorf("C role after type = %q, want accepted", got)
	}
	chk, err := app.PublishView(view1.ID)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	_, items, err := app.ViewChecklist(view1.ID)
	if err != nil {
		return fmt.Errorf("checklist: %w", err)
	}
	if len(items) != 3 {
		return fmt.Errorf("checklist items = %d, want 3", len(items))
	}

	// 7. 重启恢复：关闭数据库，重新打开同一文件，验证数据仍在。
	if err := db.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	defer db2.Close()
	app2 := service.NewApp(db2)
	restored, err := app2.Names.List()
	if err != nil {
		return fmt.Errorf("list after reopen: %w", err)
	}
	if len(restored) != 3 {
		return fmt.Errorf("restored names = %d, want 3", len(restored))
	}
	chk2, err := app2.Checklists.Get(chk.ID)
	if err != nil {
		return fmt.Errorf("restored checklist: %w", err)
	}
	if chk2.Fingerprint != chk.Fingerprint {
		return fmt.Errorf("fingerprint changed after reopen")
	}

	fmt.Printf("smoke OK: names=%d relations=%d view=%s checklist=%s fingerprint=%s\n",
		len(restored), 1, view1.ID, chk.ID, chk.Fingerprint[:12])
	return nil
}
