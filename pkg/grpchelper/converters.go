package grpchelper

import (
	"fmt"
	pb "gophKeeper/pkg/proto"
)

// --- TextData Converters ---

// ToProto converts a domain TextData model to a Protobuf TextData model.
// Шифрует чувствительные данные (Title, Text) перед отправкой, если установлен Encryptor.
func (t *TextData) ToProto() *pb.NoteItem {
	title := t.Title
	text := t.Text

	// Шифруем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if encryptedTitle, err := encryptor.EncryptString(title); err == nil {
			title = encryptedTitle
		}
		if encryptedText, err := encryptor.EncryptString(text); err == nil {
			text = encryptedText
		}
	}

	return pb.NoteItem_builder{
		LocalId:  &t.LocalID,
		ServerId: &t.ServerID,
		Title:    &title,
		Text:     &text,
		Checksum: &t.Checksum,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &t.ChangeTime,
			SyncTime:   &t.SyncTime,
		}.Build(),
		Deleted: &t.Deleted,
	}.Build()
}

// FromProtoText converts a Protobuf NoteItem to a domain TextData model.
// Расшифровывает чувствительные данные (Title, Text) после получения, если установлен Encryptor.
func FromProtoText(pbText *pb.NoteItem) TextData {
	title := pbText.GetTitle()
	text := pbText.GetText()

	// Расшифровываем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedTitle, err := encryptor.DecryptString(title); err == nil {
			title = decryptedTitle
		}
		if decryptedText, err := encryptor.DecryptString(text); err == nil {
			text = decryptedText
		}
	}

	return TextData{
		LocalID:    pbText.GetLocalId(),
		ServerID:   pbText.GetServerId(),
		Title:      title,
		Text:       text,
		Checksum:   pbText.GetChecksum(),
		ChangeTime: pbText.GetTimemap().GetChangeTime(),
		SyncTime:   pbText.GetTimemap().GetSyncTime(),
		Deleted:    pbText.GetDeleted(),
	}
}

// FromMapText converts a map to a domain TextData model.
// Расшифровывает чувствительные данные (Title, Text) после получения, если установлен Encryptor.
func FromMapText(data map[string]interface{}) (TextData, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	title := InterfaceToString(data["title"])
	text := InterfaceToString(data["text"])

	// Расшифровываем данные, если установлен Encryptor
	// Это необходимо для данных, которые могли быть сохранены в зашифрованном виде
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedTitle, err := encryptor.DecryptString(title); err == nil {
			title = decryptedTitle
		}
		if decryptedText, err := encryptor.DecryptString(text); err == nil {
			text = decryptedText
		}
	}

	return TextData{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Title:      title,
		Text:       text,
		Checksum:   InterfaceToString(data["checksum"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}

// --- Card Converters ---

// ToProto converts a domain Card model to a Protobuf CardData model.
// Шифрует чувствительные данные перед отправкой, если установлен Encryptor.
func (c *Card) ToProto() *pb.CardItem {
	number := c.Number
	holder := c.Holder
	expiry := c.Expiry
	cvv := c.CVV
	metadata := c.Metadata

	// Шифруем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if encryptedNumber, err := encryptor.EncryptString(number); err == nil {
			number = encryptedNumber
		}
		if encryptedHolder, err := encryptor.EncryptString(holder); err == nil {
			holder = encryptedHolder
		}
		if encryptedExpiry, err := encryptor.EncryptString(expiry); err == nil {
			expiry = encryptedExpiry
		}
		if encryptedCvv, err := encryptor.EncryptString(cvv); err == nil {
			cvv = encryptedCvv
		}
		if encryptedMetadata, err := encryptor.EncryptString(metadata); err == nil {
			metadata = encryptedMetadata
		}
	}

	return pb.CardItem_builder{
		LocalId:  &c.LocalID,
		ServerId: &c.ServerID,
		Number:   &number,
		Holder:   &holder,
		Expiry:   &expiry,
		Cvv:      &cvv,
		Metadata: &metadata,
		Checksum: &c.Checksum,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &c.ChangeTime,
			SyncTime:   &c.SyncTime,
		}.Build(),
		Deleted: &c.Deleted,
	}.Build()
}

func (c *SyncInfo) ToProto() *pb.ShortItem {
	opType := string(c.OperationType)
	return pb.ShortItem_builder{
		LocalId:  &c.LocalID,
		ServerId: &c.ServerID,
		Checksum: &c.Checksum,
		Type:     &opType,
	}.Build()
}

