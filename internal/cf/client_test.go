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

// is resource cache enabled and cache timeout
var resourceCacheEnabled = false
var resourceCacheTimeout = 5 * time.Minute

var cfg = &config.Config{
	IsResourceCacheEnabled: resourceCacheEnabled,
	CacheTimeOut:           resourceCacheTimeout.String(),
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
			server.RouteToHandler("GET", "/", ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &rootResult),
			))
			server.RouteToHandler("GET", spacesURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &rootResult),
			))
			server.RouteToHandler("POST", uaaURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &tokenResult),
			))
		})

		It("should create OrgClient", func() {
			NewOrganizationClient(OrgName, url, Username, Password)

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("should be able to query some org", func() {
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password)
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
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password)
			Expect(err).To(BeNil())

			orgClient.GetSpace(ctx, Owner)
			orgClient, err = NewOrganizationClient(OrgName, url, Username, Password)
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
			orgClient1, err1 := NewOrganizationClient(OrgName, url, Username, Password)
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
			orgClient2, err2 := NewOrganizationClient(OrgName2, url, Username, Password)
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

			// Register handlers
			server.RouteToHandler("GET", "/", ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &rootResult),
			))
			server.RouteToHandler("GET", serviceInstancesURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &rootResult),
			))
			server.RouteToHandler("POST", uaaURI, ghttp.CombineHandlers(
				ghttp.RespondWithJSONEncodedPtr(&statusCode, &tokenResult),
			))
		})

		It("should create SpaceClient", func() {
			NewSpaceClient(OrgName, url, Username, Password, *cfg)

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("should be able to query some space", func() {
			spaceClient, err := NewSpaceClient(OrgName, url, Username, Password, *cfg)
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
			spaceClient, err := NewSpaceClient(OrgName, url, Username, Password, *cfg)
			Expect(err).To(BeNil())

			spaceClient.GetInstance(ctx, map[string]string{"owner": Owner})
			spaceClient, err = NewSpaceClient(OrgName, url, Username, Password, *cfg)
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
			spaceClient1, err1 := NewSpaceClient(SpaceName, url, Username, Password, *cfg)
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
			spaceClient2, err2 := NewSpaceClient(SpaceName2, url, Username, Password, *cfg)
			Expect(err2).To(BeNil())
			spaceClient2.GetInstance(ctx, map[string]string{"owner": Owner2})
			// no discovery of UAA endpoint or oAuth token here due to caching
			// Get instance
			Expect(server.ReceivedRequests()[3].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[3].RequestURI).To(ContainSubstring(Owner2))
		})

		It("should register prometheus metrics for OrgClient", func() {
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password)
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
			spaceClient, err := NewSpaceClient(SpaceName, url, Username, Password, *cfg)
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

	})
})
