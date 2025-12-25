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

func HandleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}

	var password string
	row := dbx.QueryRowContext(c.Request.Context(), `
		SELECT user_id, password
		FROM users
		WHERE username = ? AND status = 1
	`, req.Username)
	if err := row.Scan(new(int), &password); err != nil {
		if err == sql.ErrNoRows {
			role := "USER"
			total := int64(0)
			if req.Username == "admin" {
				role = "ADMIN"
				total = 10000
			}
			res, err := dbx.ExecContext(c.Request.Context(), `
				INSERT INTO users (username, password, nickname, role, status, total_quota, remaining_quota)
				VALUES (?, ?, ?, ?, 1, ?, ?)
			`, req.Username, req.Password, req.Username, role, total, total)
			if err != nil {
				c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to create user", ErrCode: 500})
				return
			}
			_, _ = res.LastInsertId()
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

	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}

	var totalCount int
	if err := dbx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM users`).Scan(&totalCount); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	offset := (page - 1) * pageSize
	rows, err := dbx.QueryContext(c.Request.Context(), `
		SELECT user_id, username, nickname, role, total_quota, remaining_quota
		FROM users
		ORDER BY user_id DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.UserID, &u.Username, &u.Nickname, &u.Role, &u.TotalQuota, &u.RemainingQuota); err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
			return
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}

	var exists int
	if err := dbx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM users WHERE username = ?`, req.Username).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	if exists > 0 {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "user exists", ErrCode: 400})
		return
	}

	res, err := dbx.ExecContext(c.Request.Context(), `
		INSERT INTO users (username, password, nickname, role, status, total_quota, remaining_quota)
		VALUES (?, ?, ?, ?, 1, ?, ?)
	`, req.Username, req.Password, req.Nickname, req.Role, req.TotalQuota, req.RemainingQuota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	newID, _ := res.LastInsertId()

	newUser := User{
		UserID:         int(newID),
		Username:       req.Username,
		Nickname:       req.Nickname,
		Role:           req.Role,
		TotalQuota:     req.TotalQuota,
		RemainingQuota: req.RemainingQuota,
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

	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	res, err := dbx.ExecContext(c.Request.Context(), `
		UPDATE users SET total_quota = ?, remaining_quota = ?
		WHERE user_id = ?
	`, req.Quota, req.Quota, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	var current string
	row := dbx.QueryRowContext(c.Request.Context(), `SELECT password FROM users WHERE username = ?`, username)
	if err := row.Scan(&current); err != nil {
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
	if _, err := dbx.ExecContext(c.Request.Context(), `UPDATE users SET password = ? WHERE username = ?`, req.NewPassword, username); err != nil {
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
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
	res, err := dbx.ExecContext(c.Request.Context(), `
		INSERT INTO conversations (user_id, title, status, llm_model, system_prompt)
		VALUES (?, ?, 'ACTIVE', ?, ?)
	`, userID, req.Title, llmModel, systemPrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	convID, _ := res.LastInsertId()
	conv := gin.H{
		"conversation_id": int(convID),
		"title":           req.Title,
		"status":          "ACTIVE",
		"llm_model":       llmModel,
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	res, err := dbx.ExecContext(c.Request.Context(), `
		UPDATE conversations SET title = ?
		WHERE conversation_id = ? AND user_id = ?
	`, req.Title, convID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	res, err := dbx.ExecContext(c.Request.Context(), `
		UPDATE conversations SET status = 'DELETED'
		WHERE conversation_id = ? AND user_id = ?
	`, convID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
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

	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}

	var totalCount int
	if err := dbx.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*) FROM messages m
		JOIN conversations c ON m.conversation_id = c.conversation_id
		WHERE c.user_id = ? AND m.conversation_id = ?
	`, userID, convID).Scan(&totalCount); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	offset := (page - 1) * pageSize
	rows, err := dbx.QueryContext(c.Request.Context(), `
		SELECT m.message_id, m.sender_type, m.content_type, m.content, m.token_total
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.conversation_id
		WHERE c.user_id = ? AND m.conversation_id = ?
		ORDER BY m.created_at ASC
		LIMIT ? OFFSET ?
	`, userID, convID, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	defer rows.Close()

	messageIDs := make([]int, 0)
	type msgRow struct {
		MessageID   int
		SenderType int
		ContentType string
		Content      string
		TokenTotal   int
	}
	items := make([]msgRow, 0)
	for rows.Next() {
		var m msgRow
		if err := rows.Scan(&m.MessageID, &m.SenderType, &m.ContentType, &m.Content, &m.TokenTotal); err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
			return
		}
		messageIDs = append(messageIDs, m.MessageID)
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	attachmentsMap, err := loadAttachmentsMap(c.Request.Context(), dbx, messageIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	messages := make([]gin.H, 0, len(items))
	for _, m := range items {
		messages = append(messages, gin.H{
			"message_id":   m.MessageID,
			"sender_type":  senderTypeToAPI(m.SenderType),
			"content_type": m.ContentType,
			"content":      m.Content,
			"token_total":  m.TokenTotal,
			"attachments":  attachmentsMap[m.MessageID],
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	res, err := dbx.ExecContext(c.Request.Context(), `DELETE FROM users WHERE user_id = ?`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

type PromptPreset struct {
	PromptPresetID int    `json:"prompt_preset_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Content        string `json:"content"`
}

type ConversationInfo struct {
	ConversationID int    `json:"conversation_id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	LLMModel       string `json:"llm_model"`
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}

	llmModel := "unknown"
	if client := llm.Get(); client != nil && client.Model() != "" {
		llmModel = client.Model()
	}
	uploadConvID, err := getOrCreateUploadConversation(c.Request.Context(), userID, llmModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	uploadMsgID, err := createUploadMessage(c.Request.Context(), uploadConvID)
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

	res, err := dbx.ExecContext(c.Request.Context(), `
		INSERT INTO message_attachments (message_id, attachment_type, mime_type, storage_type, url_or_path, duration_ms)
		VALUES (?, 'FILE', ?, 'LOCAL', ?, NULL)
	`, uploadMsgID, mimeType, storePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	attachID, _ := res.LastInsertId()

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"attachment": gin.H{
			"attachment_id":   int(attachID),
			"attachment_type": "FILE",
			"mime_type":       mimeType,
			"url_or_path":     storePath,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	})
}

func HandleGetPromptPreset(c *gin.Context) {
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	rows, err := dbx.QueryContext(c.Request.Context(), `
		SELECT prompt_preset_id, name, description, content
		FROM prompt_presets
		ORDER BY prompt_preset_id DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	defer rows.Close()

	list := make([]gin.H, 0)
	for rows.Next() {
		var p PromptPreset
		if err := rows.Scan(&p.PromptPresetID, &p.Name, &p.Description, &p.Content); err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
			return
		}
		list = append(list, gin.H{
			"prompt_preset_id": p.PromptPresetID,
			"name":             p.Name,
			"description":      p.Description,
			"content":          p.Content,
		})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
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
	user, err := getUserByUsername(c.Request.Context(), username)
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	rows, err := dbx.QueryContext(c.Request.Context(), `
		SELECT conversation_id, title, status, llm_model
		FROM conversations
		WHERE user_id = ?
		ORDER BY conversation_id DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	defer rows.Close()

	conversations := make([]ConversationInfo, 0)
	for rows.Next() {
		var info ConversationInfo
		if err := rows.Scan(&info.ConversationID, &info.Title, &info.Status, &info.LLMModel); err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
			return
		}
		conversations = append(conversations, info)
	}
	if err := rows.Err(); err != nil {
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	rows, err := dbx.QueryContext(c.Request.Context(), `
		SELECT prompt_preset_id, name, description, content
		FROM prompt_presets
		ORDER BY prompt_preset_id DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	defer rows.Close()

	list := make([]PromptPreset, 0)
	for rows.Next() {
		var p PromptPreset
		if err := rows.Scan(&p.PromptPresetID, &p.Name, &p.Description, &p.Content); err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
			return
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	_, err = dbx.ExecContext(c.Request.Context(), `
		INSERT INTO prompt_presets (name, description, content)
		VALUES (?, ?, ?)
	`, req.Name, req.Description, req.Content)
	if err != nil {
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
	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}
	res, err := dbx.ExecContext(c.Request.Context(), `DELETE FROM prompt_presets WHERE prompt_preset_id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}