// FromProtoCard converts a Protobuf CardItem to a domain Card model.
// Расшифровывает чувствительные данные после получения, если установлен Encryptor.
func FromProtoCard(pbCard *pb.CardItem) Card {
	number := pbCard.GetNumber()
	holder := pbCard.GetHolder()
	expiry := pbCard.GetExpiry()
	cvv := pbCard.GetCvv()
	metadata := pbCard.GetMetadata()

	// Расшифровываем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedNumber, err := encryptor.DecryptString(number); err == nil {
			number = decryptedNumber
		}
		if decryptedHolder, err := encryptor.DecryptString(holder); err == nil {
			holder = decryptedHolder
		}
		if decryptedExpiry, err := encryptor.DecryptString(expiry); err == nil {
			expiry = decryptedExpiry
		}
		if decryptedCvv, err := encryptor.DecryptString(cvv); err == nil {
			cvv = decryptedCvv
		}
		if decryptedMetadata, err := encryptor.DecryptString(metadata); err == nil {
			metadata = decryptedMetadata
		}
	}

	return Card{
		LocalID:    pbCard.GetLocalId(),
		ServerID:   pbCard.GetServerId(),
		Number:     number,
		Holder:     holder,
		Expiry:     expiry,
		CVV:        cvv,
		Metadata:   metadata,
		Checksum:   pbCard.GetChecksum(),
		ChangeTime: pbCard.GetTimemap().GetChangeTime(),
		SyncTime:   pbCard.GetTimemap().GetSyncTime(),
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
		"metadata":   c.Metadata,
		"checksum":   c.Checksum,
		"changeTime": c.ChangeTime,
		"syncTime":   c.SyncTime,
		"deleted":    c.Deleted,
	}
}

// FromMapCard converts a map to a domain Card model.
// Расшифровывает чувствительные данные после получения, если установлен Encryptor.
func FromMapCard(data map[string]interface{}) (Card, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	number := InterfaceToString(data["number"])
	holder := InterfaceToString(data["holder"])
	expiry := InterfaceToString(data["expiry"])
	cvv := InterfaceToString(data["cvv"])
	metadata := InterfaceToString(data["metadata"])

	// Расшифровываем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedNumber, err := encryptor.DecryptString(number); err == nil {
			number = decryptedNumber
		}
		if decryptedHolder, err := encryptor.DecryptString(holder); err == nil {
			holder = decryptedHolder
		}
		if decryptedExpiry, err := encryptor.DecryptString(expiry); err == nil {
			expiry = decryptedExpiry
		}
		if decryptedCvv, err := encryptor.DecryptString(cvv); err == nil {
			cvv = decryptedCvv
		}
		if decryptedMetadata, err := encryptor.DecryptString(metadata); err == nil {
			metadata = decryptedMetadata
		}
	}

	return Card{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Number:     number,
		Holder:     holder,
		Expiry:     expiry,
		CVV:        cvv,
		Metadata:   metadata,
		Checksum:   InterfaceToString(data["checksum"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}

// --- Password Converters ---

// ToProto converts a domain Password model to a Protobuf PasswordData model.
// Шифрует чувствительные данные перед отправкой, если установлен Encryptor.
func (p *Password) ToProto() *pb.PasswordItem {
	login := p.Login
	password := p.Password
	description := p.Description

	// Шифруем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if encryptedLogin, err := encryptor.EncryptString(login); err == nil {
			login = encryptedLogin
		}
		if encryptedPassword, err := encryptor.EncryptString(password); err == nil {
			password = encryptedPassword
		}
		if encryptedDescription, err := encryptor.EncryptString(description); err == nil {
			description = encryptedDescription
		}
	}

	return pb.PasswordItem_builder{
		LocalId:     &p.LocalID,
		ServerId:    &p.ServerID,
		Login:       &login,
		Password:    &password,
		Description: &description,
		Checksum:    &p.Checksum,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &p.ChangeTime,
			SyncTime:   &p.SyncTime,
		}.Build(),
		Deleted: &p.Deleted,
	}.Build()
}

// FromProtoPassword converts a Protobuf PasswordItem to a domain Password model.
// Расшифровывает чувствительные данные после получения, если установлен Encryptor.
func FromProtoPassword(pbPass *pb.PasswordItem) Password {
	login := pbPass.GetLogin()
	password := pbPass.GetPassword()
	description := pbPass.GetDescription()

	// Расшифровываем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedLogin, err := encryptor.DecryptString(login); err == nil {
			login = decryptedLogin
		}
		if decryptedPassword, err := encryptor.DecryptString(password); err == nil {
			password = decryptedPassword
		}
		if decryptedDescription, err := encryptor.DecryptString(description); err == nil {
			description = decryptedDescription
		}
	}

	return Password{
		LocalID:     pbPass.GetLocalId(),
		ServerID:    pbPass.GetServerId(),
		Login:       login,
		Password:    password,
		Description: description,
		Checksum:    pbPass.GetChecksum(),
		ChangeTime:  pbPass.GetTimemap().GetChangeTime(),
		SyncTime:    pbPass.GetTimemap().GetSyncTime(),
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
		"checksum":    p.Checksum,
		"changeTime":  p.ChangeTime,
		"syncTime":    p.SyncTime,
		"deleted":     p.Deleted,
	}
}

