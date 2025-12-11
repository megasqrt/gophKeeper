package grpchelper

import (
	"fmt"
	pb "gophKeeper/internal/proto/gen"
)

// --- TextData Converters ---

// ToProto converts a domain TextData model to a Protobuf TextData model.
func (t *TextData) ToProto() *pb.NoteItem {
	return pb.NoteItem_builder{
		LocalId:  &t.LocalID,
		ServerId: &t.ServerID,
		Title:    &t.Title,
		Text:     &t.Text,
		Checksum: &t.Checksum,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &t.ChangeTime,
			SyncTime:   &t.SyncTime,
		}.Build(),
		Deleted: &t.Deleted,
	}.Build()
}

// FromProtoText converts a Protobuf NoteItem to a domain TextData model.
func FromProtoText(pbText *pb.NoteItem) TextData {
	return TextData{
		LocalID:    pbText.GetLocalId(),
		ServerID:   pbText.GetServerId(),
		Title:      pbText.GetTitle(),
		Text:       pbText.GetText(),
		Checksum:   pbText.GetChecksum(),
		ChangeTime: pbText.GetTimemap().GetChangeTime(),
		SyncTime:   pbText.GetTimemap().GetSyncTime(),
		Deleted:    pbText.GetDeleted(),
	}
}

// FromMapText converts a map to a domain TextData model.
func FromMapText(data map[string]interface{}) (TextData, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	return TextData{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Title:      InterfaceToString(data["title"]),
		Text:       InterfaceToString(data["text"]),
		Checksum:   InterfaceToString(data["checksum"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}

// --- Card Converters ---

// ToProto converts a domain Card model to a Protobuf CardData model.
// Note: Checksum is not included in proto as CardItem doesn't have this field yet.
func (c *Card) ToProto() *pb.CardItem {
	return pb.CardItem_builder{
		LocalId:  &c.LocalID,
		ServerId: &c.ServerID,
		Number:   &c.Number,
		Holder:   &c.Holder,
		Expiry:   &c.Expiry,
		Cvv:      &c.CVV,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &c.ChangeTime,
			SyncTime:   &c.SyncTime,
		}.Build(),
		Deleted: &c.Deleted,
	}.Build()
}

func (c *SyncInfo) ToProto() *pb.ShortItem {
	return pb.ShortItem_builder{
		LocalId:  &c.LocalID,
		ServerId: &c.ServerID,
		Checksum: &c.Checksum,
		Deleted:  &c.Deleted,
	}.Build()
}

// FromProtoCard converts a Protobuf CardItem to a domain Card model.
// Note: Checksum will be recalculated when saving locally.
func FromProtoCard(pbCard *pb.CardItem) Card {
	card := Card{
		LocalID:    pbCard.GetLocalId(),
		ServerID:   pbCard.GetServerId(),
		Number:     pbCard.GetNumber(),
		Holder:     pbCard.GetHolder(),
		Expiry:     pbCard.GetExpiry(),
		CVV:        pbCard.GetCvv(),
		ChangeTime: pbCard.GetTimemap().GetChangeTime(),
		SyncTime:   pbCard.GetTimemap().GetSyncTime(),
		Deleted:    pbCard.GetDeleted(),
	}
	// Checksum будет пересчитан при сохранении в локальное хранилище
	return card
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
		"checksum":   c.Checksum,
		"changeTime": c.ChangeTime,
		"syncTime":   c.SyncTime,
		"deleted":    c.Deleted,
	}
}

// FromMapCard converts a map to a domain Card model.
func FromMapCard(data map[string]interface{}) (Card, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	return Card{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Number:     InterfaceToString(data["number"]),
		Holder:     InterfaceToString(data["holder"]),
		Expiry:     InterfaceToString(data["expiry"]),
		CVV:        InterfaceToString(data["cvv"]),
		Checksum:   InterfaceToString(data["checksum"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}

// --- Password Converters ---

// ToProto converts a domain Password model to a Protobuf PasswordData model.
// Note: Checksum is not included in proto as PasswordItem may not have this field yet.
func (p *Password) ToProto() *pb.PasswordItem {
	return pb.PasswordItem_builder{
		LocalId:     &p.LocalID,
		ServerId:    &p.ServerID,
		Login:       &p.Login,
		Password:    &p.Password,
		Description: &p.Description,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &p.ChangeTime,
			SyncTime:   &p.SyncTime,
		}.Build(),
		Deleted: &p.Deleted,
	}.Build()
}

// FromProtoPassword converts a Protobuf PasswordItem to a domain Password model.
// Note: Checksum will be recalculated when saving locally.
func FromProtoPassword(pbPass *pb.PasswordItem) Password {
	pass := Password{
		LocalID:     pbPass.GetLocalId(),
		ServerID:    pbPass.GetServerId(),
		Login:       pbPass.GetLogin(),
		Password:    pbPass.GetPassword(),
		Description: pbPass.GetDescription(),
		ChangeTime:  pbPass.GetTimemap().GetChangeTime(),
		SyncTime:    pbPass.GetTimemap().GetSyncTime(),
		Deleted:     pbPass.GetDeleted(),
	}
	// Checksum будет пересчитан при сохранении в локальное хранилище
	return pass
}

// ToMap converts a Password model to a map for storage.
func (p *Password) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":          p.LocalID,
		"server_id":   p.ServerID,
		"login":       p.Login,
		"password":    p.Password,
		"description": p.Description,
		"checksum":    p.Checksum,
		"changeTime":  p.ChangeTime,
		"syncTime":    p.SyncTime,
		"deleted":     p.Deleted,
	}
}

// FromMapPassword converts a map to a domain Password model.
func FromMapPassword(data map[string]interface{}) (Password, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	return Password{
		LocalID:     InterfaceToString(data["id"]),
		ServerID:    InterfaceToString(data["server_id"]),
		Login:       InterfaceToString(data["login"]),
		Password:    InterfaceToString(data["password"]),
		Description: InterfaceToString(data["description"]),
		Checksum:    InterfaceToString(data["checksum"]),
		ChangeTime:  changeTime,
		SyncTime:    syncTime,
		Deleted:     deleted,
	}, nil
}

// --- FileData Converters ---

// ToProto converts a domain FileData model to a Protobuf FileData model.
// Note: Checksum is not included in proto as FileItem may not have this field yet.
func (f *FileData) ToProto() *pb.FileItem {
	return pb.FileItem_builder{
		LocalId:  &f.LocalID,
		ServerId: &f.ServerID,
		Name:     &f.Name,
		Size:     &f.Size,
		Metadata: &f.Metadata,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &f.ChangeTime,
			SyncTime:   &f.SyncTime,
		}.Build(),
		Deleted: &f.Deleted,
	}.Build()
}

