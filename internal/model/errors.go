package model

import "errors"

// 领域错误统一集中定义，方便 httpapi 层做错误码映射，
// 避免在业务包内散落裸错误字符串。

var (
	// ErrNotFound 表示请求的实体不存在。
	ErrNotFound = errors.New("entity not found")

	// ErrConflict 表示唯一键或业务约束冲突（如幂等指纹重复）。
	ErrConflict = errors.New("conflict")

	// ErrInvalidArgument 表示请求参数不合法。
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrCycleSynonym 表示拒绝循环同物异名：归并边若闭合会形成环。
	ErrCycleSynonym = errors.New("cyclic synonym rejected")

	// ErrMutuallyExclusiveType 表示同一模式标本被指向互斥的接受名。
	ErrMutuallyExclusiveType = errors.New("same specimen points to mutually exclusive accepted names")

	// ErrDateUnsortable 表示发表证据的日期区间无法排序（重叠或缺失）。
	ErrDateUnsortable = errors.New("publication dates unsortable")

	// ErrCrossViewMerge 表示试图跨分类观点静默合并名称簇。
	ErrCrossViewMerge = errors.New("cross-view silent merge rejected")

	// ErrFrozenChecklist 表示试图直接编辑已发布（冻结）的清单快照。
	ErrFrozenChecklist = errors.New("published checklist is frozen")

	// ErrIllegalTransition 表示实体状态机不允许的转移。
	ErrIllegalTransition = errors.New("illegal state transition")
)
