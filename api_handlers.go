package main

import (
"encoding/json"
"errors"
"fmt"
"net/http"
"strconv"
)

func (p *ProfileParams) FilingAndValidate(r *http.Request) error {
p.Login = r.FormValue("login")
if p.Login == ""{
return errors.New("validate error")
}
return nil
}

func (c *CreateParams) FilingAndValidate(r *http.Request) error {
c.Login = r.FormValue("login")
if c.Login == ""{
return errors.New("validate error")
}
c.Login = r.FormValue("login")
if len(c.Login) < 10{
return errors.New("validate error")
}
c.Name = r.FormValue("name")
c.Name = r.FormValue("full_name")
c.Status = r.FormValue("status")
if c.Status == ""{
c.Status = r.FormValue("user")
}
if c.Status == ""{
c.Status = r.FormValue("moderator")
}
if c.Status == ""{
c.Status = r.FormValue("admin")
}
c.Status = r.FormValue("status")
if c.Status == ""{
c.Status = "user"
}
c.Age,_ = strconv.Atoi(r.FormValue("age"))
if c.Age < 0{
return errors.New("validate error")
}
c.Age,_ = strconv.Atoi(r.FormValue("age"))
if c.Age > 128{
return errors.New("validate error")
}
return nil
}

func (o *OtherCreateParams) FilingAndValidate(r *http.Request) error {
o.Username = r.FormValue("username")
if o.Username == ""{
return errors.New("validate error")
}
o.Username = r.FormValue("username")
if len(o.Username) < 3{
return errors.New("validate error")
}
o.Name = r.FormValue("name")
o.Name = r.FormValue("account_name")
o.Class = r.FormValue("class")
if o.Class == ""{
o.Class = r.FormValue("warrior")
}
if o.Class == ""{
o.Class = r.FormValue("sorcerer")
}
if o.Class == ""{
o.Class = r.FormValue("rouge")
}
o.Class = r.FormValue("class")
if o.Class == ""{
o.Class = "warrior"
}
o.Level,_ = strconv.Atoi(r.FormValue("level"))
if o.Level < 1{
return errors.New("validate error")
}
o.Level,_ = strconv.Atoi(r.FormValue("level"))
if o.Level > 50{
return errors.New("validate error")
}
return nil
}

func (m *MyApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		m.profile(rw, r)
	case "/user/create":
		m.create(rw, r)
	default:
		rw.WriteHeader(http.StatusNotFound)
		responseError(rw, errors.New("unknown method"))
	}
}

func (o *OtherApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		o.create(rw, r)
	default:
		rw.WriteHeader(http.StatusNotFound)
		responseError(rw, errors.New("unknown method"))
	}
}

func(m *MyApi) profile(rw http.ResponseWriter, r * http.Request) {
profileparams := ProfileParams{}
if err := profileparams.FilingAndValidate(r); err != nil {
responseError(rw,err)
return
}
response, err := m.Profile(nil,profileparams)
if err != nil {
responseError(rw, err)
return
}
responseResult(rw, err, response)
}
func(m *MyApi) create(rw http.ResponseWriter, r * http.Request) {
createparams := CreateParams{}
if err := createparams.FilingAndValidate(r); err != nil {
responseError(rw,err)
return
}
response, err := m.Create(nil,createparams)
if err != nil {
responseError(rw, err)
return
}
responseResult(rw, err, response)
}
func(o *OtherApi) create(rw http.ResponseWriter, r * http.Request) {
othercreateparams := OtherCreateParams{}
if err := othercreateparams.FilingAndValidate(r); err != nil {
responseError(rw,err)
return
}
response, err := o.Create(nil,othercreateparams)
if err != nil {
responseError(rw, err)
return
}
responseResult(rw, err, response)
}
func responseError(rw http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	type CR map[string]interface{}

	apiErr, ok := err.(ApiError)
	if ok {
		rw.WriteHeader(apiErr.HTTPStatus)
	}
	rw.WriteHeader(http.StatusInternalServerError)
	responseMap := CR{"error": err.Error()}
	response, _ := json.Marshal(responseMap)
	rw.Write(response)
}

func responseResult(rw http.ResponseWriter, err error, result interface{}) {
	type CR map[string]interface{}
	textErr := ""

	if err != nil {
		textErr = err.Error()
	}

	responseMap := CR{
		"error":    textErr,
		"response": result,
	}

	response, err := json.Marshal(responseMap)
	if err != nil {
		fmt.Println(err)
	}
	rw.Write(response)
}