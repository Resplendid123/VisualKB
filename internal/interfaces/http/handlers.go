package http

import (
	authapp "learn/internal/application/auth"
	convapp "learn/internal/application/conversation"
	documentapp "learn/internal/application/document"
	projectapp "learn/internal/application/project"
	userapp "learn/internal/application/user"

	"learn/internal/interfaces/http/auth"
	"learn/internal/interfaces/http/conversation"
	"learn/internal/interfaces/http/document"
	"learn/internal/interfaces/http/project"
	"learn/internal/interfaces/http/user"
)

// Handlers aggregates all HTTP handlers.
type Handlers struct {
	User         *user.UserHandler
	Conversation *conversation.ConversationHandler
	Auth         *auth.AuthHandler
	Project      *project.ProjectHandler
	Document     *document.DocumentHandler
	Tree         *document.TreeHandler
}

func NewHandlers(
	userSvc *userapp.UserService,
	convoSvc *convapp.ConversationService,
	authSvc *authapp.AuthService,
	projectSvc *projectapp.ProjectService,
	docSvc *documentapp.DocumentService,
	treeSvc *documentapp.TreeService,
	cookie auth.AuthCookie,
) *Handlers {
	return &Handlers{
		User:         user.NewUserHandler(userSvc),
		Conversation: conversation.NewConversationHandler(convoSvc),
		Auth:         auth.NewAuthHandler(authSvc, cookie),
		Project:      project.NewProjectHandler(projectSvc),
		Document:     document.NewDocumentHandler(docSvc),
		Tree:         document.NewTreeHandler(treeSvc),
	}
}
