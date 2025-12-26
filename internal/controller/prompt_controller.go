package controller

import (
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleGetPromptPreset 获取提示词列表（聊天模块）。
func HandleGetPromptPreset(c *gin.Context) {
	presets, err := service.ListPromptPresets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	list := make([]gin.H, 0, len(presets))
	for _, p := range presets {
		list = append(list, gin.H{
			"prompt_preset_id": p.PromptPresetID,
			"name":             p.Name,
			"description":      p.Description,
			"content":          p.Content,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":        "success",
		"err_code":       0,
		"prompt_presets": list,
	})
}

// HandleAdminGetPromptPresets 获取提示词列表（管理端）。
func HandleAdminGetPromptPresets(c *gin.Context) {
	list, err := service.ListPromptPresets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":        "success",
		"err_code":       0,
		"prompt_presets": list,
	})
}

// HandleAdminCreatePromptPreset 新增提示词。
func HandleAdminCreatePromptPreset(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		Content     string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	if err := service.CreatePromptPreset(c.Request.Context(), req.Name, req.Description, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleAdminDeletePromptPreset 删除提示词。
func HandleAdminDeletePromptPreset(c *gin.Context) {
	idStr := c.Param("prompt_preset_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid prompt_preset_id", ErrCode: 400})
		return
	}
	deleted, err := service.DeletePromptPreset(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	if !deleted {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}
