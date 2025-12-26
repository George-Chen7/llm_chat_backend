package handler

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"backend/internal/llm"
	"backend/internal/middlewares"
	"backend/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func HandleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	_, password, err := store.GetUserPassword(c.Request.Context(), req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			role := "USER"
			total := int64(0)
			if req.Username == "admin" {
				role = "ADMIN"
				total = 10000
			}
			if _, err := store.CreateUser(c.Request.Context(), req.Username, req.Password, req.Username, role, total, total); err != nil {
				c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to create user", ErrCode: 500})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
			return
		}
	} else if password != req.Password {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "invalid credentials", ErrCode: 401})
		return
	}

	username := req.Username
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
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	users, totalCount, err := store.ListUsers(c.Request.Context(), page, pageSize)
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

func HandleAddUser(c *gin.Context) {
	var req struct {
		Username       string `json:"username" binding:"required"`
		Password       string `json:"password" binding:"required"`
		Nickname       string `json:"nickname" binding:"required"`
		Role           string `json:"role" binding:"required"`
		TotalQuota     int    `json:"total_quota" binding:"required"`
		RemainingQuota int    `json:"remaining_quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	exists, err := store.CountUsersByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	if exists > 0 {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "user exists", ErrCode: 400})
		return
	}

	newUser, err := store.CreateUserWithQuota(c.Request.Context(), req.Username, req.Password, req.Nickname, req.Role, req.TotalQuota, req.RemainingQuota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"user":     newUser,
	})
}

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

	updated, err := store.SetUserQuota(c.Request.Context(), userID, req.Quota)
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

func HandleSetPassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	username, err := getUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	_, current, err := store.GetUserPassword(c.Request.Context(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "user not found", ErrCode: 401})
			return
		}
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	if current != req.OldPassword {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "old password incorrect", ErrCode: 401})
		return
	}
	if err := store.UpdateUserPassword(c.Request.Context(), username, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleRefreshToken(c *gin.Context) {
	username, err := getUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
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

func HandleNewChat(c *gin.Context) {
	var req struct {
		Title        string `json:"title" binding:"required"`
		SystemPrompt string `json:"system_prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	llmModel := "unknown"
	if client := llm.Get(); client != nil && client.Model() != "" {
		llmModel = client.Model()
	}

	var systemPrompt sql.NullInt64
	if req.SystemPrompt != "" {
		if v, err := strconv.Atoi(req.SystemPrompt); err == nil {
			systemPrompt = sql.NullInt64{Int64: int64(v), Valid: true}
		}
	}
	convInfo, err := store.CreateConversation(c.Request.Context(), userID, req.Title, llmModel, systemPrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	conv := gin.H{
		"conversation_id": convInfo.ConversationID,
		"title":           convInfo.Title,
		"status":          convInfo.Status,
		"llm_model":       convInfo.LLMModel,
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"conversation": conv,
	})
}

func HandleRenameChat(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing conversation_id", ErrCode: 400})
		return
	}
	convID, err := strconv.Atoi(conversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid conversation_id", ErrCode: 400})
		return
	}
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	updated, err := store.RenameConversation(c.Request.Context(), convID, userID, req.Title)
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

func HandleDeleteChat(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing conversation_id", ErrCode: 400})
		return
	}
	convID, err := strconv.Atoi(conversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid conversation_id", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	deleted, err := store.DeleteConversation(c.Request.Context(), convID, userID)
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

func HandleGetChatHistory(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing conversation_id", ErrCode: 400})
		return
	}
	convID, err := strconv.Atoi(conversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid conversation_id", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("current_page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	totalCount, err := store.CountMessages(c.Request.Context(), userID, convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	items, messageIDs, err := store.ListMessages(c.Request.Context(), userID, convID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	attachmentsMap, err := store.LoadAttachmentsMap(c.Request.Context(), messageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	messages := make([]gin.H, 0, len(items))
	for _, m := range items {
		attachments := make([]gin.H, 0)
		for _, a := range attachmentsMap[m.MessageID] {
			attachments = append(attachments, gin.H{
				"attachment_id":   a.AttachmentID,
				"attachment_type": a.AttachmentType,
				"mime_type":       a.MimeType,
				"url_or_path":     a.URLOrPath,
				"duration_ms":     a.DurationMS,
			})
		}
		messages = append(messages, gin.H{
			"message_id":   m.MessageID,
			"sender_type":  senderTypeToAPI(m.SenderType),
			"content_type": m.ContentType,
			"content":      m.Content,
			"token_total":  m.TokenTotal,
			"attachments":  attachments,
		})
	}

	totalPage := (totalCount + pageSize - 1) / pageSize
	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_page":   totalPage,
		"total_count":  totalCount,
		"current_page": page,
		"page_size":    pageSize,
		"messages":     messages,
	})
}

func HandleDeleteUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid user_id", ErrCode: 400})
		return
	}
	deleted, err := store.DeleteUser(c.Request.Context(), userID)
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

func HandleUploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing file", ErrCode: 400})
		return
	}
	defer file.Close()

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	llmModel := "unknown"
	if client := llm.Get(); client != nil && client.Model() != "" {
		llmModel = client.Model()
	}
	uploadConvID, err := store.GetOrCreateUploadConversation(c.Request.Context(), userID, llmModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	uploadMsgID, err := store.CreateUploadMessage(c.Request.Context(), uploadConvID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	if err := os.MkdirAll("uploads", 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to create upload dir", ErrCode: 500})
		return
	}

	filename := "upload.bin"
	mimeType := "application/octet-stream"
	if header != nil {
		if header.Filename != "" {
			filename = filepath.Base(header.Filename)
		}
		if header.Header != nil {
			if v := header.Header.Get("Content-Type"); v != "" {
				mimeType = v
			}
		}
	}
	storeName := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + filename
	storePath := filepath.Join("uploads", storeName)
	publicPath := "/uploads/" + storeName
	out, err := os.Create(storePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to save file", ErrCode: 500})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to save file", ErrCode: 500})
		return
	}

	attachID, err := store.CreateAttachment(c.Request.Context(), uploadMsgID, "FILE", mimeType, "LOCAL", publicPath, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"attachment": gin.H{
			"attachment_id":   int(attachID),
			"attachment_type": "FILE",
			"mime_type":       mimeType,
			"url_or_path":     publicPath,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	})
}

func HandleGetPromptPreset(c *gin.Context) {
	presets, err := store.ListPromptPresets(c.Request.Context())
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

func HandleGetMeInfo(c *gin.Context) {
	username, err := getUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	user, err := store.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "user not found", ErrCode: 401})
			return
		}
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"user": gin.H{
			"user_id":         user.UserID,
			"username":        user.Username,
			"nickname":        user.Nickname,
			"role":            user.Role,
			"total_quota":     user.TotalQuota,
			"remaining_quota": user.RemainingQuota,
		},
	})
}

func HandleGetMeConversations(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	conversations, err := store.ListConversationsByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":       "success",
		"err_code":      0,
		"conversations": conversations,
	})
}

func HandleAdminGetPromptPresets(c *gin.Context) {
	list, err := store.ListPromptPresets(c.Request.Context())
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
	if err := store.CreatePromptPreset(c.Request.Context(), req.Name, req.Description, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleAdminDeletePromptPreset(c *gin.Context) {
	idStr := c.Param("prompt_preset_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid prompt_preset_id", ErrCode: 400})
		return
	}
	deleted, err := store.DeletePromptPreset(c.Request.Context(), id)
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
