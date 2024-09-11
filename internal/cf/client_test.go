/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/
package cf

import (
	"context"
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

		// Context("Populate resource cache tests", func() {

		// 	It("should initialize resource cache after start and on cache expiry", func() {
		// 		// Enable resource cache in config
		// 		clientConfig.IsResourceCacheEnabled = true
		// 		clientConfig.CacheTimeOut = "5s" // short duration for fast test

		// 		// Create client
		// 		spaceClient, err := NewSpaceClient(SpaceName, url, Username, Password, *clientConfig)
		// 		Expect(err).To(BeNil())
		// 		Expect(spaceClient).ToNot(BeNil())

		// 		// Verify resource cache is populated during client creation
		// 		Expect(server.ReceivedRequests()).To(HaveLen(3))
		// 		// - Discover UAA endpoint
		// 		Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
		// 		// - Get new oAuth token
		// 		Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
		// 		Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
		// 		// - Populate cache
		// 		Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(serviceInstancesURI))

		// 		// Make a request and verify that cache is used and no additional requests expected
		// 		spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
		// 		Expect(server.ReceivedRequests()).To(HaveLen(3)) // still same as above

		// 		// Make another request after cache expired and verify that cache is repopulated
		// 		time.Sleep(10 * time.Second)
		// 		spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
		// 		Expect(server.ReceivedRequests()).To(HaveLen(4)) // one more request to repopulate cache
		// 		Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[3].RequestURI).To(ContainSubstring(serviceInstancesURI))
		// 		Expect(server.ReceivedRequests()[3].RequestURI).NotTo(ContainSubstring(Owner))
		// 	})

		// 	It("should not initialize resource cache if disabled in config", func() {
		// 		// Disable resource cache in config
		// 		clientConfig.IsResourceCacheEnabled = false

		// 		// Create client
		// 		spaceClient, err := NewSpaceClient(SpaceName, url, Username, Password, *clientConfig)
		// 		Expect(err).To(BeNil())
		// 		Expect(spaceClient).ToNot(BeNil())

		// 		// Verify resource cache is NOT populated during client creation
		// 		// - Discover UAA endpoint
		// 		Expect(server.ReceivedRequests()).To(HaveLen(1))
		// 		Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

		// 		// Make request and verify cache is NOT used
		// 		spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
		// 		Expect(server.ReceivedRequests()).To(HaveLen(3)) // one more request to get instance
		// 		// - Get new oAuth token
		// 		Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
		// 		Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
		// 		// - Get instance
		// 		Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(Owner))
		// 	})

		// 	It("Delete instance from cache", func() {
		// 		// Enable resource cache in config
		// 		clientConfig.IsResourceCacheEnabled = true
		// 		clientConfig.CacheTimeOut = "5m"

		// 		// Route to handler for DELETE request
		// 		server.RouteToHandler("DELETE",
		// 			serviceInstancesURI+"/test-instance-guid-1",
		// 			ghttp.CombineHandlers(ghttp.RespondWith(http.StatusAccepted, nil)))

		// 		// Create client
		// 		spaceClient, err := NewSpaceClient(SpaceName, url, Username, Password, *clientConfig)
		// 		Expect(err).To(BeNil())
		// 		Expect(spaceClient).ToNot(BeNil())

		// 		// Verify resource cache is populated during client creation
		// 		Expect(server.ReceivedRequests()).To(HaveLen(3))
		// 		// - Discover UAA endpoint
		// 		Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))
		// 		// - Get new oAuth token
		// 		Expect(server.ReceivedRequests()[1].Method).To(Equal("POST"))
		// 		Expect(server.ReceivedRequests()[1].URL.Path).To(Equal(uaaURI))
		// 		// - Populate cache
		// 		Expect(server.ReceivedRequests()[2].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[2].RequestURI).To(ContainSubstring(serviceInstancesURI))
		// 		Expect(server.ReceivedRequests()[2].RequestURI).NotTo(ContainSubstring(Owner))

		// 		// Make a request and verify that cache is used and no additional requests expected
		// 		spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
		// 		Expect(server.ReceivedRequests()).To(HaveLen(3)) // still same as above

		// 		// Delete instance from cache
		// 		err = spaceClient.DeleteInstance(ctx, "test-instance-guid-1", Owner)
		// 		Expect(err).To(BeNil())
		// 		Expect(server.ReceivedRequests()).To(HaveLen(4))
		// 		// - Delete instance from cache
		// 		Expect(server.ReceivedRequests()[3].Method).To(Equal("DELETE"))
		// 		Expect(server.ReceivedRequests()[3].RequestURI).To(ContainSubstring("test-instance-guid-1"))

		// 		// Get instance from cache should return empty
		// 		spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
		// 		Expect(server.ReceivedRequests()).To(HaveLen(5))
		// 		// - get instance from cache
		// 		Expect(server.ReceivedRequests()[4].Method).To(Equal("GET"))
		// 		Expect(server.ReceivedRequests()[4].RequestURI).To(ContainSubstring(Owner))
		// 	})
		// })
	})
})
