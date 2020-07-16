package consent_utils

import (
	"encoding/json"
	"fmt"
	"github.com/cbroglie/mustache"
	"github.com/nuts-foundation/nuts-consent-logic/pkg"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"regexp"
	"strings"
	"time"
)

type ConsentUtils struct {

}


func (cl ConsentUtils) CreateFhirConsentResource(data events.ConsentData) (string, error) {

	var (
		actorAgbs []string
		err       error
		versionID uint
		res       string
	)
	actorAgbs = append(actorAgbs, valueFromUrn(data.ActorID))

	if versionID, err = cl.getVersionID(data); versionID == 0 || err != nil {
		err = fmt.Errorf("could not determine versionId: %w", err)
		//logger().Error(err)
		return "", err
	}

	// FIXME: the current class property is a string, and should be rewritten to an array
	dataClasses := make([]map[string]string, 1)
	viewModel := map[string]interface{}{
		"subjectBsn":   valueFromUrn(data.SubjectID),
		"actorAgbs":    actorAgbs,
		"custodianAgb": valueFromUrn(data.CustodianID),
		"period": map[string]string{
			"Start": data.Start.Format(time.RFC3339),
		},
		"dataClass":   dataClasses,
		"lastUpdated": time.Now().Format(time.RFC3339),
		"versionId":   fmt.Sprintf("%d", versionID),
	}

	// split data class identifiers
	for i, dc := range []string{data.Class} { // Fixme: rewrite the data.class to an array
		dataClasses[i] = make(map[string]string)
		sdc := string(dc)
		li := strings.LastIndex(sdc, ":")
		if li < 0 {
			li = 0
		}
		dataClasses[i]["system"] = sdc[0:li]
		dataClasses[i]["code"] = sdc[li+1:]
	}

	// Fixme: add proof to consentData
	//if record.ConsentProof != nil {
	//	viewModel["consentProof"] = derefPointers(record.ConsentProof)
	//}

	// Fixme: add performer to the consentData
	//if performer != "" {
	//	viewModel["performerId"] = valueFromUrn(string(performer))
	//}

	if !data.End.IsZero() {
		(viewModel["period"].(map[string]string))["End"] = data.End.Format(time.RFC3339)
	}

	if res, err = mustache.Render(template, viewModel); err != nil {
		// uh oh
		return "", err
	}

	// filter out last comma out [{},{},] since mustache templates cannot handle this:
	// https://stackoverflow.com/questions/6114435/in-mustache-templating-is-there-an-elegant-way-of-expressing-a-comma-separated-l
	re := regexp.MustCompile(`\},(\s*)]`)
	res = re.ReplaceAllString(res, `}$1]`)

	return cleanupJSON(res)
}
// getVersionID returns the correct version number for the given record. "1" for a new record and "old + 1" for an update
func (cl ConsentUtils) getVersionID(data events.ConsentData) (uint, error) {
	// FIXME: fetch version from another readmodel than the consentstore
	//if record.PreviousRecordhash == nil {
	//	return 1, nil
	//}
	//
	//cr, err := cl.NutsConsentStore.FindConsentRecordByHash(context.TODO(), *record.PreviousRecordhash, true)
	//if err != nil {
	//	return 0, err
	//}
	//
	//return cr.Version + 1, nil
	return 1, nil
}

func valueFromUrn(urn string) string {
	segments := strings.Split(urn, ":")
	return segments[len(segments)-1]
}

func derefPointers(docReference *pkg.DocumentReference) map[string]interface{} {
	m := map[string]interface{}{}

	if docReference == nil {
		return nil
	}

	m["Title"] = docReference.Title
	m["ID"] = docReference.ID

	if docReference.Hash != nil {
		m["Hash"] = *docReference.Hash
	}

	if docReference.ContentType != nil {
		m["ContentType"] = *docReference.ContentType
	}

	if docReference.URL != nil {
		m["URL"] = *docReference.URL
	}

	return m
}

// clean up the json hash
func cleanupJSON(value string) (string, error) {
	var parsedValue interface{}
	if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
		return "", err
	}
	cleanValue, err := json.Marshal(parsedValue)
	if err != nil {
		return "", err
	}
	return string(cleanValue), nil
}
