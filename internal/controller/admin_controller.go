package controller

import (
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleGetUserList 获取用户列表。
func HandleGetUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current_page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	users, totalCount, err := service.ListUsers(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	totalPage := (totalCount + pageSize - 1) / pageSize
	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_page":   totalPage,
		"total_count":  totalCount,
		"current_page": page,
		"page_size":    pageSize,
		"users":        users,
	})
}

// HandleAddUser 添加用户。
func HandleAddUser(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		Nickname   string `json:"nickname" binding:"required"`
		Role       string `json:"role" binding:"required"`
		TotalQuota int    `json:"total_quota" binding:"required"`
		UsedQuota  int    `json:"used_quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	user, err := service.CreateUser(c.Request.Context(), req.Username, req.Password, req.Nickname, req.Role, req.TotalQuota, req.UsedQuota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"user":     user,
	})
}

// HandleSetQuota 设置额度。
func HandleSetQuota(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid user_id", ErrCode: 400})
		return
	}
	var req struct {
		Quota int `json:"quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	updated, err := service.SetUserQuota(c.Request.Context(), userID, req.Quota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	if !updated {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleDeleteUser 删除用户。
func HandleDeleteUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid user_id", ErrCode: 400})
		return
	}
	deleted, err := service.DeleteUser(c.Request.Context(), userID)
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
