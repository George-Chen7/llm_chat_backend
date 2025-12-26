package store

// User 用户信息。
type User struct {
	UserID         int    `json:"user_id"`
	Username       string `json:"username"`
	Nickname       string `json:"nickname"`
	Role           string `json:"role"`
	TotalQuota     int    `json:"total_quota"`
	RemainingQuota int    `json:"remaining_quota"`
}

// ConversationInfo 对话概要信息。
type ConversationInfo struct {
	ConversationID int    `json:"conversation_id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	LLMModel       string `json:"llm_model"`
}

// PromptPreset 提示词预设。
type PromptPreset struct {
	PromptPresetID int    `json:"prompt_preset_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Content        string `json:"content"`
}

// MessageRow 消息基础字段。
type MessageRow struct {
	MessageID   int
	SenderType  int
	ContentType string
	Content     string
	TokenTotal  int
}

// AttachmentInfo 附件信息。
type AttachmentInfo struct {
	AttachmentID   int      `json:"attachment_id"`
	AttachmentType string   `json:"attachment_type"`
	MimeType       string   `json:"mime_type"`
	URLOrPath      string   `json:"url_or_path"`
	DurationMS     *float64 `json:"duration_ms,omitempty"`
}
