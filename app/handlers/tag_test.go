package handlers_test

import (
	"net/http"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
)

func TestCreateTagHandler_ValidRequests(t *testing.T) {
	RegisterT(t)

	server, services := mock.NewServer()
	status, _ := server.
		AsUser(mock.JonSnow).
		ExecutePost(
			handlers.CreateEditTag(),
			`{ "name": "Feature Request", "color": "00FF00", "isPublic": true }`,
		)

	Expect(status).Equals(http.StatusOK)

	tag, err := services.Tags.GetBySlug("feature-request")
	Expect(err).IsNil()
	Expect(tag.Name).Equals("Feature Request")
	Expect(tag.Slug).Equals("feature-request")
	Expect(tag.Color).Equals("00FF00")
	Expect(tag.IsPublic).IsTrue()
}

func TestCreateTagHandler_InvalidRequests(t *testing.T) {
	RegisterT(t)

	var testCases = []struct {
		input    string
		failures []string
	}{
		{`{ }`, []string{"failures.name", "failures.color"}},
		{`{ "name": "" }`, []string{"failures.name", "failures.color"}},
		{`{ "name": "Bug" }`, []string{"failures.color"}},
		{`{ "name": "Bug", "color": "ABC" }`, []string{"failures.color"}},
		{`{ "name": "Bug", "color": "00000X" }`, []string{"failures.color"}},
		{`{ "name": "123456789012345678901234567890A", "color": "000000" }`, []string{"failures.name"}},
	}

	for _, testCase := range testCases {
		server, _ := mock.NewServer()
		status, query := server.
			AsUser(mock.JonSnow).
			ExecutePostAsJSON(handlers.CreateEditTag(), testCase.input)

		Expect(status).Equals(http.StatusBadRequest)
		for _, failure := range testCase.failures {
			Expect(query.Contains(failure)).IsTrue()
		}
	}

}

func TestCreateTagHandler_AlreadyInUse(t *testing.T) {
	RegisterT(t)

	server, services := mock.NewServer()
	services.Tags.Add("Bug", "0000FF", true)

	status, query := server.
		AsUser(mock.JonSnow).
		ExecutePostAsJSON(
			handlers.CreateEditTag(),
			`{ "name": "Bug", "color": "0000FF", "isPublic": true }`,
		)

	Expect(status).Equals(http.StatusBadRequest)
	Expect(query.Contains("failures.name")).IsTrue()
}

func TestCreateTagHandler_Collaborator(t *testing.T) {
	RegisterT(t)

	server, _ := mock.NewServer()
	status, _ := server.
		AsUser(mock.AryaStark).
		ExecutePost(
			handlers.CreateEditTag(),
			`{ "name": "Feature Request", "color": "000000", "isPublic": true }`,
		)

	Expect(status).Equals(http.StatusForbidden)
}

func TestEditInvalidTagHandler(t *testing.T) {
	RegisterT(t)

	server, _ := mock.NewServer()
	status, _ := server.
		AsUser(mock.JonSnow).
		AddParam("slug", "bug").
		ExecutePost(
			handlers.CreateEditTag(),
			`{ "name": "Feature Request", "color": "000000", "isPublic": true }`,
		)

	Expect(status).Equals(http.StatusNotFound)
}

func TestEditExistingTagHandler(t *testing.T) {
	RegisterT(t)

	server, services := mock.NewServer()
	services.Tags.Add("Bug", "0000FF", true)

	status, _ := server.
		AsUser(mock.JonSnow).
		AddParam("slug", "bug").
		ExecutePost(
			handlers.CreateEditTag(),
			`{ "name": "Feature Request", "color": "000000", "isPublic": true }`,
		)

	Expect(status).Equals(http.StatusOK)
	tag, err := services.Tags.GetBySlug("bug")
	Expect(tag).IsNil()
	Expect(errors.Cause(err)).Equals(app.ErrNotFound)

	tag, err = services.Tags.GetBySlug("feature-request")
	Expect(tag).IsNotNil()
	Expect(err).IsNil()
}

