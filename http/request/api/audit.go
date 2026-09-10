package api

import (
	"encoding/json"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"strconv"
)

type AuditConnForm struct {
	Action    string   `json:"action"`
	ConnId    int64    `json:"conn_id"`
	Id        string   `json:"id"`
	Peer      []string `json:"peer"`
	Ip        string   `json:"ip"`
	SessionId uint64   `json:"session_id"`
	Type      int      `json:"type"`
	Uuid      string   `json:"uuid"`
}

func (a *AuditConnForm) ToAuditConn() *model.AuditConn {
	fp := ""
	fn := ""
	if len(a.Peer) >= 1 {
		fp = a.Peer[0]
		if len(a.Peer) == 2 {
			fn = a.Peer[1]
		}
	}
	ssid := strconv.FormatUint(a.SessionId, 10)
	return &model.AuditConn{
		Action:    a.Action,
		ConnId:    a.ConnId,
		PeerId:    a.Id,
		FromPeer:  fp,
		FromName:  fn,
		Ip:        a.Ip,
		SessionId: ssid,
		Type:      a.Type,
		Uuid:      a.Uuid,
	}
}

type AuditFileForm struct {
	Id     string `json:"id"`
	Info   string `json:"info"`
	IsFile bool   `json:"is_file"`
	Path   string `json:"path"`
	PeerId string `json:"peer_id"`
	Type   int    `json:"type"`
	Uuid   string `json:"uuid"`
}
type AuditFileInfo struct {
	Ip   string `json:"ip"`
	Name string `json:"name"`
	Num  int    `json:"num"`
}

type AuditAlarmForm struct {
	Id           string `json:"id" binding:"required,max=128"`
	Uuid         string `json:"uuid" binding:"required,max=256"`
	Type         int    `json:"typ" binding:"oneof=0 1 2 6 7 8 9"`
	Info         string `json:"info" binding:"required,max=3072"`
	ConnId       int64  `json:"conn_id" binding:"gte=0"`
	ConnAuditRef string `json:"conn_audit_ref" binding:"max=256"`
}

func (a *AuditAlarmForm) ToAuditAlarm() *model.AuditAlarm {
	return &model.AuditAlarm{
		PeerId:       a.Id,
		Uuid:         a.Uuid,
		Type:         a.Type,
		Info:         a.Info,
		ConnId:       a.ConnId,
		ConnAuditRef: a.ConnAuditRef,
	}
}

func (a *AuditFileForm) ToAuditFile() *model.AuditFile {
	fi := &AuditFileInfo{}
	err := json.Unmarshal([]byte(a.Info), fi)
	if err != nil {
		global.Logger.Warn("ToAuditFile", err)
	}

	return &model.AuditFile{
		PeerId:   a.Id,
		Info:     a.Info,
		IsFile:   a.IsFile,
		FromPeer: a.PeerId,
		Path:     a.Path,
		Type:     a.Type,
		Uuid:     a.Uuid,
		FromName: fi.Name,
		Ip:       fi.Ip,
		Num:      fi.Num,
	}
}
