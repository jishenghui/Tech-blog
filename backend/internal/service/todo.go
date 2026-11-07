package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"blog-api/internal/apperr"
	"blog-api/internal/model"
)

type TodoService struct {
	db *gorm.DB
}

func NewTodoService(db *gorm.DB) *TodoService {
	return &TodoService{db: db}
}

type TodoInput struct {
	Content   string     `json:"content"`
	Done      bool       `json:"done"`
	SortOrder *int       `json:"sort_order"`
	DueDate   *time.Time `json:"due_date"`
}

type TodoView struct {
	ID        uint       `json:"id"`
	Content   string     `json:"content"`
	Done      bool       `json:"done"`
	SortOrder int        `json:"sort_order"`
	DueDate   *time.Time `json:"due_date"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ReorderTodoItem struct {
	ID        uint `json:"id"`
	SortOrder int  `json:"sort_order"`
}

type ReorderTodosInput struct {
	Items []ReorderTodoItem `json:"items"`
}

func (s *TodoService) List() ([]TodoView, error) {
	var todos []model.Todo
	if err := s.db.Order("sort_order, id").Find(&todos).Error; err != nil {
		return nil, apperr.Internal("查询待办失败", err)
	}
	items := make([]TodoView, 0, len(todos))
	for _, todo := range todos {
		items = append(items, toTodoView(todo))
	}
	return items, nil
}

func (s *TodoService) Create(in TodoInput) (*model.Todo, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, apperr.BadRequest("待办内容不能为空")
	}
	if utf8.RuneCountInString(content) > 1000 {
		return nil, apperr.BadRequest("待办内容不能超过 1000 个字符")
	}

	var todo model.Todo
	err := s.db.Transaction(func(tx *gorm.DB) error {
		order := 0
		if in.SortOrder != nil {
			order = *in.SortOrder
		} else {
			next, err := nextTodoSortOrder(tx)
			if err != nil {
				return err
			}
			order = next
		}
		todo = model.Todo{
			Content:   content,
			Done:      in.Done,
			SortOrder: order,
			DueDate:   in.DueDate,
		}
		if err := tx.Create(&todo).Error; err != nil {
			return apperr.Internal("创建待办失败", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func (s *TodoService) Update(id uint, in TodoInput) (*model.Todo, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, apperr.BadRequest("待办内容不能为空")
	}
	if utf8.RuneCountInString(content) > 1000 {
		return nil, apperr.BadRequest("待办内容不能超过 1000 个字符")
	}

	var todo model.Todo
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&todo, id).Error; err != nil {
			return todoNotFound(err)
		}
		updates := map[string]any{
			"content":  content,
			"done":     in.Done,
			"due_date": in.DueDate,
		}
		if in.SortOrder != nil {
			updates["sort_order"] = *in.SortOrder
		}
		if err := tx.Model(&todo).Updates(updates).Error; err != nil {
			return apperr.Internal("更新待办失败", err)
		}
		if err := tx.First(&todo, id).Error; err != nil {
			return apperr.Internal("重新查询待办失败", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func (s *TodoService) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var todo model.Todo
		if err := tx.First(&todo, id).Error; err != nil {
			return todoNotFound(err)
		}
		if err := tx.Delete(&todo).Error; err != nil {
			return apperr.Internal("删除待办失败", err)
		}
		return nil
	})
}

func (s *TodoService) Reorder(in ReorderTodosInput) error {
	if len(in.Items) == 0 {
		return apperr.BadRequest("排序列表不能为空")
	}
	seen := make(map[uint]struct{}, len(in.Items))
	for _, item := range in.Items {
		if item.ID == 0 {
			return apperr.BadRequest("待办 ID 不能为空")
		}
		if _, ok := seen[item.ID]; ok {
			return apperr.BadRequest("排序列表中存在重复待办")
		}
		seen[item.ID] = struct{}{}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range in.Items {
			var todo model.Todo
			if err := tx.First(&todo, item.ID).Error; err != nil {
				return todoNotFound(err)
			}
			if err := tx.Model(&todo).Update("sort_order", item.SortOrder).Error; err != nil {
				return apperr.Internal("更新待办排序失败", err)
			}
		}
		return nil
	})
}

func toTodoView(todo model.Todo) TodoView {
	return TodoView{
		ID:        todo.ID,
		Content:   todo.Content,
		Done:      todo.Done,
		SortOrder: todo.SortOrder,
		DueDate:   todo.DueDate,
		CreatedAt: todo.CreatedAt,
		UpdatedAt: todo.UpdatedAt,
	}
}

func nextTodoSortOrder(tx *gorm.DB) (int, error) {
	var maxOrder int
	if err := tx.Model(&model.Todo{}).Select("COALESCE(MAX(sort_order), -1)").Row().Scan(&maxOrder); err != nil {
		return 0, apperr.Internal("计算待办排序位置失败", err)
	}
	return maxOrder + 1, nil
}

func todoNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.NotFound("待办不存在")
	}
	return apperr.Internal("查询待办失败", err)
}
