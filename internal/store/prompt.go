package store

import "context"

// ListPromptPresets 获取提示词列表。
func ListPromptPresets(ctx context.Context) ([]PromptPreset, error) {
	dbx, err := GetDB()
	if err != nil {
		return nil, err
	}
	rows, err := dbx.QueryContext(ctx, `
		SELECT prompt_preset_id, name, description, content
		FROM prompt_presets
		ORDER BY prompt_preset_id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]PromptPreset, 0)
	for rows.Next() {
		var p PromptPreset
		if err := rows.Scan(&p.PromptPresetID, &p.Name, &p.Description, &p.Content); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// CreatePromptPreset 创建提示词。
func CreatePromptPreset(ctx context.Context, name, description, content string) error {
	dbx, err := GetDB()
	if err != nil {
		return err
	}
	_, err = dbx.ExecContext(ctx, `
		INSERT INTO prompt_presets (name, description, content)
		VALUES (?, ?, ?)
	`, name, description, content)
	return err
}

// DeletePromptPreset 删除提示词，返回是否命中。
func DeletePromptPreset(ctx context.Context, id int) (bool, error) {
	dbx, err := GetDB()
	if err != nil {
		return false, err
	}
	res, err := dbx.ExecContext(ctx, `DELETE FROM prompt_presets WHERE prompt_preset_id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}
