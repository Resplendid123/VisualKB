package domain

import "errors"

// Error types map to HTTP status codes.
const (
	TypeInvalidArg = "invalid_argument"
	TypeNotFound   = "not_found"
	TypeConflict   = "conflict"
	TypeInternal   = "internal"
)

type DomainError struct {
	Type    string
	Code    int
	Message string
}

func (e *DomainError) Error() string { return e.Message }

func New(typ string, code int, msg string) *DomainError {
	return &DomainError{Type: typ, Code: code, Message: msg}
}

func AsDomainError(err error) (*DomainError, bool) {
	if err == nil {
		return nil, false
	}
	if de, ok := errors.AsType[*DomainError](err); ok {
		return de, true
	}
	return nil, false
}

var (
	ErrUserNotFound       = New(TypeNotFound, 1001, "user not found")
	ErrInvalidName        = New(TypeInvalidArg, 1002, "invalid name")
	ErrInvalidEmail       = New(TypeInvalidArg, 1003, "invalid email")
	ErrUserCreateFailed   = New(TypeInternal, 1004, "user create failed")
	ErrUserFetchFailed    = New(TypeInternal, 1005, "user fetch failed")
	ErrInvalidCredentials = New(TypeInvalidArg, 1006, "invalid credentials")
	ErrEmailTaken         = New(TypeConflict, 1007, "email already taken")
	ErrTokenRefreshFailed = New(TypeInvalidArg, 1008, "token refresh failed")
	ErrTokenInvalid       = New(TypeInvalidArg, 1009, "token invalid or user no longer exists")
	ErrPortraitTooLong    = New(TypeInvalidArg, 1010, "portrait field exceeds 4000 characters")
)

var (
	ErrInvalidConvoTitle   = New(TypeInvalidArg, 2001, "invalid conversation title")
	ErrConvoNotFound       = New(TypeNotFound, 2002, "conversation not found")
	ErrInvalidConvoContent = New(TypeInvalidArg, 2003, "empty content")
	ErrConcurrentTurn      = New(TypeConflict, 2004, "another turn is already running for this conversation")
)

var (
	ErrExecNotFound = New(TypeNotFound, 3101, "execution not found")
	ErrExecDenied   = New(TypeInvalidArg, 3102, "command denied by sandbox policy")
	ErrExecTimeout  = New(TypeInternal, 3103, "execution timed out")
)

var (
	ErrProjectNotFound    = New(TypeNotFound, 3201, "project not found")
	ErrInvalidProjectName = New(TypeInvalidArg, 3202, "invalid project name (must match ^[a-z0-9][a-z0-9-]{0,62}$)")
	ErrNoActiveProject    = New(TypeInvalidArg, 3203, "no active project: please call create_project(name) to create one first, then retry this bash command")
)

var (
	ErrSandboxNotFound     = New(TypeNotFound, 3001, "sandbox not found")
	ErrSandboxCreateFail   = New(TypeInternal, 3002, "sandbox create failed")
	ErrSandboxAlreadyUp    = New(TypeConflict, 3003, "sandbox already running")
	ErrSandboxNotRunning   = New(TypeInvalidArg, 3004, "sandbox not running")
	ErrSandboxPathInvalid  = New(TypeInvalidArg, 3005, "path must be under /workspace/project/")
	ErrSandboxFileNotFound = New(TypeNotFound, 3006, "file not found in active project")
)

var (
	ErrDocumentNotFound           = New(TypeNotFound, 4001, "document not found")
	ErrDocumentVersionNotFound    = New(TypeNotFound, 4002, "document version not found")
	ErrDocumentFetchFailed        = New(TypeInternal, 4003, "document fetch failed")
	ErrDocumentIngestFailed       = New(TypeInternal, 4004, "document ingest failed")
	ErrDocumentSplitFailed        = New(TypeInternal, 4005, "document split failed")
	ErrDocumentEmbedFailed        = New(TypeInternal, 4006, "document embed failed")
	ErrDocumentForbidden          = New(TypeInvalidArg, 4007, "you do not own this document")
	ErrDocumentInvalidSource      = New(TypeInvalidArg, 4008, "document source must be 'note' or 'knowledge'")
	ErrDocumentEmptyTitle         = New(TypeInvalidArg, 4009, "document title must not be empty")
	ErrDocumentUploadFailed       = New(TypeInternal, 4010, "document upload to object store failed")
	ErrDocumentInvalidContentType = New(TypeInvalidArg, 4011, "document content_type is not in the supported set; allowed: markdown, pdf")
	ErrDocumentNoteMarkdownOnly   = New(TypeInvalidArg, 4012, "notes are restricted to content_type='markdown'")
	ErrDocumentNotEditable        = New(TypeInvalidArg, 4013, "knowledge documents are read-only; create a new document with POST /documents instead")
	ErrDocumentEmptyContent       = New(TypeInvalidArg, 4014, "document content must not be empty")
	ErrDocumentAnchorNotFound     = New(TypeInvalidArg, 4015, "edit anchor was not found in the current document content")
	ErrDocumentAnchorAmbiguous    = New(TypeInvalidArg, 4016, "edit anchor matches more than one location; provide a longer unique anchor or pass replace_all=true")
	ErrDocumentUnknownEditOp      = New(TypeInvalidArg, 4017, "unknown edit op type; allowed: replace_anchor, append, whole_replace")
)

var (
	ErrTreeNodeNotFound    = New(TypeNotFound, 4101, "knowledge tree node not found")
	ErrTreeNodeInvalidName = New(TypeInvalidArg, 4102, "tree node name must match ^[a-z0-9_-]+$ (1-64 chars)")
	ErrTreeNodeNameTaken   = New(TypeConflict, 4103, "a node with this name already exists under the same parent")
	ErrTreeNodeCycle       = New(TypeConflict, 4104, "cannot move a folder into itself or its descendant")
	ErrTreeNodeNotFolder   = New(TypeInvalidArg, 4105, "target tree node is not a folder")
	ErrTreeNodeMaxDepth    = New(TypeConflict, 4107, "folder depth limit reached (max 3 levels)")
)
