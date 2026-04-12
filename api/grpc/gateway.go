package grpc

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Gateway struct {
	client *Client
	server *http.Server
}

func NewGateway(grpcAddr, httpAddr string) (*Gateway, error) {
	cc, err := grpclib.Dial(grpcAddr,
		grpclib.WithTransportCredentials(insecure.NewCredentials()),
		grpclib.WithDefaultCallOptions(grpclib.ForceCodec(jsonCodec{})),
	)
	if err != nil {
		return nil, err
	}

	g := &Gateway{
		client: NewClient(cc),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/agents", g.handleAgents)
	mux.HandleFunc("/api/v1/agents/", g.handleAgentByID)
	mux.HandleFunc("/health", g.handleHealth)

	g.server = &http.Server{Addr: httpAddr, Handler: mux}
	return g, nil
}

func (g *Gateway) ListenAndServe() error {
	return g.server.ListenAndServe()
}

func (g *Gateway) Close() error {
	if g.client != nil {
		g.client.Close()
	}
	if g.server != nil {
		return g.server.Close()
	}
	return nil
}

func (g *Gateway) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.listAgents(w, r)
	case http.MethodPost:
		g.createAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *Gateway) handleAgentByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	if id == "" {
		http.Error(w, "agent id is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		g.getAgent(w, r, id)
	case http.MethodPut:
		g.updateAgent(w, r, id)
	case http.MethodDelete:
		g.deleteAgent(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *Gateway) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (g *Gateway) listAgents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("page_size"))

	req := &ListAgentsRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   query.Get("status"),
	}

	resp, err := g.client.ListAgents(r.Context(), req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (g *Gateway) createAgent(w http.ResponseWriter, r *http.Request) {
	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := g.client.CreateAgent(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/agents/"+resp.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (g *Gateway) getAgent(w http.ResponseWriter, r *http.Request, id string) {
	resp, err := g.client.GetAgent(r.Context(), &GetAgentRequest{AgentID: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (g *Gateway) updateAgent(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.AgentID = id

	resp, err := g.client.UpdateAgent(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (g *Gateway) deleteAgent(w http.ResponseWriter, r *http.Request, id string) {
	_, err := g.client.DeleteAgent(r.Context(), &DeleteAgentRequest{AgentID: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code := grpcCodeToHTTP(st.Code())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": st.Message(),
		"code":  st.Code().String(),
	})
}

func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
