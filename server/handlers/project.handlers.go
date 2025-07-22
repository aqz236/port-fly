package handlers

import (
	"net/http"
	"strconv"

	"github.com/aqz236/port-fly/core/models"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) GetProjects(c *gin.Context) {
	// 查询参数
	parentIDStr := c.Query("parent_id")
	includeChildren := c.Query("include_children") == "true"
	asTree := c.Query("as_tree") == "true"

	var parentID *uint
	if parentIDStr != "" {
		id, err := strconv.ParseUint(parentIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "Invalid parent_id parameter",
			})
			return
		}
		uid := uint(id)
		parentID = &uid
	}

	if asTree {
		// 返回树状结构
		tree, err := h.storage.GetProjectTree(c.Request.Context(), parentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    tree,
		})
		return
	}

	// 如果没有特殊查询参数，使用原始的GetProjects方法
	if parentIDStr == "" && !includeChildren {
		projects, err := h.storage.GetProjects(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    projects,
		})
		return
	}

	// 返回平铺列表
	projects, err := h.storage.GetProjectsByParent(c.Request.Context(), parentID, includeChildren)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    projects,
	})
}

func (h *Handlers) CreateProject(c *gin.Context) {
	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreateProject(c.Request.Context(), &project); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    project,
	})
}

func (h *Handlers) GetProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	project, err := h.storage.GetProject(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Project not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    project,
	})
}

func (h *Handlers) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	project.ID = uint(id)
	if err := h.storage.UpdateProject(c.Request.Context(), &project); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    project,
	})
}

func (h *Handlers) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	if err := h.storage.DeleteProject(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Project deleted successfully",
	})
}

func (h *Handlers) GetProjectStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	stats, err := h.storage.GetProjectStats(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// GetProjectChildren 获取项目的直接子项目
func (h *Handlers) GetProjectChildren(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	children, err := h.storage.GetProjectChildren(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    children,
	})
}

// MoveProject 移动项目到新的父项目下
func (h *Handlers) MoveProject(c *gin.Context) {
	var params models.MoveProjectParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.MoveProject(c.Request.Context(), &params); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Project moved successfully",
	})
}
