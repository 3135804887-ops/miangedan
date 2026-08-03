// Package org 提供机构租户内存存储（TASK-070；开发/测试；生产 PostgreSQL）。
package org

import "sync"

// MemoryStore 为内存版机构存储。
type MemoryStore struct {
	mu           sync.RWMutex
	orgs         map[string]Org
	orgIDem      map[string]Org
	members      map[string]Member
	membersByOrg map[string][]Member
	invitations  map[string]Invitation
	invIDem      map[string]Invitation
	audits       map[string][]AuditEntry
	assignments  map[string]Assignment
	assignIDem   map[string]Assignment
	assignByOrg  map[string][]Assignment
	assignMember map[string]AssignmentMember
	assignMemBy  map[string][]AssignmentMember
	shares       map[string]Share
	shareIDem    map[string]Share
	sharesByUser map[string][]Share
	sharesByAsg  map[string][]Share
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		orgs:         make(map[string]Org),
		orgIDem:      make(map[string]Org),
		members:      make(map[string]Member),
		membersByOrg: make(map[string][]Member),
		invitations:  make(map[string]Invitation),
		invIDem:      make(map[string]Invitation),
		audits:       make(map[string][]AuditEntry),
		assignments:  make(map[string]Assignment),
		assignIDem:   make(map[string]Assignment),
		assignByOrg:  make(map[string][]Assignment),
		assignMember: make(map[string]AssignmentMember),
		assignMemBy:  make(map[string][]AssignmentMember),
		shares:       make(map[string]Share),
		shareIDem:    make(map[string]Share),
		sharesByUser: make(map[string][]Share),
		sharesByAsg:  make(map[string][]Share),
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
func (m *MemoryStore) SaveMember(member Member) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.members[member.OrgID+"|"+member.UserID] = member
	m.membersByOrg[member.OrgID] = append(m.membersByOrg[member.OrgID], member)
	return nil
}

// GetMember 查询成员。
func (m *MemoryStore) GetMember(orgID, userID string) (Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	member, ok := m.members[orgID+"|"+userID]
	if !ok {
		return Member{}, ErrNotFound
	}
	return member, nil
}

// ListMembers 列出机构成员。
func (m *MemoryStore) ListMembers(orgID string) ([]Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.membersByOrg[orgID]
	out := make([]Member, len(items))
	copy(out, items)
	return out, nil
}

// UpdateMember 更新成员（角色/退出时间）。
func (m *MemoryStore) UpdateMember(member Member) error {
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

// SaveAssignment 保存训练任务。
func (m *MemoryStore) SaveAssignment(a Assignment, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assignments[a.DataRegion+"|"+a.AssignmentID] = a
	if idemKey != "" {
		m.assignIDem[a.DataRegion+"|"+idemKey] = a
	}
	m.assignByOrg[a.DataRegion+"|"+a.OrgID] = append(m.assignByOrg[a.DataRegion+"|"+a.OrgID], a)
	return nil
}

// GetAssignmentByID 按 ID 查询任务。
func (m *MemoryStore) GetAssignmentByID(dataRegion, assignmentID string) (Assignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.assignments[dataRegion+"|"+assignmentID]
	if !ok {
		return Assignment{}, ErrNotFound
	}
	return a, nil
}

// GetAssignmentByIdempotencyKey 幂等键查询任务。
func (m *MemoryStore) GetAssignmentByIdempotencyKey(dataRegion, key string) (Assignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.assignIDem[dataRegion+"|"+key]
	if !ok {
		return Assignment{}, ErrNotFound
	}
	return a, nil
}

// UpdateAssignment 更新任务状态。
func (m *MemoryStore) UpdateAssignment(a Assignment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assignments[a.DataRegion+"|"+a.AssignmentID] = a
	return nil
}

// ListAssignments 列出机构任务。
func (m *MemoryStore) ListAssignments(dataRegion, orgID string) ([]Assignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.assignByOrg[dataRegion+"|"+orgID]
	out := make([]Assignment, len(items))
	copy(out, items)
	return out, nil
}

// SaveAssignmentMember 保存任务-成员状态。
func (m *MemoryStore) SaveAssignmentMember(member AssignmentMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assignMember[member.AssignmentID+"|"+member.UserID] = member
	m.assignMemBy[member.AssignmentID] = append(m.assignMemBy[member.AssignmentID], member)
	return nil
}

// GetAssignmentMember 查询任务-成员状态。
func (m *MemoryStore) GetAssignmentMember(assignmentID, userID string) (AssignmentMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	member, ok := m.assignMember[assignmentID+"|"+userID]
	if !ok {
		return AssignmentMember{}, ErrNotFound
	}
	return member, nil
}

// UpdateAssignmentMember 更新任务-成员状态。
func (m *MemoryStore) UpdateAssignmentMember(member AssignmentMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assignMember[member.AssignmentID+"|"+member.UserID] = member
	return nil
}

// ListAssignmentMembers 列出任务成员。
func (m *MemoryStore) ListAssignmentMembers(assignmentID string) ([]AssignmentMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.assignMemBy[assignmentID]
	out := make([]AssignmentMember, len(items))
	copy(out, items)
	return out, nil
}

// SaveShare 保存分享授权。
func (m *MemoryStore) SaveShare(sh Share, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shares[sh.DataRegion+"|"+sh.ShareID] = sh
	if idemKey != "" {
		m.shareIDem[sh.DataRegion+"|"+idemKey] = sh
	}
	m.sharesByUser[sh.DataRegion+"|"+sh.UserID+"|"+sh.AssignmentID] =
		append(m.sharesByUser[sh.DataRegion+"|"+sh.UserID+"|"+sh.AssignmentID], sh)
	m.sharesByAsg[sh.DataRegion+"|"+sh.AssignmentID] =
		append(m.sharesByAsg[sh.DataRegion+"|"+sh.AssignmentID], sh)
	return nil
}

// GetShareByID 按分享 ID 查询。
func (m *MemoryStore) GetShareByID(dataRegion, shareID string) (Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sh, ok := m.shares[dataRegion+"|"+shareID]
	if !ok {
		return Share{}, ErrNotFound
	}
	return sh, nil
}

// GetShareByIdempotencyKey 幂等键查询分享。
func (m *MemoryStore) GetShareByIdempotencyKey(dataRegion, key string) (Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sh, ok := m.shareIDem[dataRegion+"|"+key]
	if !ok {
		return Share{}, ErrNotFound
	}
	return sh, nil
}

// UpdateShare 更新分享状态（撤回/到期）。
func (m *MemoryStore) UpdateShare(sh Share) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shares[sh.DataRegion+"|"+sh.ShareID] = sh
	return nil
}

// ListSharesByUser 列出用户在任务上的分享。
func (m *MemoryStore) ListSharesByUser(dataRegion, userID, assignmentID string) ([]Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.sharesByUser[dataRegion+"|"+userID+"|"+assignmentID]
	out := make([]Share, len(items))
	copy(out, items)
	return out, nil
}

// ListSharesByAssignment 列出任务的分享。
func (m *MemoryStore) ListSharesByAssignment(dataRegion, assignmentID string) ([]Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.sharesByAsg[dataRegion+"|"+assignmentID]
	out := make([]Share, len(items))
	copy(out, items)
	return out, nil
}

// ListActiveShares 列出区域全部有效分享（到期扫描）。
func (m *MemoryStore) ListActiveShares(dataRegion string) ([]Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Share, 0)
	for _, sh := range m.shares {
		if sh.DataRegion == dataRegion && sh.Status == ShareActive {
			out = append(out, sh)
		}
	}
	return out, nil
}
