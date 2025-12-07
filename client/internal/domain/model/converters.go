package model

import (
	pb "gophKeeper/internal/proto"
	"time"
)

// --- TextData Converters ---

// ToProto converts a domain TextData model to a Protobuf TextData model.
func (t *TextData) ToProto() *pb.TextData {
	changeTime := t.ChangeTime.UnixNano()
	return pb.TextData_builder{
		Id:         &t.ID,
		Title:      &t.Title,
		Text:       &t.Text,
		ChangeTime: &changeTime,
	}.Build()
}

// FromProtoTexts converts a slice of Protobuf TextData to a slice of domain TextData models.
func FromProtoTexts(pbTexts []*pb.TextData) []TextData {
	texts := make([]TextData, len(pbTexts))
	for i, pbText := range pbTexts {
		texts[i] = TextData{
			ID:         pbText.GetId(),
			Title:      pbText.GetTitle(),
			Text:       pbText.GetText(),
			ChangeTime: time.Unix(0, pbText.GetChangeTime()),
		}
	}
	return texts
}

// --- Card Converters ---

// ToProto converts a domain Card model to a Protobuf CardData model.
func (c *Card) ToProto() *pb.CardData {
	changeTime := c.ChangeTime.UnixNano()
	return pb.CardData_builder{
		Id:         &c.ID,
		Number:     &c.Number,
		Holder:     &c.Holder,
		Expiry:     &c.Expiry,
		Cvv:        &c.CVV,
		ChangeTime: &changeTime,
	}.Build()
}

// FromProtoCards converts a slice of Protobuf CardData to a slice of domain Card models.
func FromProtoCards(pbCards []*pb.CardData) []Card {
	cards := make([]Card, len(pbCards))
	for i, pbCard := range pbCards {
		cards[i] = Card{
			ID:         pbCard.GetId(),
			Number:     pbCard.GetNumber(),
			Holder:     pbCard.GetHolder(),
			Expiry:     pbCard.GetExpiry(),
			CVV:        pbCard.GetCvv(),
			ChangeTime: time.Unix(0, pbCard.GetChangeTime()),
		}
	}
	return cards
}

// --- Password Converters ---

// ToProto converts a domain Password model to a Protobuf PasswordData model.
func (p *Password) ToProto() *pb.PasswordData {
	changeTime := p.ChangeTime.UnixNano()
	return pb.PasswordData_builder{
		Id:         &p.ID,
		Login:      &p.Login,
		Password:   &p.Password,
		Metadata:   &p.Description, // Используем Description как Metadata
		ChangeTime: &changeTime,
	}.Build()
}

// FromProtoPasswords converts a slice of Protobuf PasswordData to a slice of domain Password models.
func FromProtoPasswords(pbPasswords []*pb.PasswordData) []Password {
	passwords := make([]Password, len(pbPasswords))
	for i, pbPass := range pbPasswords {
		passwords[i] = Password{
			ID:          pbPass.GetId(),
			Login:       pbPass.GetLogin(),
			Password:    pbPass.GetPassword(),
			Description: pbPass.GetMetadata(), // Используем Metadata как Description
			ChangeTime:  time.Unix(0, pbPass.GetChangeTime()),
		}
	}
	return passwords
}

// --- FileData Converters ---

// ToProto converts a domain FileData model to a Protobuf FileData model.
func (f *FileData) ToProto() *pb.FileData {
	changeTime := f.ChangeTime.UnixNano()
	size := f.Size
	return pb.FileData_builder{
		Id:         &f.ID,
		Name:       &f.Name,
		Metadata:   &f.Metadata,
		Size:       &size,
		ChangeTime: &changeTime,
	}.Build()
}

// FromProtoFiles converts a slice of Protobuf FileData to a slice of domain FileData models.
func FromProtoFiles(pbFiles []*pb.FileData) []FileData {
	files := make([]FileData, len(pbFiles))
	for i, pbFile := range pbFiles {
		files[i] = FileData{
			ID:         pbFile.GetId(),
			Name:       pbFile.GetName(),
			Metadata:   pbFile.GetMetadata(),
			Size:       pbFile.GetSize(),
			ChangeTime: time.Unix(0, pbFile.GetChangeTime()),
		}
	}
	return files
}
