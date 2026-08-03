// Package adminapi 提供客服工单：默认最小可见、逐字稿需用户授权、
// 媒体访问需用户申请+双人审批（TASK-085；FR-039，US-08 场景 4；
// SCREEN-SPEC SCR-17 客服工单）。
package adminapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 工单状态。
const (
	TicketOpen     = "open"
	TicketProgress = "in_progress"
	TicketResolved = "resolved"
	TicketClosed   = "closed"
)

// 工单类别（默认最小可见：账户状态/订单/额度/故障代码/用户提交材料）。
const (
	TicketAccount    = "account"
	TicketOrder      = "order"
	TicketEntitle    = "entitlement"
	TicketFault      = "fault"
	TicketTranscript = "transcript"
	TicketMedia      = "media"
	TicketOther      = "other"
)

// 逐字稿授权状态与媒体申请状态。
const (
	AuthActive  = "active"
	AuthExpired = "expired"
	AuthRevoked = "revoked"

	MediaRequested = "requested"
	MediaApproved  = "approved"
	MediaRejected  = "rejected"
)

// Ticket 为客服工单（默认最小可见；内容访问需授权）。
type Ticket struct {
	TicketID   string
	UserID     string
	Subject    string
	Category   string
	Status     string
	Visibility string // minimal | authorized
	CreatedBy  string
	DataRegion string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

// TranscriptAuthorization 为逐字稿会话级授权（用户针对会话授权；范围+有效期）。
type TranscriptAuthorization struct {
	AuthID     string
	TicketID   string
	UserID     string
	SessionID  string
	Status     string
	ExpiresAt  time.Time
	DataRegion string
	GrantedAt  time.Time
}

// MediaAccessRequest 为媒体访问申请（用户申请+双人审批）。
type MediaAccessRequest struct {
	AccessRequestID string
	TicketID        string
	UserID          string
	SessionID       string
	Status          string
	ApproverPair    []string
	DataRegion      string
	RequestedAt     time.Time
	DecidedAt       *time.Time
}

// CreateTicket 创建工单（默认最小可见；客服角色）。
func (s *Service) CreateTicket(
	_ context.Context, actor Actor, userID, subject, category, idemKey string,
) (Ticket, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return Ticket{}, err
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(subject) == "" ||
		strings.TrimSpace(idemKey) == "" {
		return Ticket{}, fmt.Errorf("%w: user_id、subject 与幂等键必填", ErrInvalidInput)
	}
	if !validTicketCategory(category) {
		return Ticket{}, fmt.Errorf("%w: 工单类别非法", ErrInvalidInput)
	}
	if cached, err := s.store.GetTicketByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Ticket{}, err
	}
	ticket := Ticket{
		TicketID:   newID(),
		UserID:     userID,
		Subject:    subject,
		Category:   category,
		Status:     TicketOpen,
		Visibility: "minimal",
		CreatedBy:  actor.StaffID,
		DataRegion: actor.DataRegion,
		CreatedAt:  s.now().UTC(),
	}
	if err := s.store.SaveTicket(ticket, idemKey); err != nil {
		return Ticket{}, err
	}
	_ = s.appendAudit(actor, "ticket.created", ticket.TicketID)
	return ticket, nil
}

// GetTicket 返回工单（默认最小可见：不含逐字稿与媒体内容）。
func (s *Service) GetTicket(
	_ context.Context, actor Actor, ticketID string,
) (Ticket, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return Ticket{}, err
	}
	return s.store.GetTicketByID(actor.DataRegion, ticketID)
}

// ResolveTicket 解决/关闭工单。
func (s *Service) ResolveTicket(
	_ context.Context, actor Actor, ticketID string,
) (Ticket, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return Ticket{}, err
	}
	ticket, err := s.store.GetTicketByID(actor.DataRegion, ticketID)
	if err != nil {
		return Ticket{}, err
	}
	if ticket.Status == TicketResolved || ticket.Status == TicketClosed {
		return Ticket{}, fmt.Errorf("%w: 工单已结束", ErrStateConflict)
	}
	now := s.now().UTC()
	ticket.Status = TicketResolved
	ticket.ResolvedAt = &now
	if err := s.store.UpdateTicket(ticket); err != nil {
		return Ticket{}, err
	}
	_ = s.appendAudit(actor, "ticket.resolved", ticketID)
	return ticket, nil
}

