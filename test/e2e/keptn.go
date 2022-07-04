package e2e

import (
	"encoding/base64"
	"fmt"

	"github.com/keptn/go-utils/pkg/api/models"
	api "github.com/keptn/go-utils/pkg/api/utils"
)

const authHeaderName = "x-token"
const jobResourceURI = "job/config.yaml"

// KeptnAPI structure holds different api handlers for the keptn api such that they can be used more easily
type KeptnAPI struct {
	APISet          *api.APISet
	APIHandler      api.APIV1Interface
	ProjectHandler  api.ProjectsV1Interface
	ResourceHandler api.ResourcesV1Interface
	EventHandler    api.EventsV1Interface
}

// NewKeptnAPI creates a KeptnAPI structure from KeptnConnectionDetails
func NewKeptnAPI(details KeptnConnectionDetails) (*KeptnAPI, error) {
	apiSet, err := api.New(details.Endpoint, api.WithAuthToken(details.APIToken, authHeaderName))
	if err != nil {
		return nil, fmt.Errorf("unable to create Keptn APISet: %w", err)
	}

	return &KeptnAPI{
		APISet:          apiSet,
		APIHandler:      apiSet.APIV1(),
		ProjectHandler:  apiSet.ProjectsV1(),
		ResourceHandler: apiSet.ResourcesV1(),
		EventHandler:    apiSet.EventsV1(),
	}, nil
}

// CreateProject creates a keptn project from the contents of a shipyard yaml file
func (k KeptnAPI) CreateProject(projectName string, shipyardYAML []byte) error {

	shipyardFileBase64 := base64.StdEncoding.EncodeToString(shipyardYAML)

	_, err := k.APIHandler.CreateProject(models.CreateProject{
		Name:     &projectName,
		Shipyard: &shipyardFileBase64,
	})

	if err != nil {
		return fmt.Errorf("unable to create project: %s", convertKeptnModelToErrorString(err))
	}

	return nil
}

// DeleteProject deletes a project by a given name
func (k KeptnAPI) DeleteProject(projectName string) error {
	_, err := k.APIHandler.DeleteProject(models.Project{
		ProjectName: projectName,
	})

	if err != nil {
		return fmt.Errorf("unable to delete project: %s", convertKeptnModelToErrorString(err))
	}

	return nil
}

// CreateService creates a service in a given project
func (k KeptnAPI) CreateService(projectName string, serviceName string) error {
	_, err := k.APIHandler.CreateService(projectName, models.CreateService{
		ServiceName: &serviceName,
	})

	if err != nil {
		return fmt.Errorf("unable to create service %s in project %s: %s", serviceName, projectName, convertKeptnModelToErrorString(err))
	}

	return nil
}

// CreateJobConfig uploads the job configuration for the job-executor-service to a specific service and stage
func (k KeptnAPI) CreateJobConfig(projectName string, stageName string, serviceName string, jobConfigYaml []byte) error {
	return k.AddServiceResource(projectName, stageName, serviceName, jobResourceURI, string(jobConfigYaml))
}

// AddServiceResource uploads a resource to a specific service and stage
func (k KeptnAPI) AddServiceResource(projectName string, stageName string, serviceName string, path string, data string) error {
	_, err := k.ResourceHandler.CreateResources(projectName, stageName, serviceName, []*models.Resource{
		{
			Metadata:        nil,
			ResourceContent: data,
			ResourceURI:     &path,
		},
	})

	if err != nil {
		return fmt.Errorf("unable to create service resource for service %s in project %s: %s",
			serviceName, projectName, convertKeptnModelToErrorString(err),
		)
	}

	return nil
}

// SendEvent sends an event to Keptn
func (k KeptnAPI) SendEvent(keptnEvent *models.KeptnContextExtendedCE) (*models.EventContext, error) {
	keptnContext, err := k.APIHandler.SendEvent(*keptnEvent)

	if err != nil {
		return nil, fmt.Errorf("unable to send event: %s", convertKeptnModelToErrorString(err))
	}

	return keptnContext, nil
}

// GetEvents returns a list of events for the given context from keptn
func (k KeptnAPI) GetEvents(keptnContext *string) ([]*models.KeptnContextExtendedCE, error) {
	eventFilter := api.EventFilter{
		KeptnContext: *keptnContext,
	}

	events, err := k.EventHandler.GetEvents(&eventFilter)
	if err != nil {
		return nil, fmt.Errorf("unable to get events: %s", convertKeptnModelToErrorString(err))
	}

	return events, nil
}
