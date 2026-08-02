package avatar

import (
	"fmt"
	"strings"
)

// CharacterLibrary 为固定授权角色库（封闭集合；禁止动态生成新脸，FR-014）。
type CharacterLibrary struct {
	byID map[string]Character
}

// NewCharacterLibrary 创建角色库；库内容随授权角色上线登记，禁运行时新增。
func NewCharacterLibrary(characters ...Character) (*CharacterLibrary, error) {
	lib := &CharacterLibrary{byID: make(map[string]Character, len(characters))}
	for _, c := range characters {
		if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.LicenseRef) == "" {
			return nil, fmt.Errorf("%w: 角色必须含 ID 与授权凭证引用", ErrCharacterNotFound)
		}
		if _, ok := lib.byID[c.ID]; ok {
			return nil, fmt.Errorf("%w: 角色 %q 重复", ErrCharacterNotFound, c.ID)
		}
		lib.byID[c.ID] = c
	}
	return lib, nil
}

// Validate 校验角色 ID 属于授权库（未知角色拒绝，禁止每场生成新脸）。
func (l *CharacterLibrary) Validate(characterID string) error {
	if _, ok := l.byID[characterID]; !ok {
		return fmt.Errorf("%w: %q", ErrCharacterNotFound, characterID)
	}
	return nil
}

// Get 返回角色条目。
func (l *CharacterLibrary) Get(characterID string) (Character, error) {
	c, ok := l.byID[characterID]
	if !ok {
		return Character{}, ErrCharacterNotFound
	}
	return c, nil
}

// List 返回授权角色 ID 列表（稳定顺序）。
func (l *CharacterLibrary) List() []string {
	out := make([]string, 0, len(l.byID))
	for id := range l.byID {
		out = append(out, id)
	}
	return out
}

// SyntheticCharacters 为合成授权角色（开发/测试用；正式角色随授权登记）。
func SyntheticCharacters() []Character {
	return []Character{
		{ID: "avatar-zh-01", LicenseRef: "AVATAR_CHARACTER_LICENSE_REF", ProfileHash: "synthetic-zh-01"},
		{ID: "avatar-en-01", LicenseRef: "AVATAR_CHARACTER_LICENSE_REF", ProfileHash: "synthetic-en-01"},
	}
}
