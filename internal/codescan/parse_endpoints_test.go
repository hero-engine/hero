package codescan

import (
	"testing"
)

func TestExtractGoEndpoints(t *testing.T) {
	src := `package main

import "net/http"

func main() {
	api := r.Group("/api")
	api.GET("/users", listUsers)
	api.POST("/users", createUser)
	http.HandleFunc("/health", healthHandler)
	r.HandleFunc("/items/{id}", itemHandler).Methods("DELETE")
}
`
	eps := ExtractEndpoints("main.go", []byte(src), "go")

	if len(eps) != 4 {
		t.Fatalf("expected 4 endpoints, got %d: %+v", len(eps), eps)
	}

	// Group + GET
	if eps[0].Method != "GET" || eps[0].Path != "/api/users" {
		t.Errorf("ep[0] = %s %s, want GET /api/users", eps[0].Method, eps[0].Path)
	}
	if eps[1].Method != "POST" || eps[1].Path != "/api/users" {
		t.Errorf("ep[1] = %s %s, want POST /api/users", eps[1].Method, eps[1].Path)
	}
	if eps[2].Method != "ANY" || eps[2].Path != "/health" {
		t.Errorf("ep[2] = %s %s, want ANY /health", eps[2].Method, eps[2].Path)
	}
	if eps[3].Method != "DELETE" || eps[3].Path != "/api/items/{id}" {
		t.Errorf("ep[3] = %s %s, want DELETE /api/items/{id}", eps[3].Method, eps[3].Path)
	}

	for _, ep := range eps {
		if ep.Protocol != "rest" {
			t.Errorf("expected protocol rest, got %s", ep.Protocol)
		}
		if ep.File != "main.go" {
			t.Errorf("expected file main.go, got %s", ep.File)
		}
	}
}

func TestExtractJSTSEndpoints(t *testing.T) {
	src := `
const app = express();
app.get('/users', listUsers);
app.post('/users', createUser);
app.delete('/users/:id', deleteUser);
`
	eps := ExtractEndpoints("app.ts", []byte(src), "typescript")
	if len(eps) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(eps))
	}
	if eps[0].Method != "GET" || eps[0].Path != "/users" {
		t.Errorf("ep[0] = %s %s", eps[0].Method, eps[0].Path)
	}
	if eps[2].Method != "DELETE" || eps[2].Path != "/users/:id" {
		t.Errorf("ep[2] = %s %s", eps[2].Method, eps[2].Path)
	}
}

func TestExtractJSTSNextAppRouter(t *testing.T) {
	src := `export async function GET(req: Request) { return Response.json({}); }
export async function POST(req: Request) { return Response.json({}); }
`
	eps := ExtractEndpoints("app/api/users/[id]/route.ts", []byte(src), "typescript")
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(eps))
	}
	if eps[0].Path != "/api/users/[id]" {
		t.Errorf("expected path /api/users/[id], got %s", eps[0].Path)
	}
	if eps[0].Method != "GET" || eps[1].Method != "POST" {
		t.Errorf("methods: %s, %s", eps[0].Method, eps[1].Method)
	}
}

func TestExtractJSTSNestDecorators(t *testing.T) {
	src := `
@Get('/items')
async findAll() {}

@Post('/items')
async create() {}
`
	eps := ExtractEndpoints("items.controller.ts", []byte(src), "typescript")
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(eps))
	}
	if eps[0].Method != "GET" || eps[0].Path != "/items" {
		t.Errorf("ep[0] = %s %s", eps[0].Method, eps[0].Path)
	}
}

func TestExtractPythonEndpoints(t *testing.T) {
	src := `
@app.get("/users")
def list_users():
    pass

@app.post("/users")
def create_user():
    pass

@app.route("/legacy")
def legacy_handler():
    pass
`
	eps := ExtractEndpoints("app.py", []byte(src), "python")
	if len(eps) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(eps))
	}
	if eps[0].Method != "GET" || eps[0].Path != "/users" {
		t.Errorf("ep[0] = %s %s", eps[0].Method, eps[0].Path)
	}
	if eps[2].Method != "ANY" || eps[2].Path != "/legacy" {
		t.Errorf("ep[2] = %s %s", eps[2].Method, eps[2].Path)
	}
}

