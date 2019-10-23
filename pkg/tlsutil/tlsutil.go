package tlsutil

import (
	"crypto/x509"
	"fmt"
)

func GenerateSubject(cert *x509.Certificate) string {
	subject := "subject="
	if cert.Subject.Country != nil {
		subject = fmt.Sprintf("%s/C=%s", subject, cert.Subject.Country[0])
	}
	if cert.Subject.Province != nil {
		subject = fmt.Sprintf("%s/ST=%s", subject, cert.Subject.Province[0])
	}
	if cert.Subject.Locality != nil {
		subject = fmt.Sprintf("%s/L=%s", subject, cert.Subject.Locality[0])
	}
	if cert.Subject.Organization != nil {
		subject = fmt.Sprintf("%s/O=%s", subject, cert.Subject.Organization[0])
	}
	if cert.Subject.OrganizationalUnit != nil {
		subject = fmt.Sprintf("%s/OU=%s", subject, cert.Subject.OrganizationalUnit[0])
	}
	return fmt.Sprintf("%s/CN=%s", subject, cert.Subject.CommonName)
}

func GenerateIssuer(cert *x509.Certificate) string {
	issuer := "issuer="
	if cert.Issuer.Country != nil {
		issuer = fmt.Sprintf("%s/C=%s", issuer, cert.Issuer.Country[0])
	}
	if cert.Issuer.Province != nil {
		issuer = fmt.Sprintf("%s/ST=%s", issuer, cert.Issuer.Province[0])
	}
	if cert.Issuer.Locality != nil {
		issuer = fmt.Sprintf("%s/L=%s", issuer, cert.Issuer.Locality[0])
	}
	if cert.Issuer.Organization != nil {
		issuer = fmt.Sprintf("%s/O=%s", issuer, cert.Issuer.Organization[0])
	}
	if cert.Issuer.OrganizationalUnit != nil {
		issuer = fmt.Sprintf("%s/OU=%s", issuer, cert.Issuer.OrganizationalUnit[0])
	}
	return fmt.Sprintf("%s/CN=%s", issuer, cert.Issuer.CommonName)
}
