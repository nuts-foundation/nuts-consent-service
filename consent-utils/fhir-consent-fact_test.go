package consent_utils

import (
	"testing"
)

func TestFhirConsentFact(t *testing.T) {
	sut := FhirConsentFact{payload: []byte("{\"category\":[{\"coding\":[{\"code\":\"64292-6\",\"system\":\"http://loinc.org\"}]}],\"meta\":{\"lastUpdated\":\"2020-07-20T09:31:54+02:00\",\"versionId\":\"1\"},\"organization\":[{\"identifier\":{\"system\":\"urn:oid:2.16.840.1.113883.2.4.6.1\",\"value\":\"123\"}}],\"patient\":{\"identifier\":{\"system\":\"urn:oid:2.16.840.1.113883.2.4.6.3\",\"value\":\"999\"}},\"policyRule\":{\"coding\":[{\"code\":\"OPTIN\",\"system\":\"http://terminology.hl7.org/CodeSystem/v3-ActCode\"}]},\"provision\":{\"actor\":[{\"reference\":{\"identifier\":{\"system\":\"urn:oid:2.16.840.1.113883.2.4.6.1\",\"value\":\"456\"}},\"role\":{\"coding\":[{\"code\":\"PRCP\",\"system\":\"http://terminology.hl7.org/CodeSystem/v3-ParticipationType\"}]}}],\"period\":{\"start\":\"2020-07-20T09:31:54+02:00\"},\"provision\":[{\"action\":[{\"coding\":[{\"code\":\"access\",\"system\":\"http://terminology.hl7.org/CodeSystem/consentaction\"}]}],\"class\":[{\"code\":\"ransfer\",\"system\":\"\"}],\"type\":\"permit\"}]},\"resourceType\":\"Consent\",\"scope\":{\"coding\":[{\"code\":\"patient-privacy\",\"system\":\"http://terminology.hl7.org/CodeSystem/consentscope\"}]},\"verification\":[{\"verified\":true,\"verifiedWith\":{\"identifier\":{\"system\":\"urn:oid:2.16.840.1.113883.2.4.6.3\",\"value\":\"999\"},\"type\":\"Patient\"}}]}")}
	testcases := map[string]struct {
		sut ConsentFact
		exp string
		got func() string
	}{
		"parse the actor": {
			sut: sut,
			exp: "urn:oid:2.16.840.1.113883.2.4.6.1:456",
			got: func() string {
				return sut.Actor()
			},
		},
		"parse the custodian": {
			sut: sut,
			exp: "urn:oid:2.16.840.1.113883.2.4.6.1:123",
			got: func() string {
				return sut.Custodian()
			},
		},
		"parse the subject": {
			sut: sut,
			exp: "urn:oid:2.16.840.1.113883.2.4.6.3:999",
			got: func() string {
				return sut.Subject()
			},
		},
	}

	for name, testcase := range testcases{
		t.Run(name, func(t *testing.T) {
			exp := testcase.exp
			got := testcase.got()
			if got != exp {
				t.Fail()
				t.Logf("exp: %s, got: %s\n", exp, got)
			}
		})
	}

}
