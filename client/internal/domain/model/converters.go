package model

import (
	"fmt"
	pb "gophKeeper/internal/proto/gen"
	"time"
)

// toProtoTimemap является вспомогательной функцией для конвертации time.Time в *pb.TimeMap.
func toProtoTimemap(changeTime, syncTime time.Time) *pb.TimeMap {
	return pb.TimeMap_builder{
		ChangeTime: changeTime.UnixNano(),
		SyncTime:   syncTime.UnixNano(),
	}.Build()
}

// parseChangeTime извлекает и парсит время изменения из map.
func parseChangeTime(data map[string]string) time.Time {
	if changeTimeStr, ok := data["changeTime"]; ok {
		t, _ := time.Parse(time.RFC3339Nano, changeTimeStr)
		return t
	}
	return time.Time{}
}

// --- TextData Converters ---

// ToProto converts a domain TextData model to a Protobuf TextData model.
func (t *TextData) ToProto() *pb.NoteItem {
	return pb.NoteItem_builder{
		LocalId:  &t.LocalID,
		ServerId: &t.ServerID,
		Title:    &t.Title,
		Text:     &t.Text,
		Timemap:  toProtoTimemap(t.ChangeTime, t.SyncTime),
		Deleted:  &t.Deleted,
	}.Build()
}

func GetModelText(data map[string]string) TextData {
	return TextData{
		LocalID:    data["id"],
		ServerID:   data["server_id"],
		Title:      data["title"],
		Text:       data["text"],
		Deleted:    data["deleted"] == "true",
		ChangeTime: parseChangeTime(data),
	}
}

// FromProtoText converts a Protobuf NoteItem to a domain TextData model.
func FromProtoText(pbText *pb.NoteItem) TextData {
	return TextData{
		LocalID:    pbText.GetLocalId(),
		ServerID:   pbText.GetServerId(),
		Title:      pbText.GetTitle(),
		Text:       pbText.GetText(),
		ChangeTime: time.Unix(0, pbText.GetTimemap().GetChangeTime()),
		SyncTime:   time.Unix(0, pbText.GetTimemap().GetSyncTime()),
		Deleted:    pbText.GetDeleted(),
	}
}