// FromMapPassword converts a map to a domain Password model.
// Расшифровывает чувствительные данные после получения, если установлен Encryptor.
func FromMapPassword(data map[string]interface{}) (Password, error) {
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	login := InterfaceToString(data["login"])
	password := InterfaceToString(data["password"])
	description := InterfaceToString(data["description"])

	// Расшифровываем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedLogin, err := encryptor.DecryptString(login); err == nil {
			login = decryptedLogin
		}
		if decryptedPassword, err := encryptor.DecryptString(password); err == nil {
			password = decryptedPassword
		}
		if decryptedDescription, err := encryptor.DecryptString(description); err == nil {
			description = decryptedDescription
		}
	}

	return Password{
		LocalID:     InterfaceToString(data["id"]),
		ServerID:    InterfaceToString(data["server_id"]),
		Login:       login,
		Password:    password,
		Description: description,
		Checksum:    InterfaceToString(data["checksum"]),
		ChangeTime:  changeTime,
		SyncTime:    syncTime,
		Deleted:     deleted,
	}, nil
}

// --- FileData Converters ---

// ToProto converts a domain FileData model to a Protobuf FileData model.
// ToProto converts a domain FileData model to a Protobuf FileData model.
// Шифрует чувствительные данные перед отправкой, если установлен Encryptor.
func (f *FileData) ToProto() *pb.FileItem {
	name := f.Name
	metadata := f.Metadata

	// Шифруем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if encryptedName, err := encryptor.EncryptString(name); err == nil {
			name = encryptedName
		}
		if encryptedMetadata, err := encryptor.EncryptString(metadata); err == nil {
			metadata = encryptedMetadata
		}
	}

	return pb.FileItem_builder{
		LocalId:  &f.LocalID,
		ServerId: &f.ServerID,
		Name:     &name,
		Size:     &f.Size,
		Metadata: &metadata,
		Checksum: &f.Checksum,
		Timemap: pb.TimeMap_builder{
			ChangeTime: &f.ChangeTime,
			SyncTime:   &f.SyncTime,
		}.Build(),
		Deleted: &f.Deleted,
	}.Build()
}

// FromProtoFile converts a Protobuf FileItem to a domain FileData model.
// Расшифровывает чувствительные данные после получения, если установлен Encryptor.
func FromProtoFile(pbFile *pb.FileItem) FileData {
	name := pbFile.GetName()
	metadata := pbFile.GetMetadata()

	// Расшифровываем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedName, err := encryptor.DecryptString(name); err == nil {
			name = decryptedName
		}
		if decryptedMetadata, err := encryptor.DecryptString(metadata); err == nil {
			metadata = decryptedMetadata
		}
	}

	return FileData{
		LocalID:    pbFile.GetLocalId(),
		ServerID:   pbFile.GetServerId(),
		Name:       name,
		Size:       pbFile.GetSize(),
		Metadata:   metadata,
		Checksum:   pbFile.GetChecksum(),
		ChangeTime: pbFile.GetTimemap().GetChangeTime(),
		SyncTime:   pbFile.GetTimemap().GetSyncTime(),
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
		"metadata":   f.Metadata,
		"size":       f.Size,
		"checksum":   f.Checksum,
		"changeTime": f.ChangeTime,
		"syncTime":   f.SyncTime,
		"deleted":    f.Deleted,
	}
}

// FromMapFile converts a map to a domain FileData model.
// Расшифровывает чувствительные данные после получения, если установлен Encryptor.
func FromMapFile(data map[string]interface{}) (FileData, error) {
	size, err := InterfaceToInt64(data["size"])
	if err != nil {
		return FileData{}, fmt.Errorf("failed to convert size for file: %w", err)
	}
	deleted, _ := InterfaceToBool(data["deleted"])
	changeTime, _ := InterfaceToInt64(data["changeTime"])
	syncTime, _ := InterfaceToInt64(data["syncTime"])

	name := InterfaceToString(data["name"])
	metadata := InterfaceToString(data["metadata"])

	// Расшифровываем данные, если установлен Encryptor
	if encryptor := GetGlobalEncryptor(); encryptor != nil {
		if decryptedName, err := encryptor.DecryptString(name); err == nil {
			name = decryptedName
		}
		if decryptedMetadata, err := encryptor.DecryptString(metadata); err == nil {
			metadata = decryptedMetadata
		}
	}

	return FileData{
		LocalID:    InterfaceToString(data["id"]),
		ServerID:   InterfaceToString(data["server_id"]),
		Name:       name,
		Size:       size,
		Metadata:   metadata,
		Checksum:   InterfaceToString(data["checksum"]),
		ChangeTime: changeTime,
		SyncTime:   syncTime,
		Deleted:    deleted,
	}, nil
}
