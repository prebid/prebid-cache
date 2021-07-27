package backends

import (
	"errors"
	"net/http"
	"testing"

	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id, err := utils.GenerateRandomId()
	if err != nil {
		t.Errorf("Error generating version 4 UUID")
	}

	expected := id[0:4]
	got := azureTable.makePartitionKey(id)

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}

func TestShortPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id := "abc"
	got := azureTable.makePartitionKey(id)

	if got != id {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", id, got)
	}
}

func TestEmptyPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id := ""
	got := azureTable.makePartitionKey(id)

	if got != id {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", id, got)
	}
}

func TestPartitionKeyHeader(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id, err := utils.GenerateRandomId()
	if err != nil {
		t.Errorf("Error generating version 4 UUID")
	}

	expected := "[\"" + id[0:4] + "\"]"

	got := azureTable.wrapForHeader(azureTable.makePartitionKey(id))

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}

func TestUnmarshallAzureErrorResponse(t *testing.T) {
	type testExpectedValues struct {
		errDescription *AzureErrorDesc
		unmarshallErr  error
	}

	testGroups := []struct {
		desc        string
		inTestCases [][]byte
		expected    testExpectedValues
	}{
		{
			"Azure empty response",
			[][]byte{
				nil,
			},
			testExpectedValues{
				errDescription: nil,
				unmarshallErr:  errors.New("Azure CosmoDB Response: Empty response body"),
			},
		},
		{
			"Azure malformed JSON response",
			[][]byte{
				[]byte("malformed"),
			},
			testExpectedValues{
				errDescription: nil,
				unmarshallErr:  errors.New("Azure CosmoDB Response: Could not unmarshal"),
			},
		},
		{
			"Azure Cosmos DB response comes with an empty error message",
			[][]byte{
				[]byte(`{"code":"BadRequest","message":""}`),
			},
			testExpectedValues{
				errDescription: nil,
				unmarshallErr:  errors.New("Azure CosmoDB Response: Message field is empty"),
			},
		},
		{
			"Azure Cosmos DB response does not come with an 'Errors' field on its message field",
			[][]byte{
				[]byte(`{"code":"BadRequest","message":"\n"}`),
				[]byte(`{"code":"BadRequest","message":"Message: ActivityId: <some_activity_id>"}`),
				[]byte(`{"code":"BadRequest","message":"Message: \nActivityId: <some_activity_id>"}`),
				[]byte(`{"code":"BadRequest","message":"Message: \r"}`),
			},
			testExpectedValues{
				errDescription: nil,
				unmarshallErr:  errors.New("Azure CosmoDB Response: Couldn't find 'Errors' field"),
			},
		},
		{
			"Azure Cosmos DB response comes with a message field value that carries no JSON object",
			[][]byte{
				[]byte(`{"code":"BadRequest","message":"Message: {\"Errors\":[\"Some error message.\"]\r\n"}`),
			},
			testExpectedValues{
				errDescription: nil,
				unmarshallErr:  errors.New("Azure CosmoDB Response: Couldn't find JSON object inside the message response"),
			},
		},
		{
			"Azure Cosmos DB response comes with a JSON object inside its message field that could not be unmarshalled",
			[][]byte{
				[]byte(`{"code":"BadRequest","message":"Message: {\"Errors\":malformed}\r\n"}`),
			},
			testExpectedValues{
				errDescription: nil,
				unmarshallErr:  errors.New("Azure CosmoDB Response: Could not unmarshal message field value"),
			},
		},
		{
			"Azure Cosmos DB response comes with a JSON object inside its message field that could not be unmarshalled",
			[][]byte{
				[]byte(`{"code":"BadRequest","message":"Message: {\"Errors\":[]}\r\n"}`),
			},
			testExpectedValues{
				errDescription: nil,
				unmarshallErr:  errors.New("Azure CosmoDB Response: Empty 'Errors' field inside the 'message' value"),
			},
		},
		{
			"An actual Azure Cosmos DB error found in the response",
			[][]byte{
				[]byte(`{"code":"BadRequest","message":"Message: {\"Errors\":[\"The collection cannot be accessed with this SDK version as it was created with newer SDK version.\"]}\r\nActivityId: <some_activity_id>, Request URI: someUri.org, RequestStats: \r\nRequestStartTime: 2021-07-13T18:13:12.2143016Z, RequestEndTime: 2021-07-13T18:13:12.2242985Z,  Number of regions attempted:1\r\nResponseTime: 2021-07-13T18:13:12.2242985Z, StoreResult: StorePhysicalAddress: physicaladdresurl.com, LSN: 3, GlobalCommittedLsn: 3, PartitionKeyRangeId: 0, IsValid: True, StatusCode: 400, SubStatusCode: 0, RequestCharge: 0, ItemLSN: -1, SessionToken: 3, UsingLocalLSN: False, TransportException: null, BELatencyMs: 0.25, ActivityId: <someID>, ResourceType: Document, OperationType: Create\r\n, SDK: Microsoft.Azure.Documents.Common/2.14.0"}`),
			},
			testExpectedValues{
				errDescription: &AzureErrorDesc{
					Errors: []string{"The collection cannot be accessed with this SDK version as it was created with newer SDK version."},
				},
				unmarshallErr: nil,
			},
		},
	}

	for _, tg := range testGroups {
		for i, test := range tg.inTestCases {
			// run
			outErrorDesc, outUnmarshalErr := unmarshallAzureErrorResponse(test)

			// assertions
			assert.Equalf(t, tg.expected.errDescription, outErrorDesc, "[%d] %s", i, tg.desc)
			assert.Equalf(t, tg.expected.unmarshallErr, outUnmarshalErr, "[%d] %s", i, tg.desc)
		}
	}
}