// FromMapText converts a map to a domain TextData model.
func FromMapText(data map[string]interface{}) (TextData, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToTime(data["changeTime"])
	syncTime, _ := InterfaceToTime(data["syncTime"])

	return TextData{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Title:      InterfaceToString(data["title"]),
		Text:       InterfaceToString(data["text"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}

// --- Card Converters ---

// ToProto converts a domain Card model to a Protobuf CardData model.
func (c *Card) ToProto() *pb.CardItem {
	return pb.CardItem_builder{
		LocalId:  &c.LocalID,
		ServerId: &c.ServerID,
		Number:   &c.Number,
		Holder:   &c.Holder,
		Expiry:   &c.Expiry,
		Cvv:      &c.CVV,
		Timemap:  toProtoTimemap(c.ChangeTime, c.SyncTime),
		Deleted:  &c.Deleted,
	}.Build()
}

// FromProtoCard converts a Protobuf CardItem to a domain Card model.
func FromProtoCard(pbCard *pb.CardItem) Card {
	return Card{
		LocalID:    pbCard.GetLocalId(),
		ServerID:   pbCard.GetServerId(),
		Number:     pbCard.GetNumber(),
		Holder:     pbCard.GetHolder(),
		Expiry:     pbCard.GetExpiry(),
		CVV:        pbCard.GetCvv(),
		ChangeTime: time.Unix(0, pbCard.GetTimemap().GetChangeTime()),
		SyncTime:   time.Unix(0, pbCard.GetTimemap().GetSyncTime()),
		Deleted:    pbCard.GetDeleted(),
	}
}

// ToMap converts a Card model to a map for storage.
func (c *Card) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         c.LocalID,
		"server_id":  c.ServerID,
		"number":     c.Number,
		"holder":     c.Holder,
		"expiry":     c.Expiry,
		"cvv":        c.CVV,
		"changeTime": c.ChangeTime.Format(time.RFC3339Nano),
		"syncTime":   c.SyncTime.Format(time.RFC3339Nano),
		"deleted":    c.Deleted,
	}
}

// FromMapCard converts a map to a domain Card model.
func FromMapCard(data map[string]interface{}) (Card, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToTime(data["changeTime"])
	syncTime, _ := InterfaceToTime(data["syncTime"])

	return Card{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Number:     InterfaceToString(data["number"]),
		Holder:     InterfaceToString(data["holder"]),
		Expiry:     InterfaceToString(data["expiry"]),
		CVV:        InterfaceToString(data["cvv"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}

// --- Password Converters ---

// ToProto converts a domain Password model to a Protobuf PasswordData model.
func (p *Password) ToProto() *pb.PasswordItem {
	return pb.PasswordItem_builder{
		LocalId:     &p.LocalID,
		ServerId:    &p.ServerID,
		Login:       &p.Login,
		Password:    &p.Password,
		Description: &p.Description,
		Timemap:     toProtoTimemap(p.ChangeTime, p.SyncTime),
		Deleted:     &p.Deleted,
	}.Build()
}

// FromProtoPassword converts a Protobuf PasswordItem to a domain Password model.
func FromProtoPassword(pbPass *pb.PasswordItem) Password {
	return Password{
		LocalID:     pbPass.GetLocalId(),
		ServerID:    pbPass.GetServerId(),
		Login:       pbPass.GetLogin(),
		Password:    pbPass.GetPassword(),
		Description: pbPass.GetDescription(),
		ChangeTime:  time.Unix(0, pbPass.GetTimemap().GetChangeTime()),
		SyncTime:    time.Unix(0, pbPass.GetTimemap().GetSyncTime()),
		Deleted:     pbPass.GetDeleted(),
	}
}

// ToMap converts a Password model to a map for storage.
func (p *Password) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":          p.LocalID,
		"server_id":   p.ServerID,
		"login":       p.Login,
		"password":    p.Password,
		"description": p.Description,
		"changeTime":  p.ChangeTime.Format(time.RFC3339Nano),
		"syncTime":    p.SyncTime.Format(time.RFC3339Nano),
		"deleted":     p.Deleted,
	}
}

// FromMapPassword converts a map to a domain Password model.
func FromMapPassword(data map[string]interface{}) (Password, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToTime(data["changeTime"])
	syncTime, _ := InterfaceToTime(data["syncTime"])

	return Password{
		LocalID:     InterfaceToString(data["id"]),
		ServerID:    InterfaceToString(data["server_id"]),
		Login:       InterfaceToString(data["login"]),
		Password:    InterfaceToString(data["password"]),
		Description: InterfaceToString(data["description"]),
		ChangeTime:  changeTime,
		SyncTime:    syncTime,
		Deleted:     deleted,
	}, nil
}

// --- FileData Converters ---

// ToProto converts a domain FileData model to a Protobuf FileData model.
func (f *FileData) ToProto() *pb.FileItem {
	size := f.Size
	return pb.FileItem_builder{
		LocalId:  &f.LocalID,
		ServerId: &f.ServerID,
		Name:     &f.Name,
		Size:     &size,
		Timemap:  toProtoTimemap(f.ChangeTime, f.SyncTime),
		Deleted:  &f.Deleted,
	}.Build()
}

// FromProtoFile converts a Protobuf FileItem to a domain FileData model.
func FromProtoFile(pbFile *pb.FileItem) FileData {
	return FileData{
		LocalID:    pbFile.GetLocalId(),
		ServerID:   pbFile.GetServerId(),
		Name:       pbFile.GetName(),
		Size:       pbFile.GetSize(),
		ChangeTime: time.Unix(0, pbFile.GetTimemap().GetChangeTime()),
		SyncTime:   time.Unix(0, pbFile.GetTimemap().GetSyncTime()),
		Deleted:    pbFile.GetDeleted(),
	}
}

// ToMap converts a FileData model to a map for storage.
// Обратите внимание, что он возвращает map[string]interface{} из-за поля Size (int64).
func (f *FileData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         f.LocalID,
		"server_id":  f.ServerID,
		"name":       f.Name,
		"size":       f.Size,
		"changeTime": f.ChangeTime.Format(time.RFC3339Nano),
		"syncTime":   f.SyncTime.Format(time.RFC3339Nano),
		"deleted":    f.Deleted,
	}
}

// FromMapFile converts a map to a domain FileData model.
func FromMapFile(data map[string]interface{}) (FileData, error) {
	size, err := InterfaceToInt64(data["size"])
	if err != nil {
		return FileData{}, fmt.Errorf("failed to convert size for file: %w", err)
	}
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToTime(data["changeTime"])
	syncTime, _ := InterfaceToTime(data["syncTime"])

	return FileData{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Name:       InterfaceToString(data["name"]),
		Size:       size,
		Metadata:   InterfaceToString(data["metadata"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}
