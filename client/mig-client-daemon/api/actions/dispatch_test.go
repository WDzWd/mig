// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

type mockDispatcher struct{}

type mockAuthenticator struct{}

func TestDispatchHandler(t *testing.T) {
	catalog := actions.NewCatalog()
	module := &modules.Pkg{
		PackageName: "*libssl*",
	}
	target := []targeting.Query{
		&targeting.ByTag{
			TagName:  "operator",
			TagValue: "IT",
		},
	}
	validID, _ := catalog.Create(module, target, time.Hour)

	testCases := []struct {
		Description    string
		ActionID       ident.Identifier
		ExpectError    bool
		ExpectedStatus int
	}{
		{
			Description: `
We should be able to dispatch an action managed by the client daemon.
			`,
			ActionID:       validID,
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		{
			Description: `
We should not be able dispatch an action that does not exist.
			`,
			ActionID:       ident.Identifier("invalid"),
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Description: `
If the connection to the MIG API fails, we should get an internal error.
			`,
			ActionID:       ident.Identifier("irrelevant"),
			ExpectError:    true,
			ExpectedStatus: http.StatusInternalServerError,
		},
	}

	dispatcher := mockDispatcher{}
	authenticator := mockAuthenticator{}
	handler := NewDispatchHandler(&catalog, dispatcher, authenticator)
	router := mux.NewRouter()
	router.Handle("/v1/actions/{id}/dispatch", handler).Methods("PUT")
	server := httptest.NewServer(router)

	for caseNum, testCase := range testCases {
		t.Logf("Running TestDispatchHandler case #%d.\n%s\n", caseNum, testCase.Description)

		reqURL := fmt.Sprintf("%s/v1/actions/%s/dispatch", server.URL, testCase.ActionID)

		client := &http.Client{}
		request, _ := http.NewRequest("PUT", reqURL, nil)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}

		respData := dispatchResponse{}
		decoder := json.NewDecoder(response.Body)
		defer response.Body.Close()
		err = decoder.Decode(&respData)
		if err != nil {
			t.Fatal(err)
		}

		gotErr := respData.Error != nil
		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		} else if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", *respData.Error)
		}

		if testCase.ExpectedStatus != response.StatusCode {
			t.Errorf("Expected to get status %d but got %d", testCase.ExpectedStatus, response.StatusCode)
		}
	}
}

func (mockDispatcher) Dispatch(_ mig.Action, _ authentication.Authenticator) error {
	return nil
}

func (mockAuthenticator) Authenticate(_ *http.Request) error {
	return nil
}