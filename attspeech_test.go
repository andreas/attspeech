package attspeech

import (
	"bytes"
	// "fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestClient(t *testing.T) {
	Convey("Creating a new client should set values correctly", t, func() {
		client := New("foo", "bar", "")
		So(client.APIBase, ShouldEqual, "https://api.att.com")
		So(client.ID, ShouldEqual, "foo")
		So(client.Secret, ShouldEqual, "bar")
		So(client.STTResource, ShouldEqual, STTResource)
		So(client.TTSResource, ShouldEqual, TTSResource)
	})
}

func TestCustomApiBase(t *testing.T) {
	Convey("Creating a new client with a custom API URL should set values correctly", t, func() {
		client := New("foo", "bar", "http://foobar.com")
		So(client.APIBase, ShouldEqual, "http://foobar.com")
	})
}

func TestToDash(t *testing.T) {
	Convey("Converting struct elements to HTTP headers", t, func() {
		Convey("Should leave this word undashed", func() {
			word := toDash("Foobar")
			So(word, ShouldEqual, "Foobar")
		})
		Convey("Should put one dash in this word", func() {
			word := toDash("FooBar")
			So(word, ShouldEqual, "Foo-Bar")
		})
		Convey("Should put one dash in this word with multiple caps", func() {
			word := toDash("FooBarBaz")
			So(word, ShouldEqual, "Foo-BarBaz")
		})
	})
}

func TestGetTokens(t *testing.T) {
	Convey("Should get proper tokens", t, func() {
		ts := serveHTTP(t)
		client := New("foo", "bar", "")
		client.APIBase = ts.URL
		err := client.SetAuthTokens()

		So(err, ShouldBeNil)
		scopes := [3]string{"SPEECH", "TTS", "STTC"}
		for _, scope := range scopes {
			So(client.Tokens[scope].AccessToken, ShouldEqual, "123")
			So(client.Tokens[scope].RefreshToken, ShouldEqual, "456")
		}
	})
}

func TestSpeechToText(t *testing.T) {
	Convey("Should return a recognition of an audio file", t, func() {
		Convey("When no ContentType is provided", func() {
			ts := serveHTTP(t)
			client := New(os.Getenv("ATT_APP_KEY"), os.Getenv("ATT_APP_SECRET"), "")
			client.APIBase = ts.URL
			client.SetAuthTokens()
			apiRequest := client.NewAPIRequest(STTResource)
			response, err := client.SpeechToText(apiRequest)
			So(response, ShouldBeNil)
			So(err.Error(), ShouldEqual, "A ContentType must be provided")
		})
		Convey("When no Data is provided", func() {
			ts := serveHTTP(t)
			client := New(os.Getenv("ATT_APP_KEY"), os.Getenv("ATT_APP_SECRET"), "")
			client.APIBase = ts.URL
			client.SetAuthTokens()
			apiRequest := client.NewAPIRequest(STTResource)
			apiRequest.ContentType = "audio/x-wav"
			response, err := client.SpeechToText(apiRequest)
			So(response, ShouldBeNil)
			So(err.Error(), ShouldEqual, "Data to convert to text must be provided")
		})
		Convey("When an invalid ContentType is provided", func() {
			ts := serveHTTP(t)
			client := New(os.Getenv("ATT_APP_KEY"), os.Getenv("ATT_APP_SECRET"), "")
			client.APIBase = ts.URL
			client.SetAuthTokens()

			// Read the test file
			data := &bytes.Buffer{}
			file, err := os.Open("./test/test.wav")
			So(err, ShouldBeNil)
			defer file.Close()
			_, err = io.Copy(data, file)
			So(err, ShouldBeNil)

			apiRequest := client.NewAPIRequest(STTResource)
			apiRequest.Data = data
			apiRequest.ContentType = "foo/bar"

			response, err := client.SpeechToText(apiRequest)

			So(response, ShouldBeNil)
			So(err.Error(), ShouldEqual, "SVC0002 - Invalid input value for message part %1 - Content-Type")
		})
		Convey("When a valid ContentType is provided", func() {
			ts := serveHTTP(t)
			client := New(os.Getenv("ATT_APP_KEY"), os.Getenv("ATT_APP_SECRET"), "")
			client.APIBase = ts.URL
			client.SetAuthTokens()

			// Read the test file
			data := &bytes.Buffer{}
			file, err := os.Open("./test/test.wav")
			So(err, ShouldBeNil)
			defer file.Close()
			_, err = io.Copy(data, file)
			So(err, ShouldBeNil)

			apiRequest := client.NewAPIRequest(STTResource)
			apiRequest.Data = data
			apiRequest.ContentType = "audio/wav"
			response, err := client.SpeechToText(apiRequest)

			So(err, ShouldBeNil)
			So(response.Recognition.Status, ShouldEqual, "OK")
			So(response.Recognition.NBest[0].ResultText, ShouldEqual, "If you wish to keep this new greeting press one if you wish to record the greeting press two. To re store your old greeting in return to the administration menu. Press the star key.")
		})
	})
}

