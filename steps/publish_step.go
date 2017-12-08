package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Publisher contains the steps to publish a step.
type Publisher interface {
	// CreateDraft implements the first step of the publication flow. It requires
	// a manifest, the checksum and size of the tarball.
	CreateDraft(req *PublishStepRequest) (*PublishStepResponse, error)

	// UploadTarball uploads the step tarball using the response from the
	// CreateDraft endpoint.
	UploadTarball(uploadURL string, body io.Reader, size int64) error

	// FinishPublish post to the publish endpoint and the tarball is uploaded to
	// indicate that the step can be published.
	FinishPublish(token string) error
}

type PublishStepRequest struct {
	// checksum of the tarball containing the step
	Checksum string `json:"checksum,omitempty"`
	// size of the tarball containing the step
	Size int64 `json:"size,omitempty"`
	// manifest contains the manifest of the step
	Manifest *StepManifest `json:"manifest,omitempty"`
	// username
	Username string `json:"username,omitempty"`
}

type PublishStepResponse struct {
	// uploadUrl is the URL the client has to post the tarball to
	UploadUrl string `json:"uploadUrl,omitempty"`
	// token is the token to send to the done endpoint to notify the upload has
	// been finished
	Token string `json:"token,omitempty"`
	// expires is the expiration date of the uploadUrl
	Expires string `json:"expires,omitempty"`
}

// PublishStep uses ps to create a new step using manifest, tarball.
func PublishStep(ps Publisher, manifest *StepManifest, tarball io.Reader, username, checksum string, size int64) error {
	createDraftRequest := &PublishStepRequest{
		Username: username,
		Manifest: manifest,
		Checksum: checksum,
		Size:     size,
	}

	resp, err := ps.CreateDraft(createDraftRequest)
	if err != nil {
		return err
	}

	err = ps.UploadTarball(resp.UploadUrl, tarball, size)
	if err != nil {
		return err
	}

	err = ps.FinishPublish(resp.Token)
	if err != nil {
		return err
	}

	return nil
}

// NewRESTPublisher creates a publisher that uses the REST API.
func NewRESTPublisher(endpoint string, client *http.Client, stepsClient *http.Client) *RESTPublisher {
	ps := &RESTPublisher{
		endpoint:    endpoint,
		client:      client,
		stepsClient: stepsClient,
	}

	return ps
}

// RESTPublisher contains the steps to publish a step.
type RESTPublisher struct {
	endpoint    string
	client      *http.Client
	stepsClient *http.Client
}

var _ Publisher = (*RESTPublisher)(nil)

// generateURL takes s.endpoint and appends slugs to it.
func (s *RESTPublisher) generateURL(slugs ...string) string {
	path := strings.Join(slugs, "/")
	return fmt.Sprintf("%s/%s", s.endpoint, path)
}

// CreateDraft implements the first step of the publication flow. It requires a
// manifest, the checksum and size of the tarball.
func (s *RESTPublisher) CreateDraft(createDraftRequest *PublishStepRequest) (*PublishStepResponse, error) {
	log.Debug("Creating draft")

	b, err := json.Marshal(createDraftRequest)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to marshal create draft request")
	}

	req, err := http.NewRequest("POST", s.generateURL("api", "publish"), bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.stepsClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to make request")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read response body")
		}
		defer resp.Body.Close()

		log.Errorf("Received error: %d: %s", resp.StatusCode, string(respBody))
		return nil, errors.New("Did not receive status code OK")
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read response body")
	}
	defer resp.Body.Close()

	var createDraftResponse PublishStepResponse
	err = json.Unmarshal(respBody, &createDraftResponse)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal response body")
	}

	return &createDraftResponse, nil
}

// UploadTarball uploads the step tarball using the response from the
// CreateDraft endpoint.
func (s *RESTPublisher) UploadTarball(uploadURL string, body io.Reader, size int64) error {
	log.WithField("url", uploadURL).Debug("Uploading tarball to UploadUrl")

	req, err := http.NewRequest("PUT", uploadURL, body)
	if err != nil {
		return errors.Wrap(err, "Unable to create request")
	}
	req.ContentLength = size

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "Unable to make request")
	}

	log.Debugf("Received status code: %d", resp.StatusCode)
	if resp.StatusCode != 200 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		if len(respBody) > 0 {
			log.Errorf("Error body: %s", respBody)
		}

		return errors.New("Did not receive expected status code")
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	return nil
}

// FinishPublish post to the publish endpoint and the tarball is uploaded to
// indicate that the step can be published.
func (s *RESTPublisher) FinishPublish(token string) error {
	u := s.generateURL("api", "publish", "done")
	log.WithField("url", u).Debug("Finishing publication of step")

	payload := bytes.NewBuffer([]byte(fmt.Sprintf(`{"token": "%s"}`, token)))
	req, err := http.NewRequest("POST", u, payload)
	if err != nil {
		return errors.Wrap(err, "Unable to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.stepsClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Unable to make request")
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	log.Debugf("Received status code: %d", resp.StatusCode)

	if resp.StatusCode != 200 {
		return errors.New("Did not receive expected status code")
	}

	return nil
}
