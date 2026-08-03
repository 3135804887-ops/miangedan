// Package org 提供机构租户内存存储（TASK-070；开发/测试；生产 PostgreSQL）。
package org

import "sync"

// MemoryStore 为内存版机构存储。
type MemoryStore struct {
	mu           sync.RWMutex
	orgs         map[string]Org
	orgIDem      map[string]Org
	members      map[string]OrgMember
	membersByOrg map[string][]OrgMember
	invitations  map[string]Invitation
	invIDem      map[string]Invitation
	audits       map[string][]AuditEntry
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		orgs:         make(map[string]Org),
		orgIDem:      make(map[string]Org),
		members:      make(map[string]OrgMember),
		membersByOrg: make(map[string][]OrgMember),
		invitations:  make(map[string]Invitation),
		invIDem:      make(map[string]Invitation),
		audits:       make(map[string][]AuditEntry),
	}
}

// SaveOrg 保存机构。
func (m *MemoryStore) SaveOrg(o Org, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orgs[o.DataRegion+"|"+o.OrgID] = o
	if idemKey != "" {
		m.orgIDem[o.DataRegion+"|"+idemKey] = o
	}
	return nil
}

// GetOrgByID 按 ID 查询机构。
func (m *MemoryStore) GetOrgByID(dataRegion, orgID string) (Org, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orgs[dataRegion+"|"+orgID]
	if !ok {
		return Org{}, ErrNotFound
	}
	return o, nil
}

// GetOrgByIdempotencyKey 幂等键查询机构。
func (m *MemoryStore) GetOrgByIdempotencyKey(dataRegion, key string) (Org, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orgIDem[dataRegion+"|"+key]
	if !ok {
		return Org{}, ErrNotFound
	}
	return o, nil
}

// UpdateOrg 更新机构状态。
func (m *MemoryStore) UpdateOrg(o Org) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orgs[o.DataRegion+"|"+o.OrgID] = o
	return nil
}

// SaveMember 保存成员。
func (m *MemoryStore) SaveMember(member OrgMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.members[member.OrgID+"|"+member.UserID] = member
	m.membersByOrg[member.OrgID] = append(m.membersByOrg[member.OrgID], member)
	return nil
}

// GetMember 查询成员。
func (m *MemoryStore) GetMember(orgID, userID string) (OrgMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	member, ok := m.members[orgID+"|"+userID]
	if !ok {
		return OrgMember{}, ErrNotFound
	}
	return member, nil
}

// ListMembers 列出机构成员。
func (m *MemoryStore) ListMembers(orgID string) ([]OrgMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.membersByOrg[orgID]
	out := make([]OrgMember, len(items))
	copy(out, items)
	return out, nil
}

// UpdateMember 更新成员（角色/退出时间）。
func (m *MemoryStore) UpdateMember(member OrgMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.members[member.OrgID+"|"+member.UserID] = member
	return nil
}

// SaveInvitation 保存邀请。
func (m *MemoryStore) SaveInvitation(inv Invitation, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invitations[inv.DataRegion+"|"+inv.InvitationID] = inv
	if idemKey != "" {
		m.invIDem[inv.DataRegion+"|"+idemKey] = inv
	}
	return nil
}

// GetInvitationByID 按邀请 ID 查询。
func (m *MemoryStore) GetInvitationByID(dataRegion, invitationID string) (Invitation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inv, ok := m.invitations[dataRegion+"|"+invitationID]
	if !ok {
		return Invitation{}, ErrNotFound
	}
	return inv, nil
}

// GetInvitationByIdempotencyKey 幂等键查询邀请。
func (m *MemoryStore) GetInvitationByIdempotencyKey(dataRegion, key string) (Invitation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inv, ok := m.invIDem[dataRegion+"|"+key]
	if !ok {
		return Invitation{}, ErrNotFound
	}
	return inv, nil
}

// UpdateInvitation 更新邀请状态。
func (m *MemoryStore) UpdateInvitation(inv Invitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invitations[inv.DataRegion+"|"+inv.InvitationID] = inv
	return nil
}

// ListInvitations 列出机构邀请。
func (m *MemoryStore) ListInvitations(dataRegion, orgID string) ([]Invitation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Invitation, 0)
	for _, inv := range m.invitations {
		if inv.DataRegion == dataRegion && inv.OrgID == orgID {
			out = append(out, inv)
		}
	}
	return out, nil
}

// AppendAudit 追加审计（只增不改）。
func (m *MemoryStore) AppendAudit(a AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits[a.DataRegion+"|"+a.OrgID] = append(m.audits[a.DataRegion+"|"+a.OrgID], a)
	return nil
}

// ListAudits 查询机构审计。
func (m *MemoryStore) ListAudits(dataRegion, orgID string) ([]AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.audits[dataRegion+"|"+orgID]
	out := make([]AuditEntry, len(items))
	copy(out, items)
	return out, nil
}
