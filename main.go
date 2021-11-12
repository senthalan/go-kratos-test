package main

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/http/cookiejar"

	kratos "github.com/ory/kratos-client-go"
)

var kratosClient = NewKratosSDKForSelfHosted("http://127.0.0.1:4433")
var ctx = context.Background()

//go:embed templates
var templates embed.FS

// templateData contains data for template
type templateData struct {
	Title   string
	UI      *kratos.UiContainer
	Details string
}

func main() {
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/error", handleError)
	http.HandleFunc("/registration", handleRegister)
	http.HandleFunc("/verification", handleVerification)
	http.HandleFunc("/registered", handleRegistered)
	http.HandleFunc("/dashboard", handleDashboard)
	http.HandleFunc("/verified", handleVerified)
	log.Fatalln(http.ListenAndServe(":4455", http.DefaultServeMux))
}

// handleLogin handles kratos login flow
func handleLogin(w http.ResponseWriter, r *http.Request) {
	redirectTo := "http://127.0.0.1:4433/self-service/login/browser"

	// get flowID from url query parameters
	flowID := r.URL.Query().Get("flow")

	// if there is no flow id in url query parameters, create a new flow
	if flowID == "" {
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	// get cookie from headers
	cookie := r.Header.Get("cookie")
	// get the login flow
	loginFlow, _, err := kratosClient.V0alpha2Api.GetSelfServiceLoginFlow(ctx).Id(flowID).Cookie(cookie).Execute()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	templateData := templateData{
		Title: "Login",
		UI:    &loginFlow.Ui,
	}
	// render template index.html
	tmpl := template.Must(template.ParseFS(templates, "templates/index.html"))
	if err := tmpl.Execute(w, templateData); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

// handleLogout handles kratos logout flow
func handleLogout(w http.ResponseWriter, r *http.Request) {
	// get cookie from headers
	cookie := r.Header.Get("cookie")
	// create self-service logout flow for browser
	flow, _, err := kratosClient.V0alpha2Api.CreateSelfServiceLogoutFlowUrlForBrowsers(ctx).Cookie(cookie).Execute()
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	// redirect to logout url if session is valid
	if flow != nil {
		http.Redirect(w, r, flow.LogoutUrl, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleError handles login/registration error
func handleError(w http.ResponseWriter, r *http.Request) {
	// get url query parameters
	errorID := r.URL.Query().Get("id")
	// get error details
	errorDetails, _, err := kratosClient.V0alpha2Api.GetSelfServiceError(ctx).Id(errorID).Execute()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// marshal errorDetails to json
	errorDetailsJSON, err := json.MarshalIndent(errorDetails, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	templateData := templateData{
		Title:   "Error",
		Details: string(errorDetailsJSON),
	}
	// render template index.html
	tmpl := template.Must(template.ParseFS(templates, "templates/index.html"))
	if err := tmpl.Execute(w, templateData); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

// handleRegister handles kratos registration flow
func handleRegister(w http.ResponseWriter, r *http.Request) {
	// get flowID from url query parameters
	flowID := r.URL.Query().Get("flow")
	if flowID == "" {
		http.Redirect(w, r, "http://127.0.0.1:4433/self-service/registration/browser", http.StatusFound)
		return
	}
	// get cookie from headers
	cookie := r.Header.Get("cookie")
	// get the registration flow
	registrationFlow, _, err := kratosClient.V0alpha2Api.GetSelfServiceRegistrationFlow(ctx).Id(flowID).Cookie(cookie).Execute()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	templateData := templateData{
		Title: "Registration",
		UI:    &registrationFlow.Ui,
	}
	// render template index.html
	tmpl := template.Must(template.ParseFS(templates, "templates/index.html"))
	if err := tmpl.Execute(w, templateData); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

// handleVerification handles kratos verification flow
func handleVerification(w http.ResponseWriter, r *http.Request) {
	redirectTo := "http://127.0.0.1:4433/self-service/verification/browser"

	// get flowID from url query parameters
	flowID := r.URL.Query().Get("flow")

	// if there is no flow id in url query parameters, create a new flow
	if flowID == "" {
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	// get cookie from headers
	cookie := r.Header.Get("cookie")
	// get self-service verification flow for browser
	flow, _, err := kratosClient.V0alpha2Api.GetSelfServiceVerificationFlow(ctx).Id(flowID).Cookie(cookie).Execute()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	templateData := templateData{
		Title: "Verify your Email address",
		UI:    &flow.Ui,
	}
	// render template index.html
	tmpl := template.Must(template.ParseFS(templates, "templates/index.html"))
	if err := tmpl.Execute(w, templateData); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

// handleRegistered displays registration complete message to user
func handleRegistered(w http.ResponseWriter, r *http.Request) {
	templateData := templateData{
		Title: "Registration Complete",
	}
	// render template error.html
	tmpl := template.Must(template.ParseFS(templates, "templates/index.html"))
	if err := tmpl.Execute(w, templateData); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

// handleVerified displays verfification complete message to user
func handleVerified(w http.ResponseWriter, r *http.Request) {
	templateData := templateData{
		Title: "Verification Complete",
	}
	// render template error.html
	tmpl := template.Must(template.ParseFS(templates, "templates/index.html"))
	if err := tmpl.Execute(w, templateData); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

// handleDashboard shows dashboard
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	// get cookie from headers
	cookie := r.Header.Get("cookie")
	// get session details
	session, _, err := kratosClient.V0alpha2Api.ToSession(ctx).Cookie(cookie).Execute()
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// marshal session to json
	sessionJSON, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	templateData := templateData{
		Title:   "Session Details",
		Details: string(sessionJSON),
	}
	// render template index.html
	tmpl := template.Must(template.ParseFS(templates, "templates/index.html"))
	if err := tmpl.Execute(w, templateData); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

// NewKratosSDKForSelfHosted creates a new kratos client for self hosted server
func NewKratosSDKForSelfHosted(endpoint string) *kratos.APIClient {
	conf := kratos.NewConfiguration()
	conf.Servers = kratos.ServerConfigurations{{URL: endpoint}}
	cj, _ := cookiejar.New(nil)
	conf.HTTPClient = &http.Client{Jar: cj}
	return kratos.NewAPIClient(conf)
}

// writeError writes error to the response
func writeError(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	_, e := w.Write([]byte(err.Error()))
	if e != nil {
		log.Fatal(err)
	}
}
