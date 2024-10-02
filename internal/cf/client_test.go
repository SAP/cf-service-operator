/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package cf

import (
	"context"
	"net/http"
	"testing"
	"time"

	cfResource "github.com/cloudfoundry-community/go-cfclient/v3/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sap/cf-service-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// constants useful for this file
// Note:
// - if constants are used in multiple controllers, consider moving them to suite_test.go
// - use separate resource names to prevent collisions between tests

const (
	OrgName    = "test-org"
	OrgName2   = "test-org-2"
	SpaceName  = "test-space"
	SpaceName2 = "test-space-2"
	Username   = "testUser"
	Password   = "testPass"
	Owner      = "testOwner"
	Owner2     = "testOwner2"

	spacesURI           = "/v3/spaces"
	serviceInstancesURI = "/v3/service_instances"
	spaceURI            = "/v3/spaces"
	serviceBindingURI   = "/v3/service_credential_bindings"
	userURI             = "v3/users"
	roleURI             = "v3/roles"
	uaaURI              = "/uaa/oauth/token"
	labelSelector       = "service-operator.cf.cs.sap.com"
)

type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

func TestCFClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CF Client Test Suite")
}

func String(s string) *string {
	return &s
}

// configuration for CF client
var clientConfig = &config.Config{
	IsResourceCacheEnabled: false,
	CacheTimeOut:           "5m",
}

// fake response for service instances
var fakeServiceInstances = cfResource.ServiceInstanceList{
	Resources: []*cfResource.ServiceInstance{
		{
			GUID: "test-instance-guid-1",
			Name: "test-instance-name-1",
			Tags: []string{},
			LastOperation: cfResource.LastOperation{
				Type:        "create",
				State:       "succeeded",
				Description: "",
			},
			Relationships: cfResource.ServiceInstanceRelationships{
				ServicePlan: &cfResource.ToOneRelationship{
					Data: &cfResource.Relationship{
						GUID: "test-instance-service_plan-1",
					},
				},
			},
			Metadata: &cfResource.Metadata{
				Labels: map[string]*string{
					"service-operator.cf.cs.sap.com/owner": String("testOwner"),
				},
				Annotations: map[string]*string{
					"service-operator.cf.cs.sap.com/generation":     String("1"),
					"service-operator.cf.cs.sap.com/parameter-hash": String("74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"),
				},
			},
		},
	},
}

var fakeServiceBindings = cfResource.ServiceCredentialBindingList{
	Resources: []*cfResource.ServiceCredentialBinding{
		{
			GUID: "test-binding-guid-1",
			Name: "test-binding-name-1",
			LastOperation: cfResource.LastOperation{
				Type:        "create",
				State:       "succeeded",
				Description: "",
			},
			Metadata: &cfResource.Metadata{
				Labels: map[string]*string{
					"service-operator.cf.cs.sap.com/owner": String("testOwner"),
				},
				Annotations: map[string]*string{
					"service-operator.cf.cs.sap.com/generation":     String("1"),
					"service-operator.cf.cs.sap.com/parameter-hash": String("74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"),
				},
			},
		},
	},
}

var fakeBingdingDetails = cfResource.ServiceCredentialBindingDetails{
	Credentials: map[string]interface{}{
		"key": "value",
	},
}

var fakeSpaces = cfResource.SpaceList{
	Resources: []*cfResource.Space{
		{
			GUID: "test-space-guid-1",
			Name: "test-space-name-1",
			Metadata: &cfResource.Metadata{
				Labels: map[string]*string{
					"service-operator.cf.cs.sap.com/owner": String("testOwner"),
				},
				Annotations: map[string]*string{
					"service-operator.cf.cs.sap.com/generation":     String("1"),
					"service-operator.cf.cs.sap.com/parameter-hash": String("74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"),
				},
			},
		},
	},
}

var fakeUsers = cfResource.UserList{
	Resources: []*cfResource.User{
		{
			GUID:     "test-user-guid-1",
			Username: "testUser",
		},
	},
}

