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
)

// constants useful for this file
// Note:
// - if constants are used in multiple controllers, consider moving them to suite_test.go
// - use separate resource names to prevent collisions between tests

const (
	OrgName  = "test-org"
	Username = "testUser"
	Password = "testPass"
	Owner    = "testOwner"

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
			// Reset the cache so tests can be run independently
			orgClientCache = make(map[clientIdentifier]*organizationClient)
			// Reset server call counts
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
		})

		It("should be able to query some org twice", func() {
			orgClient, err := NewOrganizationClient(OrgName, url, Username, Password)
			Expect(err).To(BeNil())

			orgClient.GetSpace(ctx, Owner)
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
	})

	Describe("NewSpaceClient", func() {
		BeforeEach(func() {
			// Reset the cache so tests can be run independently
			spaceClientCache = make(map[clientIdentifier]*spaceClient)
			// Reset server call counts
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
			NewSpaceClient(OrgName, url, Username, Password)

			// Discover UAA endpoint
			Expect(server.ReceivedRequests()[0].Method).To(Equal("GET"))
			Expect(server.ReceivedRequests()[0].URL.Path).To(Equal("/"))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("should be able to query space", func() {
			spaceClient, err := NewSpaceClient(OrgName, url, Username, Password)
			Expect(err).To(BeNil())

			spaceClient.GetInstance(ctx, Owner)

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
		})

		It("should be able to query some space twice", func() {
			spaceClient, err := newSpaceClient(OrgName, url, Username, Password)
			Expect(err).To(BeNil())

			spaceClient.GetInstance(ctx, Owner)
			spaceClient.GetInstance(ctx, Owner)

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
	})
})
