package handler

import (
	"github.com/gin-gonic/gin"

	"blog-api/internal/response"
	"blog-api/internal/service"
)

type TodoHandler struct {
	todos *service.TodoService
}

func NewTodoHandler(todos *service.TodoService) *TodoHandler {
	return &TodoHandler{todos: todos}
}

func (h *TodoHandler) PublicList(c *gin.Context) {
	items, err := h.todos.List()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *TodoHandler) AdminList(c *gin.Context) {
	items, err := h.todos.List()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *TodoHandler) Create(c *gin.Context) {
	var in service.TodoInput
	if !bindJSON(c, &in) {
		return
	}
	todo, err := h.todos.Create(in)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, todo)
}

func (h *TodoHandler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in service.TodoInput
	if !bindJSON(c, &in) {
		return
	}
	todo, err := h.todos.Update(id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, todo)
}

func (h *TodoHandler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.todos.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *TodoHandler) Reorder(c *gin.Context) {
	var in service.ReorderTodosInput
	if !bindJSON(c, &in) {
		return
	}
	if err := h.todos.Reorder(in); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}