func TestTextToSpeech(t *testing.T) {
	Convey("Should handle Text to Speech (TTS)", t, func() {
		ts := serveHTTP(t)
		client := New(os.Getenv("ATT_APP_KEY"), os.Getenv("ATT_APP_SECRET"), "")
		client.APIBase = ts.URL
		client.SetAuthTokens()
		Convey("Should set the default ContentType", func() {
			apiRequest := client.NewAPIRequest(TTSResource)
			apiRequest.Text = "foobar"
			So(apiRequest.ContentType, ShouldEqual, "text/plain")
		})
		Convey("Should return an error if Text not set", func() {
			apiRequest := client.NewAPIRequest(TTSResource)
			_, err := client.TextToSpeech(apiRequest)
			So(err.Error(), ShouldEqual, "Text to convert to speech must be provided")
		})
		Convey("Should return an error if an invalid ContentType", func() {
			apiRequest := client.NewAPIRequest(TTSResource)
			apiRequest.ContentType = "foo/bar"
			apiRequest.Text = "foobar"
			response, err := client.TextToSpeech(apiRequest)
			So(response, ShouldBeNil)
			So(err.Error(), ShouldEqual, "SVC0002 - Invalid input value for message part %1 - Content-Type")
		})
	})
}

func serveHTTP(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, OauthResource) {
			w.WriteHeader(200)
			w.Write(oauthJSON())
			return
		}
		if strings.Contains(req.RequestURI, STTResource) {
			checkHeaders(t, req)
			if req.Header.Get("Content-Type") == "foo/bar" {
				w.WriteHeader(400)
				w.Write(contentTypeErrorJSON())
				return
			}
			w.WriteHeader(200)
			w.Write(recognitionJSON())
			return
		}
		if strings.Contains(req.RequestURI, TTSResource) {
			checkHeaders(t, req)
			if req.Header.Get("Content-Type") == "foo/bar" {
				w.WriteHeader(400)
				w.Write(contentTypeErrorJSON())
				return
			}
			data, _ := ioutil.ReadFile("./test/tts_test.wav")
			w.WriteHeader(200)
			w.Write(data)
			return
		}
	}))
}

func checkHeaders(t *testing.T, req *http.Request) {
	Convey("Default headers should be set", t, func() {
		So(req.Header.Get("X-Arg"), ShouldEqual, "ClientApp=GoLibForATTSpeech,ClientVersion=0.1,DeviceType=amd64,DeviceOs=darwin")
		So(req.Header.Get("User-Agent"), ShouldEqual, "Golang net/http")
		So(req.Header.Get("Accept"), ShouldNotBeNil)
		So(req.Header.Get("Authorization"), ShouldEqual, "Bearer 123")
	})
}

func oauthJSON() []byte {
	return []byte(`
	{
	    "access_token":"123",
	    "token_type": "bearer",
	    "expires_in":500,
	    "refresh_token":"456"
	}
	`)
}

func recognitionJSON() []byte {
	return []byte(`
		{
		    "Recognition": {
		        "Info": {
		            "metrics": {
		                "audioBytes": 92102,
		                "audioTime": 11.5100002
		            }
		        },
		        "NBest": [
		            {
		                "Confidence": 0.667999999,
		                "Grade": "accept",
		                "Hypothesis": "if you wish to keep this new greeting press one if you wish to record the greeting press two to re store your old greeting in return to the administration menu press the star key",
		                "LanguageId": "en-US",
		                "ResultText": "If you wish to keep this new greeting press one if you wish to record the greeting press two. To re store your old greeting in return to the administration menu. Press the star key.",
		                "WordScores": [
		                    1,
		                    1,
		                    1,
		                    1,
		                    1,
		                    1,
		                    0.449,
		                    0.449,
		                    0.449,
		                    1,
		                    1,
		                    1,
		                    1,
		                    1,
		                    0.289,
		                    0.23,
		                    0.31,
		                    0.37,
		                    1,
		                    1,
		                    0.36,
		                    0.15,
		                    0.14,
		                    0.189,
		                    0.589,
		                    0.07,
		                    0.189,
		                    0.37,
		                    0.4,
		                    0.37,
		                    1,
		                    1,
		                    1,
		                    1,
		                    1
		                ],
		                "Words": [
		                    "If",
		                    "you",
		                    "wish",
		                    "to",
		                    "keep",
		                    "this",
		                    "new",
		                    "greeting",
		                    "press",
		                    "one",
		                    "if",
		                    "you",
		                    "wish",
		                    "to",
		                    "record",
		                    "the",
		                    "greeting",
		                    "press",
		                    "two.",
		                    "To",
		                    "re",
		                    "store",
		                    "your",
		                    "old",
		                    "greeting",
		                    "in",
		                    "return",
		                    "to",
		                    "the",
		                    "administration",
		                    "menu.",
		                    "Press",
		                    "the",
		                    "star",
		                    "key."
		                ]
		            }
		        ],
		        "ResponseId": "cf928a1adb259abf409da1993543fcdc",
		        "Status": "OK"
		    }
		}
	`)
}

func contentTypeErrorJSON() []byte {
	return []byte(`
	{
	    "RequestError": {
	        "ServiceException": {
	            "MessageId": "SVC0002",
	            "Text": "Invalid input value for message part %1",
	            "Variables": "Content-Type"
	        }
	    }
	}
	`)
}