// GrantTranscriptAuthorization 记录用户针对会话的逐字稿授权（范围+有效期）。
func (s *Service) GrantTranscriptAuthorization(
	_ context.Context, actor Actor, userID, ticketID, sessionID string,
	expiresAt time.Time, idemKey string,
) (TranscriptAuthorization, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return TranscriptAuthorization{}, err
	}
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(idemKey) == "" ||
		expiresAt.IsZero() {
		return TranscriptAuthorization{}, fmt.Errorf("%w: session_id、expires_at 与幂等键必填", ErrInvalidInput)
	}
	if !s.now().UTC().Before(expiresAt) {
		return TranscriptAuthorization{}, fmt.Errorf("%w: 有效期必须晚于当前时间", ErrInvalidInput)
	}
	if cached, err := s.store.GetTranscriptAuthByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return TranscriptAuthorization{}, err
	}
	auth := TranscriptAuthorization{
		AuthID:     newID(),
		TicketID:   ticketID,
		UserID:     userID,
		SessionID:  sessionID,
		Status:     AuthActive,
		ExpiresAt:  expiresAt,
		DataRegion: actor.DataRegion,
		GrantedAt:  s.now().UTC(),
	}
	if err := s.store.SaveTranscriptAuthorization(auth, idemKey); err != nil {
		return TranscriptAuthorization{}, err
	}
	_ = s.appendAudit(actor, "ticket.transcript_authorized", ticketID+":"+sessionID)
	return auth, nil
}

// CheckTranscriptAccess 逐字稿访问校验：授权有效且未过期（否则不可见；写审计）。
func (s *Service) CheckTranscriptAccess(
	_ context.Context, actor Actor, ticketID, sessionID string,
) (bool, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return false, err
	}
	auths, err := s.store.ListTranscriptAuths(actor.DataRegion, ticketID, sessionID)
	if err != nil {
		return false, err
	}
	for _, auth := range auths {
		if auth.Status == AuthActive && !s.now().UTC().After(auth.ExpiresAt) {
			_ = s.appendAudit(actor, "ticket.transcript_accessed", ticketID+":"+sessionID)
			return true, nil
		}
	}
	_ = s.appendAudit(actor, "ticket.transcript_blocked", ticketID+":"+sessionID)
	return false, nil
}

// RequestMediaAccess 媒体访问申请（用户申请；双人审批）。
func (s *Service) RequestMediaAccess(
	_ context.Context, actor Actor, userID, ticketID, sessionID, idemKey string,
) (MediaAccessRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return MediaAccessRequest{}, err
	}
	if cached, err := s.store.GetMediaRequestByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return MediaAccessRequest{}, err
	}
	req := MediaAccessRequest{
		AccessRequestID: newID(),
		TicketID:        ticketID,
		UserID:          userID,
		SessionID:       sessionID,
		Status:          MediaRequested,
		DataRegion:      actor.DataRegion,
		RequestedAt:     s.now().UTC(),
	}
	if err := s.store.SaveMediaRequest(req, idemKey); err != nil {
		return MediaAccessRequest{}, err
	}
	_ = s.appendAudit(actor, "ticket.media_requested", ticketID+":"+sessionID)
	return req, nil
}

// ApproveMediaAccess 媒体访问双人审批（两人不同审批人后放行；用户本人不可自批）。
func (s *Service) ApproveMediaAccess(
	_ context.Context, actor Actor, requestID, approverID string,
) (MediaAccessRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return MediaAccessRequest{}, err
	}
	req, err := s.store.GetMediaRequestByID(actor.DataRegion, requestID)
	if err != nil {
		return MediaAccessRequest{}, err
	}
	if req.UserID == approverID {
		return MediaAccessRequest{}, fmt.Errorf("%w: 用户本人不可审批自己的媒体申请", ErrForbidden)
	}
	if req.Status == MediaApproved || req.Status == MediaRejected {
		return req, nil
	}
	pair, err := s.store.AppendMediaApproval(actor.DataRegion, requestID, approverID)
	if err != nil {
		return MediaAccessRequest{}, err
	}
	req.ApproverPair = pair
	if len(pair) == 2 {
		now := s.now().UTC()
		req.Status = MediaApproved
		req.DecidedAt = &now
		if err := s.store.UpdateMediaRequest(req); err != nil {
			return MediaAccessRequest{}, err
		}
		_ = s.appendAudit(actor, "ticket.media_approved", requestID)
	}
	return req, nil
}

// RejectMediaAccess 拒绝媒体访问申请。
func (s *Service) RejectMediaAccess(
	_ context.Context, actor Actor, requestID string,
) (MediaAccessRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return MediaAccessRequest{}, err
	}
	req, err := s.store.GetMediaRequestByID(actor.DataRegion, requestID)
	if err != nil {
		return MediaAccessRequest{}, err
	}
	if req.Status != MediaRequested {
		return MediaAccessRequest{}, fmt.Errorf("%w: 当前状态不可拒绝", ErrStateConflict)
	}
	now := s.now().UTC()
	req.Status = MediaRejected
	req.DecidedAt = &now
	if err := s.store.UpdateMediaRequest(req); err != nil {
		return MediaAccessRequest{}, err
	}
	_ = s.appendAudit(actor, "ticket.media_rejected", requestID)
	return req, nil
}

func validTicketCategory(category string) bool {
	switch category {
	case TicketAccount, TicketOrder, TicketEntitle, TicketFault,
		TicketTranscript, TicketMedia, TicketOther:
		return true
	}
	return false
}
