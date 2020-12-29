package certificate

import (
	"bytes"
	"encoding/gob"
)

type dts struct {
	ServerKey  []byte
	ServerCert []byte
	CaKey      []byte
	CaCert     []byte
	ClientKey  []byte
	ClientCert []byte
}

// GobDecode the config
func (c *CertConfigCarrier) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	var dto dts
	err := dec.Decode(&dto)
	if err != nil {
		return err
	}
	c.caCert = dto.CaCert
	c.caKey = dto.CaKey
	c.serverCert = dto.ServerCert
	c.serverKey = dto.ServerKey
	c.clientCert = dto.ClientCert
	c.clientKey = dto.ClientKey
	serverCert, err := generateCertificate(c.serverCert, c.serverKey)
	if err != nil {
		return err
	}
	err = c.generateServerConf(serverCert)
	if err != nil {
		return err
	}
	clientCert, err := generateCertificate(c.clientCert, c.clientKey)
	if err != nil {
		return err
	}
	err = c.generateClientConf(clientCert)
	if err != nil {
		return err
	}
	return nil
}

// GobEncode the config
func (c CertConfigCarrier) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(dts{
		ServerKey:  c.serverKey,
		ServerCert: c.serverCert,
		CaKey:      c.caKey,
		CaCert:     c.caCert,
		ClientKey:  c.clientKey,
		ClientCert: c.clientCert,
	})
	return b.Bytes(), err
}