var fakeRoles = cfResource.RoleList{
	Resources: []*cfResource.Role{
		{
			GUID: "test-role-guid-1",
			Type: "test-role-type-1",
			Relationships: cfResource.RoleSpaceUserOrganizationRelationships{
				User: cfResource.ToOneRelationship{
					Data: &cfResource.Relationship{
						GUID: "test-user-guid-1",
					},
				},
				Space: cfResource.ToOneRelationship{
					Data: &cfResource.Relationship{
						GUID: "test-space-guid-1",
					},
				},
			},
		},
	},
}

// -----------------------------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------------------------

var _ = Describe("CF Client tests", Ordered, func() {
	var server *ghttp.Server
	var url string
	var rootResult cfResource.Root
	var statusCode int
	var ctx context.Context
	var tokenResult Token

	BeforeAll(func() {
		ctx = context.Background()
		// Setup fake server
		server = ghttp.NewServer()
		url = "http://" + server.Addr()
		statusCode = 200
		rootResult = cfResource.Root{
			Links: cfResource.RootLinks{
				Uaa: cfResource.Link{
					Href: url + "/uaa",
				},
				Login: cfResource.Link{
					Href: url + "/login",
				},
			},
		}
		tokenResult = Token{
			AccessToken:  "Foo",
			TokenType:    "Bar",
			RefreshToken: "Baz",
			Expiry:       time.Now().Add(time.Minute),
		}
		By("creating space CR")
	})
	AfterAll(func() {
		// Shutdown the server after tests
		server.Close()
	})

	Describe("NewOrganizationClient", func() {
		BeforeEach(func() {
			// Reset some entities to enable tests to run independently
			clientCache = make(map[clientIdentifier]*clientCacheEntry)
			metrics.Registry = prometheus.NewRegistry()
			server.Reset()

			// Create a new configuration for each test
			clientConfig = &config.Config{
				IsResourceCacheEnabled: false,
				CacheTimeOut:           "5m",
			}

			// Register handlers
			// - Fake response for discover UAA endpoint
			server.RouteToHandler("GET", "/", ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &rootResult),
			))
			// - Fake response for new oAuth token
			server.RouteToHandler("POST", uaaURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &tokenResult),
			))
			// - Fake response for get service instance
			server.RouteToHandler("GET", spacesURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &rootResult),
			))
			// - Fake response for get service instance
			server.RouteToHandler("GET", spaceURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &fakeSpaces),
			))
			// - Fake response for get users
			server.RouteToHandler("GET", "/v3/users", ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &fakeUsers),
			))
			// - Fake response for get roles
			server.RouteToHandler("GET", "/v3/roles", ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &fakeRoles),
			))

		})

		It("should create OrgClient", func() {
			NewOrganizationClient(OrgName, url, Username, Password, clientConfig)

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("should be able to query some org", func() {
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())

			orgClient.GetSpace(ctx, Owner)

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// Get space
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].URL.Path).To(Equal(spacesURI))

			Expect(server.ReceivedRequests()).To(HaveLen(3))

			// verify metrics
			metricsList, err := metrics.Registry.Gather()
			Expect(err).To(BeNil())
			Expect(metricsList).To(HaveLen(3))
			verified := false
			for _, m := range metricsList {
				if *m.Name == "http_client_requests_total" {
					Expect(m.Metric[0].Counter.GetValue()).To(BeNumerically(">", 0))
					verified = true
				}
			}
			Expect(verified).To(BeTrue())
		})

		It("should be able to query some org twice", func() {
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())

			orgClient.GetSpace(ctx, Owner)
			orgClient, err = NewOrganizationClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			orgClient.GetSpace(ctx, Owner)

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// Get space
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].URL.Path).To(Equal(spacesURI))

			// Get space
			Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[3].URL.Path).To(Equal(spacesURI))

			Expect(server.ReceivedRequests()).To(HaveLen(4))
		})

		It("should be able to query two different orgs", func() {
			// test org 1
			orgClient1, err1 := NewOrganizationClient(OrgName, url, Username, Password, clientConfig)
			Expect(err1).To(BeNil())
			orgClient1.GetSpace(ctx, Owner)
			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// Get instance
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(Owner))

			// test org 2
			orgClient2, err2 := NewOrganizationClient(OrgName2, url, Username, Password, clientConfig)
			Expect(err2).To(BeNil())
			orgClient2.GetSpace(ctx, Owner2)
			// no discovery of UAA endpoint or oAuth token here due to caching
			// Get instance
			Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[3].RequestURI).To(ContainSubstring(Owner2))
		})

		It("should not initialize resource cache if disabled in config", func() {
			// Disable resource cache in config
			clientConfig.IsResourceCacheEnabled = false

			// Create client
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			Expect(orgClient).ToNot(BeNil())

			// Verify resource cache is NOT populated during client creation
			// - Discover UAA endpoint
			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

			// Make request and verify cache is NOT used
			orgClient.GetSpace(ctx, Owner)
			Expect(server.ReceivedRequests()).To(HaveLen(3)) // one more request to get instance
			// - Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// - Get space
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(Owner))
		})

		It("should initialize/manage resource cache after start and on cache expiry", func() {
			// Enable resource cache in config
			clientConfig.IsResourceCacheEnabled = true
			clientConfig.CacheTimeOut = "5s" // short duration for fast test

			// 	// Route to handler for DELETE request
			server.RouteToHandler("DELETE",
				spaceURI+"/test-space-guid-1",
				ghttp.CombineHandlers(ghttp.RespondWith(http.StatusAccepted, nil)))

			// Create client
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			Expect(orgClient).ToNot(BeNil())

			// Verify resource cache is populated during client creation
			Expect(server.ReceivedRequests()).To(HaveLen(6))
			// - Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// - Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// - Populate cache with spaces
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(spaceURI))
			//- Populate cache with spaces,user and role
			Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[3].RequestURI).To(ContainSubstring(spaceURI))
			Expect(server.ReceivedRequests()[4].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[4].RequestURI).To(ContainSubstring(userURI))
			Expect(server.ReceivedRequests()[5].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[5].RequestURI).To(ContainSubstring(roleURI))

			// Make a request and verify that cache is used and no additional requests expected
			orgClient.GetSpace(ctx, Owner)
			Expect(server.ReceivedRequests()).To(HaveLen(6)) // still same as above

			// Make another request after cache expired and verify that cache is repopulated
			time.Sleep(10 * time.Second)
			orgClient.GetSpace(ctx, Owner)
			Expect(server.ReceivedRequests()).To(HaveLen(7)) // one more request to repopulate cache
			Expect(server.ReceivedRequests()[6].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[6].RequestURI).To(ContainSubstring(spaceURI))
			Expect(server.ReceivedRequests()[6].RequestURI).NotTo(ContainSubstring(Owner))

			// Delete space from cache
			err = orgClient.DeleteSpace(ctx, "test-space-guid-1", Owner)
			Expect(err).To(BeNil())
			Expect(server.ReceivedRequests()).To(HaveLen(8))
			// - Delete space from cache
			Expect(server.ReceivedRequests()[7].Method).To(Equal("DELETE"))
			Expect(server.ReceivedRequests()[7].RequestURI).To(ContainSubstring("test-space-guid-1"))

			// Get space from cache should return empty
			orgClient.GetSpace(ctx, Owner)
			Expect(server.ReceivedRequests()).To(HaveLen(9))
			// - Get call to cf to get the space
			Expect(server.ReceivedRequests()[8].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[8].RequestURI).To(ContainSubstring(Owner))
		})

	})

	Describe("NewSpaceClient", func() {
		BeforeEach(func() {
			// Reset some entities to enable tests to run independently
			clientCache = make(map[clientIdentifier]*clientCacheEntry)
			metrics.Registry = prometheus.NewRegistry()
			server.Reset()

			// Create a new configuration for each test
			clientConfig = &config.Config{
				IsResourceCacheEnabled: false,
				CacheTimeOut:           "5m",
			}

			// Register handlers
			// - Fake response for discover UAA endpoint
			server.RouteToHandler("GET", "/", ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &rootResult),
			))
			// - Fake response for new oAuth token
			server.RouteToHandler("POST", uaaURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &tokenResult),
			))
			// - Fake response for get service instance
			server.RouteToHandler("GET", serviceInstancesURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &fakeServiceInstances),
			))

			// - Fake response for get service binding
			server.RouteToHandler("GET", serviceBindingURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &fakeServiceBindings),
			))

			// - Fake response for get service binding
			server.RouteToHandler("GET", serviceBindingURI+"/"+fakeServiceBindings.Resources[0].GUID+"/details", ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &fakeBingdingDetails),
			))
		})

		It("should create SpaceClient", func() {
			NewSpaceClient(OrgName, url, Username, Password, clientConfig)

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("should be able to query some space", func() {
			spaceClient, err := NewSpaceClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())

			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// Get instance
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].URL.Path).To(Equal(serviceInstancesURI))

			Expect(server.ReceivedRequests()).To(HaveLen(3))

			// verify metrics
			metricsList, err := metrics.Registry.Gather()
			Expect(err).To(BeNil())
			Expect(metricsList).To(HaveLen(3))
			verified := false
			for _, m := range metricsList {
				if *m.Name == "http_client_requests_total" {
					Expect(m.Metric[0].Counter.GetValue()).To(BeNumerically(">", 0))
					verified = true
				}
			}
			Expect(verified).To(BeTrue())
		})

		It("should be able to query some space twice", func() {
			spaceClient, err := NewSpaceClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())

			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
			spaceClient, err = NewSpaceClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// Get instance
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].URL.Path).To(Equal(serviceInstancesURI))

			// Get instance
			Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[3].URL.Path).To(Equal(serviceInstancesURI))

			Expect(server.ReceivedRequests()).To(HaveLen(4))
		})

		It("should be able to query two different spaces", func() {
			// test space 1
			spaceClient1, err1 := NewSpaceClient(SpaceName, url, Username, Password, clientConfig)
			Expect(err1).To(BeNil())
			spaceClient1.GetInstance(ctx, map[string]string{"owner": Owner})
			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// Get instance
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(Owner))

			// test space 2
			spaceClient2, err2 := NewSpaceClient(SpaceName2, url, Username, Password, clientConfig)
			Expect(err2).To(BeNil())
			spaceClient2.GetInstance(ctx, map[string]string{"owner": Owner2})
			// no discovery of UAA endpoint or oAuth token here due to caching
			// Get instance
			Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[3].RequestURI).To(ContainSubstring(Owner2))
		})

		It("should register prometheus metrics for OrgClient", func() {
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			Expect(orgClient).ToNot(BeNil())

			// retrieve names of registered metrics
			metricsList, err := metrics.Registry.Gather()
			Expect(err).To(BeNil())
			Expect(metricsList).To(HaveLen(3))
			metricNames := make([]string, len(metricsList))
			for i, m := range metricsList {
				metricNames[i] = *m.Name
			}

			Expect(metricNames).To(ContainElement("http_client_request_duration_seconds"))
			Expect(metricNames).To(ContainElement("http_client_requests_in_flight"))
			Expect(metricNames).To(ContainElement("http_client_requests_total"))
		})

		It("should register prometheus metrics for SpaceClient", func() {
			spaceClient, err := NewSpaceClient(SpaceName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			Expect(spaceClient).ToNot(BeNil())

			// retrieve names of registered metrics
			metricsList, err := metrics.Registry.Gather()
			Expect(err).To(BeNil())
			Expect(metricsList).To(HaveLen(3))
			metricNames := make([]string, len(metricsList))
			for i, m := range metricsList {
				metricNames[i] = *m.Name
			}

			Expect(metricNames).To(ContainElement("http_client_request_duration_seconds"))
			Expect(metricNames).To(ContainElement("http_client_requests_in_flight"))
			Expect(metricNames).To(ContainElement("http_client_requests_total"))

			// for debugging: write metrics to file
			// prometheus.WriteToTextfile("metrics.txt", metrics.Registry)
		})

		It("should not initialize resource cache if disabled in config", func() {
			// Disable resource cache in config
			clientConfig.IsResourceCacheEnabled = false

			// Create client
			spaceClient, err := NewSpaceClient(SpaceName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			Expect(spaceClient).ToNot(BeNil())

			// Verify resource cache is NOT populated during client creation
			// - Discover UAA endpoint
			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

			// Make request and verify cache is NOT used
			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
			Expect(server.ReceivedRequests()).To(HaveLen(3)) // one more request to get instance
			// - Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// - Get instance
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(Owner))
		})

		It("should initialize resource cache after start and on cache expiry", func() {
			// Enable resource cache in config
			clientConfig.IsResourceCacheEnabled = true
			clientConfig.CacheTimeOut = "5s" // short duration for fast test

			//resource cache will be initialized only only once, so we need to wait till the cache expiry from previous test
			time.Sleep(10 * time.Second)

			// Route to handler for DELETE request
			server.RouteToHandler("DELETE",
				serviceInstancesURI+"/test-instance-guid-1",
				ghttp.CombineHandlers(ghttp.RespondWith(http.StatusAccepted, nil)))

			// Route to handler for DELETE request
			server.RouteToHandler("DELETE",
				serviceBindingURI+"/test-binding-guid-1",
				ghttp.CombineHandlers(ghttp.RespondWith(http.StatusAccepted, nil)))

			// Create client
			spaceClient, err := NewSpaceClient(SpaceName, url, Username, Password, clientConfig)
			Expect(err).To(BeNil())
			Expect(spaceClient).ToNot(BeNil())

			// Verify resource cache is populated during client creation
			const numRequests = 4
			Expect(server.ReceivedRequests()).To(HaveLen(numRequests))
			// - Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
			// - Get new oAuth token
			Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
			Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
			// - Populate cache with instances
			Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(serviceInstancesURI))
			// - Populate cache with bindings
			Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[3].RequestURI).To(ContainSubstring(serviceBindingURI))

			// Make a request and verify that cache is used and no additional requests expected
			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
			Expect(server.ReceivedRequests()).To(HaveLen(numRequests)) // still same as above

			// Make another request after cache expired and verify that cache is repopulated
			time.Sleep(10 * time.Second)
			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
			spaceClient.GetBinding(ctx, map[string]string{"owner": Owner})
			Expect(server.ReceivedRequests()).To(HaveLen(6)) // one more request to repopulate cache
			Expect(server.ReceivedRequests()[4].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[4].RequestURI).To(ContainSubstring(serviceInstancesURI))
			Expect(server.ReceivedRequests()[4].RequestURI).NotTo(ContainSubstring(Owner))
			Expect(server.ReceivedRequests()[5].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[5].RequestURI).To(ContainSubstring(serviceBindingURI))
			Expect(server.ReceivedRequests()[5].RequestURI).NotTo(ContainSubstring(Owner))

			// Delete instance from cache
			err = spaceClient.DeleteInstance(ctx, "test-instance-guid-1", Owner)
			Expect(err).To(BeNil())
			Expect(server.ReceivedRequests()).To(HaveLen(7))
			// - Delete instance from cache
			Expect(server.ReceivedRequests()[6].Method).To(Equal("DELETE"))
			Expect(server.ReceivedRequests()[6].RequestURI).To(ContainSubstring("test-instance-guid-1"))

			// Get instance from cache should return empty
			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
			Expect(server.ReceivedRequests()).To(HaveLen(8))
			// - Get call to cf to get the instance
			Expect(server.ReceivedRequests()[7].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[7].RequestURI).To(ContainSubstring(Owner))

			// Delete binding from cache
			err = spaceClient.DeleteBinding(ctx, "test-binding-guid-1", Owner)
			Expect(err).To(BeNil())
			Expect(server.ReceivedRequests()).To(HaveLen(9))
			// - Delete binding from cache
			Expect(server.ReceivedRequests()[8].Method).To(Equal("DELETE"))
			Expect(server.ReceivedRequests()[8].RequestURI).To(ContainSubstring("test-binding-guid-1"))

			// Get binding from cache should return empty
			spaceClient.GetBinding(ctx, map[string]string{"owner": Owner})
			Expect(server.ReceivedRequests()).To(HaveLen(10))
			// - Get call to cf to get the binding
			Expect(server.ReceivedRequests()[9].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[9].RequestURI).To(ContainSubstring(Owner))
		})

	})
})