func TestExtractPythonDjango(t *testing.T) {
	src := `
urlpatterns = [
    path('users/', views.user_list),
    path('users/<int:pk>/', views.user_detail),
]
`
	eps := ExtractEndpoints("urls.py", []byte(src), "python")
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(eps))
	}
	if eps[0].Path != "/users/" {
		t.Errorf("ep[0].Path = %s", eps[0].Path)
	}
}

func TestExtractJavaSpringEndpoints(t *testing.T) {
	src := `
@RequestMapping("/api")
public class UserController {

    @GetMapping("/users")
    public List<User> list() {}

    @PostMapping("/users")
    public User create() {}

    @DeleteMapping("/users/{id}")
    public void delete() {}
}
`
	eps := ExtractEndpoints("UserController.java", []byte(src), "java")
	if len(eps) < 3 {
		t.Fatalf("expected at least 3 endpoints, got %d: %+v", len(eps), eps)
	}
	// Verify we detect the method-level endpoints
	methods := map[string]bool{}
	for _, ep := range eps {
		if ep.Protocol != "rest" {
			t.Errorf("expected rest protocol, got %s", ep.Protocol)
		}
		methods[ep.Method] = true
	}
	for _, m := range []string{"GET", "POST", "DELETE"} {
		if !methods[m] {
			t.Errorf("missing %s endpoint", m)
		}
	}
}

func TestExtractProtoEndpoints(t *testing.T) {
	src := `
syntax = "proto3";

service UserService {
  rpc GetUser (GetUserRequest) returns (User);
  rpc CreateUser (CreateUserRequest) returns (User);
}

service OrderService {
  rpc PlaceOrder (PlaceOrderRequest) returns (Order);
}
`
	eps := ExtractEndpoints("user.proto", []byte(src), "")
	if len(eps) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(eps))
	}
	if eps[0].Protocol != "grpc" {
		t.Errorf("expected grpc protocol, got %s", eps[0].Protocol)
	}
	if eps[0].Method != "RPC" || eps[0].Path != "UserService/GetUser" {
		t.Errorf("ep[0] = %s %s", eps[0].Method, eps[0].Path)
	}
	if eps[2].Path != "OrderService/PlaceOrder" {
		t.Errorf("ep[2].Path = %s", eps[2].Path)
	}
}

func TestExtractGraphQLEndpoints(t *testing.T) {
	src := `
type Query {
  users: [User!]!
  user(id: ID!): User
}

type Mutation {
  createUser(input: CreateUserInput!): User!
  deleteUser(id: ID!): Boolean!
}
`
	eps := ExtractEndpoints("schema.graphql", []byte(src), "")
	if len(eps) != 4 {
		t.Fatalf("expected 4 endpoints, got %d: %+v", len(eps), eps)
	}
	if eps[0].Method != "QUERY" || eps[0].Path != "users" {
		t.Errorf("ep[0] = %s %s", eps[0].Method, eps[0].Path)
	}
	if eps[2].Method != "MUTATION" || eps[2].Path != "createUser" {
		t.Errorf("ep[2] = %s %s", eps[2].Method, eps[2].Path)
	}
	if eps[0].Protocol != "graphql" {
		t.Errorf("expected graphql protocol, got %s", eps[0].Protocol)
	}
}

func TestExtractGroovyEndpoints(t *testing.T) {
	src := `
class UrlMappings {
    static mappings = {
        "/api/users"(controller: "user", action: "list")
        get "/api/items"
        post "/api/items"
    }
}
`
	eps := ExtractEndpoints("UrlMappings.groovy", []byte(src), "groovy")
	if len(eps) != 3 {
		t.Fatalf("expected 3 endpoints, got %d: %+v", len(eps), eps)
	}
}

func TestExtractEndpointsEmptyLang(t *testing.T) {
	// Non-endpoint file with empty lang should return nothing
	eps := ExtractEndpoints("readme.md", []byte("# Hello"), "")
	if len(eps) != 0 {
		t.Errorf("expected 0 endpoints, got %d", len(eps))
	}
}

func TestIsEndpointOnlyExt(t *testing.T) {
	cases := []struct {
		ext  string
		want bool
	}{
		{".proto", true},
		{".graphql", true},
		{".gql", true},
		{".go", false},
		{".txt", false},
	}
	for _, tc := range cases {
		if got := isEndpointOnlyExt(tc.ext); got != tc.want {
			t.Errorf("isEndpointOnlyExt(%q) = %v, want %v", tc.ext, got, tc.want)
		}
	}
}