func TestInterpretAzurePutResponse(t *testing.T) {
	testCases := []struct {
		desc                 string
		getMockAzureResponse func() *fasthttp.Response
		expectedErr          error
	}{
		{
			"interpret a nil Azure Put Response",
			func() *fasthttp.Response {
				return nil
			},
			errors.New(http.StatusText(http.StatusInternalServerError)),
		},
		{
			"interpret an Azure Put Response were entry wasn't overwritten because prebid-cache doesn't implement 'upsert'",
			func() *fasthttp.Response {
				azureServiceMockResponse := &fasthttp.Response{}
				azureServiceMockResponse.SetStatusCode(http.StatusConflict)
				return azureServiceMockResponse
			},
			utils.RecordExistsError{},
		},
		{
			"interpret an Azure Put Response that comes with a non-conflict error that could not be correclty unmarshalled",
			func() *fasthttp.Response {
				azureServiceMockResponse := &fasthttp.Response{}
				azureServiceMockResponse.SetStatusCode(http.StatusBadRequest)

				malformedErrorResponseBody := []byte(`{"code":"BadRequest","message":"Message: {\"Errors\":malformed}\r\n"}`)
				azureServiceMockResponse.BodyWriter().Write(malformedErrorResponseBody)
				return azureServiceMockResponse
			},
			errors.New("Azure CosmoDB Response: Could not unmarshal message field value"),
		},
		{
			"interpret an Azure Put Response that comes with a non-conflict error that could be correctly unmarshalled and read",
			func() *fasthttp.Response {
				azureServiceMockResponse := &fasthttp.Response{}
				azureServiceMockResponse.SetStatusCode(http.StatusBadRequest)

				malformedErrorResponseBody := []byte(`{"code":"BadRequest","message":"Message: {\"Errors\":[\"The collection cannot be accessed with this SDK version as it was created with newer SDK version.\"]}\r\nActivityId: <some_activity_id>\r\n, SDK: Microsoft.Azure.Documents.Common/2.14.0"}`)
				azureServiceMockResponse.BodyWriter().Write(malformedErrorResponseBody)
				return azureServiceMockResponse
			},
			errors.New("The collection cannot be accessed with this SDK version as it was created with newer SDK version."),
		},
		{
			"interpret an Azure Put Response of an element that was successfully written to the documents service storage",
			func() *fasthttp.Response {
				azureServiceMockResponse := &fasthttp.Response{}
				azureServiceMockResponse.SetStatusCode(http.StatusCreated)

				successResponseBody := []byte(`{"id":"cust-key-maps-to-no-value-in-backend","value":"xml\u003ctag\u003eYourXMLcontentgoeshere\u003c/tag\u003e","partitionkey":"somePartitionKey"}`)
				azureServiceMockResponse.BodyWriter().Write(successResponseBody)
				return azureServiceMockResponse
			},
			nil,
		},
	}

	for _, tc := range testCases {
		// run
		outError := interpretAzurePutResponse(tc.getMockAzureResponse())

		// assertions
		assert.Equalf(t, tc.expectedErr, outError, tc.desc)
	}
}

func TestValidatePutArgs(t *testing.T) {
	testCases := []struct {
		desc           string
		inKey, inValue string
		expectedErr    error
	}{
		{
			desc:        "Both key and value are empty, expect Invalid Key error since the key gets checked first",
			inKey:       "",
			inValue:     "",
			expectedErr: errors.New("Invalid Key"),
		},
		{
			desc:        "Empty key, expect Invalid Key error",
			inKey:       "",
			inValue:     "xml\u003ctag\u003eYourXMLcontentgoeshere\u003c/tag\u003e",
			expectedErr: errors.New("Invalid Key"),
		},
		{
			desc:        "Empty value, expect Invalid Value error",
			inKey:       "cust-key-maps-to-no-value-in-backend",
			inValue:     "",
			expectedErr: errors.New("Invalid Value"),
		},
		{
			desc:        "Non-empty key and value. Expect no error.",
			inKey:       "cust-key-maps-to-no-value-in-backend",
			inValue:     "xml\u003ctag\u003eYourXMLcontentgoeshere\u003c/tag\u003e",
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		// run
		outErr := validatePutArgs(tc.inKey, tc.inValue)
		// assertions
		assert.Equalf(t, tc.expectedErr, outErr, tc.desc)
	}
}

func TestNewPutValue(t *testing.T) {
	testCases := []struct {
		desc                           string
		inKey, inValue, inPartitionKey string
		expectedPutBody                []byte
		expectedMarshalErr             error
	}{
		{
			desc:               "newPutValue should return the expected marshalled object successfully",
			inKey:              "cust-key-maps-to-no-value-in-backend",
			inValue:            "xml\u003ctag\u003eYourXMLcontentgoeshere\u003c/tag\u003e",
			inPartitionKey:     "somePArtitionKey",
			expectedPutBody:    []byte(`{"id":"cust-key-maps-to-no-value-in-backend","value":"xml\u003ctag\u003eYourXMLcontentgoeshere\u003c/tag\u003e","uuid":"somePArtitionKey"}`),
			expectedMarshalErr: nil,
		},
	}

	for _, tc := range testCases {
		// run
		outMarshalledObj, outMarshalError := newPutValue(tc.inKey, tc.inValue, tc.inPartitionKey)

		// assertions
		assert.Equalf(t, tc.expectedPutBody, outMarshalledObj, tc.desc)
		assert.Equalf(t, tc.expectedMarshalErr, outMarshalError, tc.desc)
	}
}
