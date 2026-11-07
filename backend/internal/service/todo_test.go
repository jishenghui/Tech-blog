package service_test

import (
	"testing"

	"blog-api/internal/service"
)

func TestTodoCRUDAndReorder(t *testing.T) {
	env := newTestEnv(t)

	initial, err := env.todos.List()
	if err != nil || len(initial) != 0 {
		t.Fatalf("初始待办列表应为空: err=%v len=%d", err, len(initial))
	}

	first, err := env.todos.Create(service.TodoInput{Content: "写 M4 页面"})
	if err != nil {
		t.Fatalf("创建待办失败: %v", err)
	}
	second, err := env.todos.Create(service.TodoInput{Content: "补后端测试", Done: true})
	if err != nil {
		t.Fatalf("创建待办失败: %v", err)
	}

	items, err := env.todos.List()
	if err != nil || len(items) != 2 || items[0].ID != first.ID || items[1].ID != second.ID {
		t.Fatalf("待办列表顺序不正确: err=%v items=%+v", err, items)
	}

	updated, err := env.todos.Update(first.ID, service.TodoInput{
		Content: "写 M4 页面与 POC",
		Done:    true,
	})
	if err != nil {
		t.Fatalf("更新待办失败: %v", err)
	}
	if updated.Content != "写 M4 页面与 POC" || !updated.Done {
		t.Fatalf("待办更新结果不正确: %+v", updated)
	}

	if err := env.todos.Reorder(service.ReorderTodosInput{Items: []service.ReorderTodoItem{
		{ID: second.ID, SortOrder: 0},
		{ID: first.ID, SortOrder: 1},
	}}); err != nil {
		t.Fatalf("待办排序失败: %v", err)
	}
	items, _ = env.todos.List()
	if items[0].ID != second.ID || items[1].ID != first.ID {
		t.Fatalf("待办排序结果不正确: %+v", items)
	}

	if err := env.todos.Delete(first.ID); err != nil {
		t.Fatalf("删除待办失败: %v", err)
	}
	items, _ = env.todos.List()
	if len(items) != 1 || items[0].ID != second.ID {
		t.Fatalf("删除后待办列表不正确: %+v", items)
	}

	_, err = env.todos.Create(service.TodoInput{Content: "   "})
	requireErrorContains(t, err, "不能为空")

	requireErrorContains(t, env.todos.Reorder(service.ReorderTodosInput{Items: []service.ReorderTodoItem{
		{ID: second.ID, SortOrder: 1},
		{ID: second.ID, SortOrder: 2},
	}}), "重复待办")
}
