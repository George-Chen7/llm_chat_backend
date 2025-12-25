package handler

import (
	"net/http"
	"strconv"
	"time"

	"backend/internal/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	UserID         int    `json:"user_id"`
	Username       string `json:"username"`
	Nickname       string `json:"nickname"`
	Role           string `json:"role"`
	TotalQuota     int    `json:"total_quota"`
	RemainingQuota int    `json:"remaining_quota"`
}

var mockUserList = []User{
	{UserID: 1, Username: "admin", Nickname: "Admin", Role: "ADMIN", TotalQuota: 10000, RemainingQuota: 9500},
	{UserID: 2, Username: "yang", Nickname: "Yang", Role: "USER", TotalQuota: 5000, RemainingQuota: 5000},
}
var nextUserID = 3

func HandleLogin(c *gin.Context) {
	username := "admin"
	claims := middlewares.MyClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(middlewares.JWTSecret)

	c.JSON(http.StatusOK, gin.H{
		"err_msg":   "success",
		"err_code":  0,
		"jwt_token": tokenString,
	})
}

func HandleGetUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current_page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	totalCount := len(mockUserList)
	totalPage := (totalCount + pageSize - 1) / pageSize

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_page":   totalPage,
		"total_count":  totalCount,
		"current_page": page,
		"page_size":    pageSize,
		"users":        mockUserList[start:end],
	})
}

func HandleAddUser(c *gin.Context) {
	var req struct {
		Username       string `json:"username" binding:"required"`
		Nickname       string `json:"nickname" binding:"required"`
		Role           string `json:"role" binding:"required"`
		TotalQuota     int    `json:"total_quota"`
		RemainingQuota int    `json:"remaining_quota"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}

	for _, u := range mockUserList {
		if u.Username == req.Username {
			c.JSON(http.StatusOK, BaseResponse{ErrMsg: "user exists", ErrCode: 400})
			return
		}
	}

	newUser := User{
		UserID:         nextUserID,
		Username:       req.Username,
		Nickname:       req.Nickname,
		Role:           req.Role,
		TotalQuota:     req.TotalQuota,
		RemainingQuota: req.RemainingQuota,
	}
	mockUserList = append(mockUserList, newUser)
	nextUserID++

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"user":     newUser,
	})
}

func HandleSetQuota(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, _ := strconv.Atoi(userIDStr)

	var req struct {
		Quota int `json:"quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}

	for i := range mockUserList {
		if mockUserList[i].UserID == userID {
			mockUserList[i].TotalQuota = req.Quota
			c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
			return
		}
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
}

func HandleSetPassword(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleRefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":   "success",
		"err_code":  0,
		"jwt_token": "new_token",
	})
}

func HandleNewChat(c *gin.Context) {
	var req struct {
		Title        string `json:"title" binding:"required"`
		SystemPrompt string `json:"system_prompt"`
	}
	_ = c.ShouldBindJSON(&req)

	conv := gin.H{
		"conversation_id": 1001,
		"title":           req.Title,
		"status":          "ACTIVE",
		"llm_model":       "gpt-4o",
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"conversation": conv,
	})
}

func HandleRenameChat(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleDeleteChat(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleGetChatHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current_page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	msgs := []gin.H{
		{
			"message_id":   1,
			"sender_type":  "USER",
			"content_type": "TEXT",
			"content":      "Hello",
			"token_total":  5,
			"attachments":  []any{},
		},
		{
			"message_id":   2,
			"sender_type":  "ASSISTANT",
			"content_type": "TEXT",
			"content":      "Hi there!",
			"token_total":  8,
			"attachments":  []any{},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_page":   1,
		"total_count":  2,
		"current_page": page,
		"page_size":    pageSize,
		"messages":     msgs,
	})
}

func HandleDeleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

type PromptPreset struct {
	PromptPresetID int    `json:"prompt_preset_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Content        string `json:"content"`
}

var mockPromptPresets = []PromptPreset{
	{PromptPresetID: 1, Name: "Assistant", Description: "Helpful assistant", Content: "You are a helpful assistant."},
	{PromptPresetID: 2, Name: "Coder", Description: "Code generator", Content: "Write clean and robust code."},
}
var nextPromptPresetID = 3

type ConversationInfo struct {
	ConversationID int    `json:"conversation_id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	LLMModel       string `json:"llm_model"`
}

var mockMeConversations = []ConversationInfo{
	{ConversationID: 101, Title: "Welcome", Status: "ACTIVE", LLMModel: "gpt-4o"},
	{ConversationID: 102, Title: "Daily", Status: "ARCHIVED", LLMModel: "gpt-4o-mini"},
}

func HandleUploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err == nil && file != nil {
		defer file.Close()
	}
	aid := time.Now().UnixNano() % 1000000
	name := ""
	if header != nil {
		name = header.Filename
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"attachment": gin.H{
			"attachment_id":   aid,
			"attachment_type": "FILE",
			"mime_type":       "application/octet-stream",
			"url_or_path":     "/uploads/" + name,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	})
}

func HandleGetPromptPreset(c *gin.Context) {
	list := make([]gin.H, 0, len(mockPromptPresets))
	for _, p := range mockPromptPresets {
		list = append(list, gin.H{
			"name":        p.Name,
			"description": p.Description,
			"content":     p.Content,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":        "success",
		"err_code":       0,
		"prompt_presets": list,
	})
}

func HandleGetMeInfo(c *gin.Context) {
	username, _ := c.Get("username")
	u := "admin"
	if s, ok := username.(string); ok && s != "" {
		u = s
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"user": gin.H{
			"user_id":         1,
			"username":        u,
			"nickname":        "Admin",
			"role":            "ADMIN",
			"total_quota":     10000,
			"remaining_quota": 9500,
		},
	})
}

func HandleGetMeConversations(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":       "success",
		"err_code":      0,
		"conversations": mockMeConversations,
	})
}

func HandleAdminGetPromptPresets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":        "success",
		"err_code":       0,
		"prompt_presets": mockPromptPresets,
	})
}

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
	mockPromptPresets = append(mockPromptPresets, PromptPreset{
		PromptPresetID: nextPromptPresetID,
		Name:           req.Name,
		Description:    req.Description,
		Content:        req.Content,
	})
	nextPromptPresetID++
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleAdminDeletePromptPreset(c *gin.Context) {
	idStr := c.Param("prompt_preset_id")
	id, _ := strconv.Atoi(idStr)
	found := false
	for i := range mockPromptPresets {
		if mockPromptPresets[i].PromptPresetID == id {
			mockPromptPresets = append(mockPromptPresets[:i], mockPromptPresets[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}