// FromProtoFile converts a Protobuf FileItem to a domain FileData model.
// Note: Checksum will be recalculated when saving locally.
func FromProtoFile(pbFile *pb.FileItem) FileData {
	file := FileData{
		LocalID:    pbFile.GetLocalId(),
		ServerID:   pbFile.GetServerId(),
		Name:       pbFile.GetName(),
		Size:       pbFile.GetSize(),
		Metadata:   pbFile.GetMetadata(),
		ChangeTime: pbFile.GetTimemap().GetChangeTime(),
		SyncTime:   pbFile.GetTimemap().GetSyncTime(),
		Deleted:    pbFile.GetDeleted(),
	}
	// Checksum будет пересчитан при сохранении в локальное хранилище
	return file
}

// ToMap converts a FileData model to a map for storage.
// Обратите внимание, что он возвращает map[string]interface{} из-за поля Size (int64).
func (f *FileData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         f.LocalID,
		"server_id":  f.ServerID,
		"name":       f.Name,
		"metadata":   f.Metadata,
		"size":       f.Size,
		"checksum":   f.Checksum,
		"changeTime": f.ChangeTime,
		"syncTime":   f.SyncTime,
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
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	return FileData{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Name:       InterfaceToString(data["name"]),
		Size:       size,
		Metadata:   InterfaceToString(data["metadata"]),
		Checksum:   InterfaceToString(data["checksum"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}