func TestDeleteInvalidTagHandler(t *testing.T) {
	RegisterT(t)

	server, _ := mock.NewServer()
	status, _ := server.
		AsUser(mock.JonSnow).
		AddParam("slug", "bug").
		Execute(handlers.DeleteTag())

	Expect(status).Equals(http.StatusNotFound)
}

func TestDeleteExistingTagHandler(t *testing.T) {
	RegisterT(t)

	server, services := mock.NewServer()
	services.Tags.Add("Bug", "0000FF", true)

	status, _ := server.
		AsUser(mock.JonSnow).
		AddParam("slug", "bug").
		Execute(handlers.DeleteTag())

	tag, err := services.Tags.GetBySlug("bug")
	Expect(status).Equals(http.StatusOK)
	Expect(errors.Cause(err)).Equals(app.ErrNotFound)
	Expect(tag).IsNil()
}

func TestDeleteExistingTagHandler_Collaborator(t *testing.T) {
	RegisterT(t)

	server, services := mock.NewServer()
	services.Tags.Add("Bug", "0000FF", true)

	status, _ := server.
		AsUser(mock.AryaStark).
		AddParam("slug", "bug").
		Execute(handlers.DeleteTag())

	tag, err := services.Tags.GetBySlug("bug")
	Expect(status).Equals(http.StatusForbidden)
	Expect(tag).IsNotNil()
	Expect(err).IsNil()

}

func TestAssignTagHandler_Success(t *testing.T) {
	RegisterT(t)

	server, services := mock.NewServer()
	services.SetCurrentTenant(mock.DemoTenant)
	services.SetCurrentUser(mock.JonSnow)
	tag, _ := services.Tags.Add("Bug", "0000FF", true)
	idea, _ := services.Ideas.Add("Idea Title", "Idea Description")

	status, _ := server.
		AsUser(mock.JonSnow).
		AddParam("slug", tag.Slug).
		AddParam("number", idea.Number).
		Execute(handlers.AssignTag())

	tags, err := services.Tags.GetAssigned(idea)
	Expect(status).Equals(http.StatusOK)
	Expect(err).IsNil()
	Expect(tags[0]).Equals(tag)
}

func TestAssignTagHandler_UnknownTag(t *testing.T) {
	RegisterT(t)

	server, _ := mock.NewServer()

	status, _ := server.
		AsUser(mock.JonSnow).
		AddParam("slug", "bug").
		AddParam("number", 1).
		Execute(handlers.AssignTag())

	Expect(status).Equals(http.StatusNotFound)
}

func TestAssignOrUnassignTagHandler_Unauthorized(t *testing.T) {
	RegisterT(t)

	var testCases = []web.HandlerFunc{
		handlers.AssignTag(),
		handlers.UnassignTag(),
	}

	for _, handler := range testCases {
		server, services := mock.NewServer()
		services.SetCurrentTenant(mock.DemoTenant)
		services.SetCurrentUser(mock.JonSnow)
		tag, _ := services.Tags.Add("Bug", "0000FF", true)
		idea, _ := services.Ideas.Add("Idea Title", "Idea Description")

		status, _ := server.
			AsUser(mock.AryaStark).
			AddParam("slug", tag.Slug).
			AddParam("number", idea.Number).
			Execute(handler)

		Expect(status).Equals(http.StatusForbidden)
	}
}

func TestUnassignTagHandler_Success(t *testing.T) {
	RegisterT(t)

	server, services := mock.NewServer()
	services.SetCurrentTenant(mock.DemoTenant)
	services.SetCurrentUser(mock.JonSnow)
	tag, _ := services.Tags.Add("Bug", "0000FF", true)
	idea, _ := services.Ideas.Add("Idea Title", "Idea Description")
	services.Tags.AssignTag(tag, idea)

	status, _ := server.
		AsUser(mock.JonSnow).
		AddParam("slug", tag.Slug).
		AddParam("number", idea.Number).
		Execute(handlers.UnassignTag())

	tags, err := services.Tags.GetAssigned(idea)
	Expect(status).Equals(http.StatusOK)
	Expect(err).IsNil()
	Expect(tags).HasLen(0)
}
