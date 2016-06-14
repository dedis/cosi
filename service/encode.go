package service

import (
	"encoding/base64"
	"encoding/json"
)

type jsonSignature struct {
	Sum       string `json:"sum"`
	Signature string `json:"signature"`
}

// MarshalJSON implements golang's JSON marshal interface
func (s *SignatureResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSignature{
		Sum:       base64.StdEncoding.EncodeToString(s.Sum),
		Signature: base64.StdEncoding.EncodeToString(s.Signature),
	})
}

// UnmarshalJSON implements golang's JSON unmarshal interface
func (s *SignatureResponse) UnmarshalJSON(data []byte) error {
	jsonSig := &jsonSignature{}
	if err := json.Unmarshal(data, jsonSig); err != nil {
		return err
	}
	var err error
	if s.Sum, err = base64.StdEncoding.DecodeString(jsonSig.Sum); err != nil {
		return err
	}
	s.Signature, err = base64.StdEncoding.DecodeString(jsonSig.Signature)
	return err
}
